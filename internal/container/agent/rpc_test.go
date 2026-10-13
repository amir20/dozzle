package agent

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"sync"
	"testing"
	"testing/iotest"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/imagecheck"
	"github.com/amir20/dozzle/internal/notification/dispatcher"
	"github.com/amir20/dozzle/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

// newTestAgent serves svc and handler as an agent over an in-memory connection with
// the real TLS setup, and returns the hub's client for it.
func newTestAgent(t *testing.T, svc ClientService, handler NotificationConfigHandler) *Client {
	t.Helper()
	l := bufconn.Listen(bufSize)
	srv, err := NewServer(svc, certs, "test", handler)
	require.NoError(t, err)
	go srv.Serve(l)
	t.Cleanup(srv.Stop)

	rpc, err := NewClient("passthrough://bufnet", certs, grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
		return l.Dial()
	}))
	require.NoError(t, err)
	t.Cleanup(func() { rpc.Close() })
	return rpc
}

func receive[T any](t *testing.T, ctx context.Context, ch <-chan T, n int) []T {
	t.Helper()
	got := make([]T, 0, n)
	for range n {
		select {
		case v := <-ch:
			got = append(got, v)
		case <-ctx.Done():
			t.Fatalf("timed out after %d of %d", len(got), n)
		}
	}
	return got
}

// fakeService is the agent's local engine. Methods a test does not need panic on
// the nil embedded interface.
type fakeService struct {
	ClientService

	stats   []container.ContainerStat
	events  []container.ContainerEvent
	started []container.Container
	logs    []*container.LogEvent
	raw     []byte
	rawErr  error
	// rawTogether returns raw and rawErr from the same Read, which io.Reader allows.
	rawTogether bool
	histogram   container.LogHistogram
	image       imagecheck.Result
	actionErr   error

	mu      sync.Mutex
	calls   []string
	resizes [][2]uint
}

func (f *fakeService) record(call string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, call)
}

func (f *fakeService) recorded() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.calls...)
}

func (f *fakeService) FindContainer(_ context.Context, id string, _ container.ContainerLabels) (container.Container, error) {
	if id == "missing" {
		return container.Container{}, errors.New("no such container")
	}
	return container.Container{ID: id}, nil
}

func feed[T any](ctx context.Context, ch chan<- T, items []T) {
	go func() {
		for _, item := range items {
			select {
			case ch <- item:
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (f *fakeService) SubscribeStats(ctx context.Context, ch chan<- container.ContainerStat) {
	feed(ctx, ch, f.stats)
}

func (f *fakeService) SubscribeEvents(ctx context.Context, ch chan<- container.ContainerEvent) {
	feed(ctx, ch, f.events)
}

func (f *fakeService) SubscribeContainersStarted(ctx context.Context, ch chan<- container.Container) {
	feed(ctx, ch, f.started)
}

func (f *fakeService) LogsBetweenDates(_ context.Context, c container.Container, from, to time.Time, _ container.StdType) (<-chan *container.LogEvent, error) {
	f.record("logs " + c.ID + " " + from.UTC().Format(time.RFC3339) + " " + to.UTC().Format(time.RFC3339))
	ch := make(chan *container.LogEvent, len(f.logs))
	for _, e := range f.logs {
		ch <- e
	}
	close(ch)
	return ch, nil
}

func (f *fakeService) RawLogs(context.Context, container.Container, time.Time, time.Time, container.StdType) (io.ReadCloser, error) {
	if f.rawTogether {
		return io.NopCloser(&lastReadFails{data: f.raw, err: f.rawErr}), nil
	}
	if f.rawErr != nil {
		return io.NopCloser(io.MultiReader(bytes.NewReader(f.raw), iotest.ErrReader(f.rawErr))), nil
	}
	return io.NopCloser(bytes.NewReader(f.raw)), nil
}

func (f *fakeService) LogHistogram(_ context.Context, c container.Container, _, _ time.Time, width time.Duration) (container.LogHistogram, error) {
	f.record("histogram " + c.ID + " " + width.String())
	return f.histogram, nil
}

func (f *fakeService) ContainerAction(_ context.Context, c container.Container, action container.ContainerAction) error {
	f.record(string(action) + " " + c.ID)
	return f.actionErr
}

func (f *fakeService) CheckImageUpdate(_ context.Context, c container.Container, force bool) (imagecheck.Result, error) {
	if force {
		f.record("check " + c.ID + " force")
	} else {
		f.record("check " + c.ID)
	}
	return f.image, nil
}

// terminal echoes input back and records resizes, for two events.
func (f *fakeService) terminal(events container.ExecEventReader, stdout io.Writer) error {
	for range 2 {
		e, err := events.ReadEvent()
		if err != nil {
			return err
		}
		switch e.Type {
		case "userinput":
			stdout.Write([]byte("echo:" + e.Data))
		case "resize":
			f.mu.Lock()
			f.resizes = append(f.resizes, [2]uint{e.Width, e.Height})
			f.mu.Unlock()
		}
	}
	return nil
}

func (f *fakeService) Exec(_ context.Context, c container.Container, cmd []string, events container.ExecEventReader, stdout io.Writer) error {
	f.record("exec " + c.ID + " " + strings.Join(cmd, " "))
	return f.terminal(events, stdout)
}

func (f *fakeService) Attach(_ context.Context, c container.Container, events container.ExecEventReader, stdout io.Writer) error {
	f.record("attach " + c.ID)
	return f.terminal(events, stdout)
}

// recordingHandler is the agent's notification manager.
type recordingHandler struct {
	mu            sync.Mutex
	subscriptions []types.SubscriptionConfig
	dispatchers   []types.DispatcherConfig
	cloud         dispatcher.Dispatcher
	streamLogs    *bool
	cleared       bool
	stats         []types.SubscriptionStats
	err           error
}

func (h *recordingHandler) HandleNotificationConfig(subscriptions []types.SubscriptionConfig, dispatchers []types.DispatcherConfig) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.subscriptions, h.dispatchers = subscriptions, dispatchers
	return h.err
}

func (h *recordingHandler) SetCloudDispatcher(d dispatcher.Dispatcher) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cloud = d
}

func (h *recordingHandler) SetCloudStreamLogs(enabled *bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.streamLogs = enabled
}

func (h *recordingHandler) ClearCloudDispatcher() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cleared = true
}

func (h *recordingHandler) GetNotificationStats() []types.SubscriptionStats {
	return h.stats
}

func TestStreamStatsOverAgent(t *testing.T) {
	stats := []container.ContainerStat{
		{ID: "a", CPUPercent: 12.5, MemoryPercent: 40, MemoryUsage: 1024, NetworkRxTotal: 1, NetworkTxTotal: 2, DiskReadTotal: 3, DiskWriteTotal: 4},
		{ID: "b", CPUPercent: 0.5},
	}
	rpc := newTestAgent(t, &fakeService{stats: stats}, &recordingHandler{})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ch := make(chan container.ContainerStat)
	go rpc.StreamStats(ctx, ch)

	assert.Equal(t, stats, receive(t, ctx, ch, len(stats)))
}

func TestStreamEventsOverAgent(t *testing.T) {
	now := time.Date(2026, 10, 12, 9, 30, 0, 0, time.UTC)
	events := []container.ContainerEvent{
		{Name: "start", Host: "agent-1", ActorID: "abc", ActorAttributes: map[string]string{"image": "nginx"}, Time: now, Container: &wantedContainer},
		{Name: "die", Host: "agent-1", ActorID: "def", ActorAttributes: map[string]string{"exitCode": "1"}, Time: now.Add(time.Second)},
	}
	rpc := newTestAgent(t, &fakeService{events: events}, &recordingHandler{})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ch := make(chan container.ContainerEvent)
	go rpc.StreamEvents(ctx, ch)

	assert.Equal(t, events, receive(t, ctx, ch, len(events)), "the container rides along when the agent has one, and stays nil when it does not")
}

func TestStreamNewContainersOverAgent(t *testing.T) {
	rpc := newTestAgent(t, &fakeService{started: []container.Container{wantedContainer}}, &recordingHandler{})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ch := make(chan container.Container)
	go rpc.StreamNewContainers(ctx, ch)

	assert.Equal(t, []container.Container{wantedContainer}, receive(t, ctx, ch, 1))
}

// Cancelling the hub's side ends a live stream instead of leaking the goroutine
// SubscribeStats leaves running.
func TestStreamStatsEndsOnCancel(t *testing.T) {
	rpc := newTestAgent(t, &fakeService{stats: []container.ContainerStat{{ID: "a"}}}, &recordingHandler{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := make(chan container.ContainerStat)
	done := make(chan error, 1)
	go func() { done <- rpc.StreamStats(ctx, ch) }()

	waitCtx, waitCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer waitCancel()
	receive(t, waitCtx, ch, 1)
	cancel()

	select {
	case err := <-done:
		assert.ErrorIs(t, err, context.Canceled)
	case <-time.After(5 * time.Second):
		t.Fatal("stream did not end after cancel")
	}
}

func TestContainerActionOverAgent(t *testing.T) {
	svc := &fakeService{}
	rpc := newTestAgent(t, svc, &recordingHandler{})
	ctx := context.Background()

	for _, action := range []container.ContainerAction{container.Start, container.Stop, container.Restart, container.Remove} {
		require.NoError(t, rpc.ContainerAction(ctx, "abc", action))
	}
	assert.Equal(t, []string{"start abc", "stop abc", "restart abc", "remove abc"}, svc.recorded())

	assert.Error(t, rpc.ContainerAction(ctx, "abc", "pause"), "an unknown action is refused before it is sent")
	assert.Len(t, svc.recorded(), 4)

	err := rpc.ContainerAction(ctx, "missing", container.Start)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestContainerActionFailureOverAgent(t *testing.T) {
	rpc := newTestAgent(t, &fakeService{actionErr: errors.New("permission denied")}, &recordingHandler{})

	err := rpc.ContainerAction(context.Background(), "abc", container.Stop)
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.ErrorContains(t, err, "permission denied")
}

// pruningService is a fakeService whose engine can prune.
type pruningService struct {
	*fakeService
	opts container.PruneOptions
}

func (p *pruningService) PruneHost(_ context.Context, opts container.PruneOptions) (container.PruneResult, error) {
	p.opts = opts
	return container.PruneResult{SpaceReclaimed: 4096}, nil
}

func TestPruneHostOverAgent(t *testing.T) {
	svc := &pruningService{fakeService: &fakeService{}}
	rpc := newTestAgent(t, svc, &recordingHandler{})

	opts := container.PruneOptions{Images: true, Volumes: true, Containers: false, BuildCache: true}
	result, err := rpc.PruneHost(context.Background(), opts)

	require.NoError(t, err)
	assert.Equal(t, uint64(4096), result.SpaceReclaimed)
	assert.Equal(t, opts, svc.opts)
}

// An agent whose engine cannot prune answers like one too old to know the call.
func TestPruneHostUnsupportedOverAgent(t *testing.T) {
	rpc := newTestAgent(t, &fakeService{}, &recordingHandler{})

	_, err := rpc.PruneHost(context.Background(), container.PruneOptions{Images: true})
	assert.ErrorIs(t, err, errors.ErrUnsupported)
}

func TestLogsBetweenDatesOverAgent(t *testing.T) {
	logs := []*container.LogEvent{
		{Id: 1, ContainerID: "abc", Type: container.LogTypeSingle, Message: "hello", RawMessage: "hello", Timestamp: 1789424936, Level: "info", Stream: "stdout"},
		{Id: 2, ContainerID: "abc", Type: container.LogTypeComplex, Message: jsonBytesToFields([]byte(`{"msg":"boom","code":500}`)), Timestamp: 1789424937, Level: "error", Stream: "stderr"},
	}
	svc := &fakeService{logs: logs}
	rpc := newTestAgent(t, svc, &recordingHandler{})

	from := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	ch, err := rpc.LogsBetweenDates(context.Background(), "abc", from, from.Add(time.Hour), container.STDALL)
	require.NoError(t, err)

	var got []*container.LogEvent
	for e := range ch {
		got = append(got, e)
	}
	assert.Equal(t, logs, got, "the channel closes once the agent has sent everything")
	assert.Equal(t, []string{"logs abc 2026-10-01T00:00:00Z 2026-10-01T01:00:00Z"}, svc.recorded())
}

func TestStreamRawBytesOverAgent(t *testing.T) {
	// Larger than the server's 1 KiB read buffer, so it crosses several messages.
	raw := bytes.Repeat([]byte("0123456789abcdef"), 300)
	rpc := newTestAgent(t, &fakeService{raw: raw}, &recordingHandler{})

	r, err := rpc.StreamRawBytes(context.Background(), "abc", time.Time{}, time.Now(), container.STDALL)
	require.NoError(t, err)
	got, err := io.ReadAll(r)
	require.NoError(t, err)
	assert.Equal(t, raw, got)
}

func TestStreamRawBytesFailsMidStream(t *testing.T) {
	rpc := newTestAgent(t, &fakeService{raw: []byte("partial"), rawErr: errors.New("daemon went away")}, &recordingHandler{})

	r, err := rpc.StreamRawBytes(context.Background(), "abc", time.Time{}, time.Now(), container.STDALL)
	require.NoError(t, err)
	got, err := io.ReadAll(r)
	assert.Equal(t, "partial", string(got))
	assert.ErrorContains(t, err, "daemon went away", "a cut-off download must not read as complete")
}

type lastReadFails struct {
	data []byte
	err  error
}

func (r *lastReadFails) Read(p []byte) (int, error) {
	return copy(p, r.data), r.err
}

func TestStreamRawBytesKeepsDataReadWithError(t *testing.T) {
	rpc := newTestAgent(t, &fakeService{raw: []byte("last chunk"), rawErr: errors.New("daemon went away"), rawTogether: true}, &recordingHandler{})

	r, err := rpc.StreamRawBytes(context.Background(), "abc", time.Time{}, time.Now(), container.STDALL)
	require.NoError(t, err)
	got, err := io.ReadAll(r)
	assert.Equal(t, "last chunk", string(got))
	assert.ErrorContains(t, err, "daemon went away")
}

func TestStreamRawBytesStopsOnCancel(t *testing.T) {
	// More than a pipe write can hold, so the agent is still sending when the hub cancels.
	rpc := newTestAgent(t, &fakeService{raw: bytes.Repeat([]byte("x"), 1<<20)}, &recordingHandler{})

	ctx, cancel := context.WithCancel(context.Background())
	r, err := rpc.StreamRawBytes(ctx, "abc", time.Time{}, time.Now(), container.STDALL)
	require.NoError(t, err)
	cancel()

	done := make(chan error, 1)
	go func() {
		_, err := io.Copy(io.Discard, r)
		done <- err
	}()
	select {
	case err := <-done:
		assert.ErrorIs(t, err, context.Canceled)
	case <-time.After(5 * time.Second):
		t.Fatal("download kept going after the request was cancelled")
	}
}

func TestLogHistogramOverAgent(t *testing.T) {
	scanned := time.Date(2026, 10, 1, 0, 30, 0, 0, time.UTC)
	svc := &fakeService{histogram: container.LogHistogram{Total: []uint32{3, 0, 7}, Errors: []uint32{0, 0, 2}, ScannedFrom: scanned}}
	rpc := newTestAgent(t, svc, &recordingHandler{})

	from := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	h, err := rpc.LogHistogram(context.Background(), "abc", from, from.Add(3*time.Minute), time.Minute)
	require.NoError(t, err)

	assert.Equal(t, container.LogHistogram{
		Start:       from,
		Width:       time.Minute,
		Total:       []uint32{3, 0, 7},
		Errors:      []uint32{0, 0, 2},
		ScannedFrom: scanned,
	}, h, "zero Before and After stay zero")
	assert.Equal(t, []string{"histogram abc 1m0s"}, svc.recorded())
}

func TestCheckImageUpdateOverAgent(t *testing.T) {
	result := imagecheck.Result{
		Status:       imagecheck.Status("available"),
		Image:        "nginx:latest",
		LocalDigest:  "sha256:aaa",
		RemoteDigest: "sha256:bbb",
		CheckedAt:    time.Date(2026, 10, 12, 9, 0, 0, 0, time.Local),
		Reason:       "newer digest",
	}
	svc := &fakeService{image: result}
	rpc := newTestAgent(t, svc, &recordingHandler{})

	got, err := rpc.CheckImageUpdate(context.Background(), "abc", true)
	require.NoError(t, err)
	assert.Equal(t, result, got)
	assert.Equal(t, []string{"check abc force"}, svc.recorded())

	_, err = rpc.CheckImageUpdate(context.Background(), "missing", false)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

type scriptedEvents struct {
	events []*container.ExecEvent
}

func (s *scriptedEvents) ReadEvent() (*container.ExecEvent, error) {
	if len(s.events) == 0 {
		return nil, io.EOF
	}
	e := s.events[0]
	s.events = s.events[1:]
	return e, nil
}

func TestExecOverAgent(t *testing.T) {
	svc := &fakeService{}
	rpc := newTestAgent(t, svc, &recordingHandler{})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	events := &scriptedEvents{events: []*container.ExecEvent{
		{Type: "userinput", Data: "ls\n"},
		{Type: "resize", Width: 120, Height: 40},
	}}
	var stdout bytes.Buffer
	require.NoError(t, rpc.Exec(ctx, "abc", []string{"sh", "-c", "ls"}, events, &stdout))

	assert.Equal(t, "echo:ls\n", stdout.String())
	assert.Equal(t, []string{"exec abc sh -c ls"}, svc.recorded())
	svc.mu.Lock()
	defer svc.mu.Unlock()
	assert.Equal(t, [][2]uint{{120, 40}}, svc.resizes)
}

func TestContainerAttachOverAgent(t *testing.T) {
	svc := &fakeService{}
	rpc := newTestAgent(t, svc, &recordingHandler{})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	session, err := rpc.ContainerAttach(ctx, "abc")
	require.NoError(t, err)
	defer session.Writer.Close()

	_, err = session.Writer.Write([]byte("top\n"))
	require.NoError(t, err)
	require.NoError(t, session.Resize(80, 24))

	out, err := io.ReadAll(session.Reader)
	require.NoError(t, err)
	assert.Equal(t, "echo:top\n", string(out), "the reader ends when the agent closes the session")
	assert.Equal(t, []string{"attach abc"}, svc.recorded())
	svc.mu.Lock()
	defer svc.mu.Unlock()
	assert.Equal(t, [][2]uint{{80, 24}}, svc.resizes)
}

func TestUpdateNotificationConfigOverAgent(t *testing.T) {
	handler := &recordingHandler{}
	rpc := newTestAgent(t, &fakeService{}, handler)

	subscriptions := []types.SubscriptionConfig{{
		ID:                  7,
		Name:                "errors",
		Enabled:             true,
		DispatcherID:        2,
		LogExpression:       `level == "error"`,
		ContainerExpression: `name contains "api"`,
		MetricExpression:    "cpu > 90",
		EventExpression:     `name == "die"`,
		Cooldown:            300,
		SampleWindow:        60,
	}}
	dispatchers := []types.DispatcherConfig{{
		ID:       2,
		Name:     "ops",
		Type:     "webhook",
		URL:      "https://example.com/hook",
		Template: "{{ .Message }}",
		Headers:  map[string]string{"Authorization": "Bearer x"},
	}}
	require.NoError(t, rpc.UpdateNotificationConfig(context.Background(), subscriptions, dispatchers))

	handler.mu.Lock()
	defer handler.mu.Unlock()
	assert.Equal(t, subscriptions, handler.subscriptions)
	assert.Equal(t, dispatchers, handler.dispatchers)
}

func TestUpdateNotificationConfigRejectedOverAgent(t *testing.T) {
	handler := &recordingHandler{}
	rpc := newTestAgent(t, &fakeService{}, handler)

	tooMany := make([]types.SubscriptionConfig, 1001)
	err := rpc.UpdateNotificationConfig(context.Background(), tooMany, nil)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	handler.mu.Lock()
	assert.Nil(t, handler.subscriptions, "an oversized config never reaches the handler")
	handler.err = errors.New("disk full")
	handler.mu.Unlock()

	err = rpc.UpdateNotificationConfig(context.Background(), nil, nil)
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.ErrorContains(t, err, "disk full")
}

func TestUpdateCloudConfigOverAgent(t *testing.T) {
	handler := &recordingHandler{}
	rpc := newTestAgent(t, &fakeService{}, handler)

	expires := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	streamLogs := false
	require.NoError(t, rpc.UpdateCloudConfig(context.Background(), &types.CloudConfig{
		APIKey:     "key-123",
		Prefix:     "home",
		ExpiresAt:  &expires,
		StreamLogs: &streamLogs,
	}))

	handler.mu.Lock()
	d, ok := handler.cloud.(*dispatcher.CloudDispatcher)
	require.True(t, ok, "the agent builds its own cloud dispatcher from the key")
	assert.Equal(t, "key-123", d.APIKey)
	assert.Equal(t, "home", d.Prefix)
	require.NotNil(t, d.ExpiresAt)
	assert.True(t, expires.Equal(*d.ExpiresAt))
	require.NotNil(t, handler.streamLogs, "an explicit off must not read as unset")
	assert.False(t, *handler.streamLogs)
	assert.False(t, handler.cleared)
	handler.mu.Unlock()

	require.NoError(t, rpc.UpdateCloudConfig(context.Background(), nil))
	handler.mu.Lock()
	defer handler.mu.Unlock()
	assert.True(t, handler.cleared, "a nil config clears the agent's cloud dispatcher")
}

func TestGetNotificationStatsOverAgent(t *testing.T) {
	triggered := time.Date(2026, 10, 12, 8, 0, 0, 0, time.UTC)
	stats := []types.SubscriptionStats{
		{SubscriptionID: 7, TriggerCount: 3, LastTriggeredAt: &triggered, TriggeredContainerIDs: []string{"abc", "def"}},
		{SubscriptionID: 8, TriggerCount: 0, TriggeredContainerIDs: []string{"ghi"}},
	}
	rpc := newTestAgent(t, &fakeService{}, &recordingHandler{stats: stats})

	got, err := rpc.GetNotificationStats(context.Background())
	require.NoError(t, err)
	assert.Equal(t, stats, got, "a subscription that never fired keeps a nil LastTriggeredAt")
}

func TestRPCErrToErr(t *testing.T) {
	assert.NoError(t, rpcErrToErr(nil))
	assert.NoError(t, rpcErrToErr(status.Error(codes.OK, "")))

	plain := errors.New("plain")
	assert.Equal(t, plain, rpcErrToErr(plain), "a non-gRPC error passes through")

	assert.ErrorIs(t, rpcErrToErr(status.Error(codes.Unknown, "EOF")), io.EOF)
	assert.ErrorIs(t, rpcErrToErr(status.Error(codes.Canceled, "bye")), context.Canceled)
	assert.ErrorIs(t, rpcErrToErr(status.Error(codes.DeadlineExceeded, "slow")), context.DeadlineExceeded)

	unavailable := rpcErrToErr(status.Error(codes.Unavailable, "agent down"))
	assert.Equal(t, codes.Unavailable, status.Code(errors.Unwrap(unavailable)), "other codes keep the status underneath")
}

package cloud

import (
	"context"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/container"
	container_support "github.com/amir20/dozzle/internal/support/container"
	pb "github.com/amir20/dozzle/proto/cloud"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeClientService is a ClientService that delivers scripted log events.
type fakeClientService struct {
	host     container.Host
	logsCh   chan *container.LogEvent
	streamed atomic.Bool
	wait     chan struct{} // closed once StreamLogs is invoked
	once     sync.Once
}

func newFakeClientService(hostID string) *fakeClientService {
	return &fakeClientService{
		host:   container.Host{ID: hostID, Name: hostID},
		logsCh: make(chan *container.LogEvent, 16),
		wait:   make(chan struct{}),
	}
}

func (f *fakeClientService) FindContainer(_ context.Context, _ string, _ container.ContainerLabels) (container.Container, error) {
	return container.Container{}, nil
}
func (f *fakeClientService) ListContainers(_ context.Context, _ container.ContainerLabels) ([]container.Container, error) {
	return nil, nil
}
func (f *fakeClientService) Host(_ context.Context) (container.Host, error) { return f.host, nil }
func (f *fakeClientService) ContainerAction(_ context.Context, _ container.Container, _ container.ContainerAction) error {
	return nil
}
func (f *fakeClientService) UpdateContainer(_ context.Context, _ container.Container, progressCh chan<- container.UpdateProgress) (bool, error) {
	close(progressCh)
	return false, nil
}
func (f *fakeClientService) LogsBetweenDates(_ context.Context, _ container.Container, _ time.Time, _ time.Time, _ container.StdType) (<-chan *container.LogEvent, error) {
	return nil, nil
}
func (f *fakeClientService) RawLogs(_ context.Context, _ container.Container, _ time.Time, _ time.Time, _ container.StdType) (io.ReadCloser, error) {
	return nil, nil
}
func (f *fakeClientService) SubscribeStats(_ context.Context, _ chan<- container.ContainerStat)  {}
func (f *fakeClientService) SubscribeEvents(_ context.Context, _ chan<- container.ContainerEvent) {}
func (f *fakeClientService) SubscribeContainersStarted(_ context.Context, _ chan<- container.Container) {
}
func (f *fakeClientService) StreamLogs(ctx context.Context, _ container.Container, _ time.Time, _ container.StdType, events chan<- *container.LogEvent) error {
	f.once.Do(func() { close(f.wait) })
	f.streamed.Store(true)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ev, ok := <-f.logsCh:
			if !ok {
				return io.EOF
			}
			select {
			case events <- ev:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
}
func (f *fakeClientService) Attach(_ context.Context, _ container.Container, _ container.ExecEventReader, _ io.Writer) error {
	return nil
}
func (f *fakeClientService) Exec(_ context.Context, _ container.Container, _ []string, _ container.ExecEventReader, _ io.Writer) error {
	return nil
}

// fakeHostService is a LogStreamHostService driven from a map of containers.
type fakeHostService struct {
	mu         sync.Mutex
	containers []container.Container
	clients    map[string]*fakeClientService // hostID -> client
	startedCh  chan<- container.Container
}

func (f *fakeHostService) ListAllContainers(_ container.ContainerLabels) ([]container.Container, []error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]container.Container(nil), f.containers...), nil
}

func (f *fakeHostService) FindContainer(host string, id string, _ container.ContainerLabels) (*container_support.ContainerService, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	client, ok := f.clients[host]
	if !ok {
		return nil, assert.AnError
	}
	for _, c := range f.containers {
		if c.ID == id && c.Host == host {
			return container_support.NewContainerService(client, c), nil
		}
	}
	return nil, assert.AnError
}

func (f *fakeHostService) Hosts() []container.Host {
	f.mu.Lock()
	defer f.mu.Unlock()
	hosts := make([]container.Host, 0, len(f.clients))
	for id := range f.clients {
		hosts = append(hosts, container.Host{ID: id, Name: id})
	}
	return hosts
}

func (f *fakeHostService) SubscribeContainersStarted(_ context.Context, ch chan<- container.Container, _ container_support.ContainerFilter) {
	f.mu.Lock()
	f.startedCh = ch
	f.mu.Unlock()
}

func (f *fakeHostService) emitStart(c container.Container) {
	f.mu.Lock()
	f.containers = append(f.containers, c)
	ch := f.startedCh
	f.mu.Unlock()
	if ch != nil {
		ch <- c
	}
}

func collectBatches(t *testing.T, mu *sync.Mutex, out *[]*pb.LogBatch, wantEntries int, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		mu.Lock()
		total := 0
		for _, b := range *out {
			total += len(b.Entries)
		}
		mu.Unlock()
		if total >= wantEntries {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %d entries", wantEntries)
}

func TestLogStreamer_InitialSnapshotAndBatching(t *testing.T) {
	client := newFakeClientService("host-1")
	hs := &fakeHostService{
		containers: []container.Container{
			{ID: "c1", Name: "nginx", Host: "host-1", State: "running"},
		},
		clients: map[string]*fakeClientService{"host-1": client},
	}

	var sendMu sync.Mutex
	var sent []*pb.LogBatch
	send := func(resp *pb.ToolResponse) error {
		sendMu.Lock()
		defer sendMu.Unlock()
		if lb := resp.GetLogBatch(); lb != nil {
			sent = append(sent, lb)
		}
		return nil
	}

	ls := newLogStreamer(hs, nil, send)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runDone := make(chan struct{})
	go func() {
		ls.run(ctx)
		close(runDone)
	}()

	// Wait for reader to hook in.
	select {
	case <-client.wait:
	case <-time.After(2 * time.Second):
		t.Fatal("StreamLogs was never called")
	}

	ts := time.Now().UnixMilli()
	for i := 0; i < 3; i++ {
		client.logsCh <- &container.LogEvent{
			Timestamp:  ts,
			RawMessage: "hello world",
			Stream:     "stdout",
			Level:      "info",
		}
	}

	collectBatches(t, &sendMu, &sent, 3, 3*time.Second)

	sendMu.Lock()
	var allEntries []*pb.LogBatchEntry
	for _, b := range sent {
		allEntries = append(allEntries, b.Entries...)
	}
	sendMu.Unlock()

	require.GreaterOrEqual(t, len(allEntries), 3)
	e := allEntries[0]
	assert.Equal(t, "host-1", e.HostId)
	assert.Equal(t, "c1", e.ContainerId)
	assert.Equal(t, "nginx", e.ContainerName)
	assert.Equal(t, "hello world", e.Message)
	assert.Equal(t, "stdout", e.Stream)
	assert.Equal(t, "info", e.Level)
	assert.Equal(t, ts*int64(time.Millisecond), e.TimestampNs)

	cancel()
	<-runDone
}

func TestLogStreamer_NewContainerStartsReader(t *testing.T) {
	client := newFakeClientService("host-1")
	hs := &fakeHostService{
		containers: nil,
		clients:    map[string]*fakeClientService{"host-1": client},
	}

	send := func(_ *pb.ToolResponse) error { return nil }
	ls := newLogStreamer(hs, nil, send)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runDone := make(chan struct{})
	go func() {
		ls.run(ctx)
		close(runDone)
	}()

	// Wait for run() to subscribe before emitting; emitting before
	// startedCh is registered would silently no-op.
	require.Eventually(t, func() bool {
		hs.mu.Lock()
		defer hs.mu.Unlock()
		return hs.startedCh != nil
	}, 2*time.Second, 5*time.Millisecond, "subscription was never registered")

	hs.emitStart(container.Container{ID: "c-new", Name: "redis", Host: "host-1", State: "running"})

	select {
	case <-client.wait:
	case <-time.After(2 * time.Second):
		t.Fatal("reader was not started for new container")
	}

	cancel()
	<-runDone
}

func TestLogStreamer_LevelUnknownIsBlank(t *testing.T) {
	client := newFakeClientService("host-1")
	hs := &fakeHostService{
		containers: []container.Container{
			{ID: "c1", Name: "n", Host: "host-1", State: "running"},
		},
		clients: map[string]*fakeClientService{"host-1": client},
	}

	var sendMu sync.Mutex
	var sent []*pb.LogBatch
	send := func(resp *pb.ToolResponse) error {
		sendMu.Lock()
		defer sendMu.Unlock()
		if lb := resp.GetLogBatch(); lb != nil {
			sent = append(sent, lb)
		}
		return nil
	}
	ls := newLogStreamer(hs, nil, send)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go ls.run(ctx)

	<-client.wait
	client.logsCh <- &container.LogEvent{Timestamp: time.Now().UnixMilli(), RawMessage: "m", Stream: "stdout", Level: "unknown"}

	collectBatches(t, &sendMu, &sent, 1, 2*time.Second)
	sendMu.Lock()
	assert.Equal(t, "", sent[0].Entries[0].Level)
	sendMu.Unlock()
}

func TestLogStreamer_BatchFlushesOnMaxEntries(t *testing.T) {
	client := newFakeClientService("host-1")
	hs := &fakeHostService{
		containers: []container.Container{
			{ID: "c1", Name: "n", Host: "host-1", State: "running"},
		},
		clients: map[string]*fakeClientService{"host-1": client},
	}

	var sendMu sync.Mutex
	var sent []*pb.LogBatch
	send := func(resp *pb.ToolResponse) error {
		sendMu.Lock()
		defer sendMu.Unlock()
		if lb := resp.GetLogBatch(); lb != nil {
			sent = append(sent, lb)
		}
		return nil
	}
	ls := newLogStreamer(hs, nil, send)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runDone := make(chan struct{})
	go func() { ls.run(ctx); close(runDone) }()

	<-client.wait
	// Push just over the max entries cap so we flush before the 1s timer.
	ts := time.Now().UnixMilli()
	total := logBatchMaxEntries + 10
	for i := 0; i < total; i++ {
		client.logsCh <- &container.LogEvent{Timestamp: ts, RawMessage: "x", Stream: "stdout", Level: "info"}
	}

	// A full flush should happen well under 1s (we're below the timer).
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		sendMu.Lock()
		hasFullBatch := false
		for _, b := range sent {
			if len(b.Entries) >= logBatchMaxEntries {
				hasFullBatch = true
				break
			}
		}
		sendMu.Unlock()
		if hasFullBatch {
			cancel()
			<-runDone
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	<-runDone
	t.Fatal("expected a batch with >= logBatchMaxEntries entries")
}

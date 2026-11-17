package cloud

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/container"
	pb "github.com/amir20/dozzle/proto/cloud"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// resumeClientService records where the reader read from. LogsBetweenDates
// serves gap; StreamLogs serves live(from) and ends, so runReader returns.
type resumeClientService struct {
	*fakeClientService
	gap  []*container.LogEvent
	live func(from time.Time) []*container.LogEvent

	mu                         sync.Mutex
	gapCalls, streamCalls      int
	gapFrom, gapTo, streamFrom time.Time
}

func (r *resumeClientService) LogsBetweenDates(_ context.Context, _ container.Container, from, to time.Time, _ container.StdType) (<-chan *container.LogEvent, error) {
	r.mu.Lock()
	r.gapCalls++
	r.gapFrom, r.gapTo = from, to
	r.mu.Unlock()
	ch := make(chan *container.LogEvent, len(r.gap))
	for _, ev := range r.gap {
		ch <- ev
	}
	close(ch)
	return ch, nil
}

func (r *resumeClientService) StreamLogs(ctx context.Context, _ container.Container, from time.Time, _ container.StdType, events chan<- *container.LogEvent) error {
	r.mu.Lock()
	r.streamCalls++
	r.streamFrom = from
	r.mu.Unlock()
	if r.live != nil {
		for _, ev := range r.live(from) {
			select {
			case events <- ev:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
	return nil
}

func lineAt(ms int64, msg string) *container.LogEvent {
	return &container.LogEvent{Timestamp: ms, RawMessage: msg, Stream: "stdout"}
}

// runResumeReader runs one reader to completion and returns the messages sent.
func runResumeReader(t *testing.T, client *resumeClientService, resumeAfter time.Time) []string {
	t.Helper()
	var msgs []string
	send := func(resp *pb.ToolResponse) error {
		for _, e := range resp.GetLogBatch().GetEntries() {
			msgs = append(msgs, e.Message)
		}
		return nil
	}
	ls := newLogStreamer(nil, nil, send, nil)
	cs := container.NewContainerService(client, container.Container{ID: "c1", Host: "host-1"})
	ls.runReader(t.Context(), cs, 0, resumeAfter)
	return msgs
}

func TestLogStreamer_ResumeSendsGapThenLive(t *testing.T) {
	resume := time.Now().Add(-10 * time.Minute)
	resumeMs := resume.UnixMilli()
	client := &resumeClientService{
		fakeClientService: newFakeClientService("host-1"),
		gap: []*container.LogEvent{
			lineAt(resumeMs-1000, "before resume"), // k8s sinceTime is whole seconds
			lineAt(resumeMs, "at resume"),          // Cloud already stored it
			lineAt(resumeMs+1, "gap 1"),
			lineAt(resumeMs+60_000, "gap 2"),
		},
		live: func(from time.Time) []*container.LogEvent {
			return []*container.LogEvent{
				lineAt(from.UnixMilli()-10, "already in gap"), // Docker follow reaches back 50ms
				lineAt(from.UnixMilli(), "live 1"),
			}
		},
	}

	msgs := runResumeReader(t, client, resume)

	assert.Equal(t, []string{"gap 1", "gap 2", "live 1"}, msgs)
	assert.Equal(t, resume, client.gapFrom)
	assert.Equal(t, client.gapTo, client.streamFrom, "the gap ends where the live stream starts")
}

func TestLogStreamer_ResumeCappedAtOneHour(t *testing.T) {
	start := time.Now()
	client := &resumeClientService{
		fakeClientService: newFakeClientService("host-1"),
		gap: []*container.LogEvent{
			lineAt(start.Add(-2*time.Hour).UnixMilli(), "too old"),
			lineAt(start.Add(-30*time.Minute).UnixMilli(), "recent"),
		},
	}

	msgs := runResumeReader(t, client, start.Add(-3*time.Hour))

	assert.Equal(t, []string{"recent"}, msgs)
	assert.WithinDuration(t, start.Add(-logResumeMaxGap), client.gapFrom, time.Second)
}

func TestLogStreamer_NoResumeStartsFromNow(t *testing.T) {
	for name, resume := range map[string]time.Time{
		"no hint":     {},
		"future hint": time.Now().Add(time.Minute),
	} {
		t.Run(name, func(t *testing.T) {
			client := &resumeClientService{fakeClientService: newFakeClientService("host-1")}
			start := time.Now()
			runResumeReader(t, client, resume)
			assert.Zero(t, client.gapCalls)
			assert.Equal(t, 1, client.streamCalls)
			assert.WithinDuration(t, start, client.streamFrom, time.Second)
		})
	}
}

func TestLogStreamer_ResumeUsedOnce(t *testing.T) {
	client := &resumeClientService{fakeClientService: newFakeClientService("host-1")}
	c := container.Container{ID: "c1", Host: "host-1", State: "running"}
	hs := &fakeHostService{containers: []container.Container{c}}
	hs.clients = map[string]*fakeClientService{"host-1": client.fakeClientService}
	// FindContainer must hand out the recording client, not the plain fake.
	ls := newLogStreamer(&resumeHostService{fakeHostService: hs, client: client}, nil,
		func(*pb.ToolResponse) error { return nil },
		map[string]time.Time{readerKey("host-1", "c1"): time.Now().Add(-time.Minute)})

	// The reader ends on its own (StreamLogs returns), as on a container restart.
	ls.startReader(t.Context(), c)
	ls.wg.Wait()
	ls.startReader(t.Context(), c)
	ls.wg.Wait()

	assert.Equal(t, 1, client.gapCalls, "only the first reader resumes")
	assert.Equal(t, 2, client.streamCalls)
	require.Empty(t, ls.resume)
}

type resumeHostService struct {
	*fakeHostService
	client *resumeClientService
}

func (r *resumeHostService) FindContainer(_ string, _ string, _ container.ContainerLabels) (*container.ContainerService, error) {
	return container.NewContainerService(r.client, r.containers[0]), nil
}

type resumeCloud struct {
	pb.CloudToolServiceClient
	resp *pb.GetLogResumeResponse
	err  error
}

func (r resumeCloud) GetLogResume(context.Context, *pb.GetLogResumeRequest, ...grpc.CallOption) (*pb.GetLogResumeResponse, error) {
	return r.resp, r.err
}

func TestFetchLogResume(t *testing.T) {
	got := fetchLogResume(t.Context(), resumeCloud{resp: &pb.GetLogResumeResponse{Points: []*pb.LogResumePoint{
		{HostId: "h", ContainerId: "c1", AfterTsNs: 1_700_000_000_123_000_000},
		{HostId: "h", ContainerId: "c2"}, // no time: starts from now
	}}})
	assert.Equal(t, map[string]time.Time{readerKey("h", "c1"): time.Unix(0, 1_700_000_000_123_000_000)}, got)

	// A Cloud that predates the RPC.
	assert.Nil(t, fetchLogResume(t.Context(), resumeCloud{err: status.Error(codes.Unimplemented, "unknown method")}))
}

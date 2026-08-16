package docker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockedClient struct {
	mock.Mock
	container.Client
}

func (m *mockedClient) ListContainers(ctx context.Context, filter container.ContainerLabels) ([]container.Container, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]container.Container), args.Error(1)
}

func (m *mockedClient) FindContainer(ctx context.Context, id string) (container.Container, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(container.Container), args.Error(1)
}

func (m *mockedClient) ContainerEvents(ctx context.Context, events chan<- container.ContainerEvent) error {
	args := m.Called(ctx, events)
	return args.Error(0)
}

func (m *mockedClient) ContainerStats(ctx context.Context, id string, stats chan<- container.ContainerStat) error {
	args := m.Called(ctx, id, stats)
	return args.Error(0)
}

func (m *mockedClient) Host() container.Host {
	args := m.Called()
	return args.Get(0).(container.Host)
}

func startedCollector(ctx context.Context) *DockerStatsCollector {
	client := new(mockedClient)
	client.On("ListContainers", mock.Anything, mock.Anything).Return([]container.Container{
		{
			ID:    "1234",
			Name:  "test",
			State: "running",
		},
	}, nil)
	client.On("ContainerEvents", mock.Anything, mock.AnythingOfType("chan<- container.ContainerEvent")).
		Return(nil).
		Run(func(args mock.Arguments) {
			ctx := args.Get(0).(context.Context)
			<-ctx.Done()
		})
	client.On("ContainerStats", mock.Anything, mock.Anything, mock.AnythingOfType("chan<- container.ContainerStat")).
		Return(nil).
		Run(func(args mock.Arguments) {
			stats := args.Get(2).(chan<- container.ContainerStat)
			stats <- container.ContainerStat{
				ID: "1234",
			}
		})
	client.On("Host").Return(container.Host{
		ID: "localhost",
	})

	collector := NewDockerStatsCollector(client, container.ContainerLabels{})
	stats := make(chan container.ContainerStat)

	collector.Subscribe(ctx, stats)

	go collector.Start(ctx)

	<-stats

	return collector
}

func TestCancelers(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	collector := startedCollector(ctx)

	_, ok := collector.cancelers.Load("1234")
	assert.True(t, ok, "canceler should be stored")

	assert.False(t, collector.Start(ctx), "second start should return false")
	assert.Equal(t, int32(2), collector.totalStarted.Load(), "total started should be 2")

	collector.Stop()

	assert.Equal(t, int32(1), collector.totalStarted.Load(), "total started should be 1")
}

func TestSecondStart(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	collector := startedCollector(ctx)

	assert.False(t, collector.Start(ctx), "second start should return false")
	assert.Equal(t, int32(2), collector.totalStarted.Load(), "total started should be 2")

	collector.Stop()
	assert.Equal(t, int32(1), collector.totalStarted.Load(), "total started should be 1")
}

func TestStop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	collector := startedCollector(ctx)
	collector.Stop()
	assert.Equal(t, int32(0), collector.totalStarted.Load(), "total started should be 1")
}

// fastRetries shrinks one collector's backoff so a retry test finishes in
// milliseconds. It is set on the instance before Start, so it never races with
// the goroutines of collectors other tests left running.
func fastRetries(sc *DockerStatsCollector) *DockerStatsCollector {
	sc.retryMin, sc.retryMax = time.Millisecond, time.Millisecond
	return sc
}

// A container's stats stream breaking is not the container going away — that
// arrives as a `die` event. This used to be one-shot, so a single transient
// error left one container permanently absent from stats while every other
// container on the host carried on: indistinguishable from an idle container,
// and unrecoverable without restarting the process.
func TestStatsStreamRetriesAfterError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	client := new(mockedClient)
	client.On("Host").Return(container.Host{ID: "localhost"})
	client.On("ListContainers", mock.Anything, mock.Anything).Return([]container.Container{
		{ID: "1234", Name: "test", State: "running"},
	}, nil)
	client.On("ContainerEvents", mock.Anything, mock.Anything).
		Return(nil).
		Run(func(args mock.Arguments) { <-args.Get(0).(context.Context).Done() })

	// First attempt fails, every later attempt delivers.
	client.On("ContainerStats", mock.Anything, mock.Anything, mock.Anything).
		Return(errors.New("connection reset")).
		Once()
	client.On("ContainerStats", mock.Anything, mock.Anything, mock.Anything).
		Return(nil).
		Run(func(args mock.Arguments) {
			args.Get(2).(chan<- container.ContainerStat) <- container.ContainerStat{ID: "1234"}
		})

	collector := fastRetries(NewDockerStatsCollector(client, container.ContainerLabels{}))
	stats := make(chan container.ContainerStat)
	collector.Subscribe(ctx, stats)
	go collector.Start(ctx)

	select {
	case s := <-stats:
		assert.Equal(t, "1234", s.ID, "stats resumed after the stream errored")
	case <-time.After(5 * time.Second):
		t.Fatal("stats never resumed after a failed stream — the retry is gone")
	}
}

// The event stream ending used to call forceStop(), which bypasses the
// reference count: every subscriber went silent at once and none was told. A
// Cloud connection holds a permanent subscription and never resubscribes, so
// that was terminal for it. It must reconnect instead.
func TestEventStreamRetriesInsteadOfStoppingTheCollector(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	reconnected := make(chan struct{}, 1)
	client := new(mockedClient)
	client.On("Host").Return(container.Host{ID: "localhost"})
	client.On("ListContainers", mock.Anything, mock.Anything).Return([]container.Container{}, nil)
	client.On("ContainerStats", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

	client.On("ContainerEvents", mock.Anything, mock.Anything).
		Return(errors.New("docker went away")).
		Once()
	client.On("ContainerEvents", mock.Anything, mock.Anything).
		Return(nil).
		Run(func(args mock.Arguments) {
			select {
			case reconnected <- struct{}{}:
			default:
			}
			<-args.Get(0).(context.Context).Done()
		})

	collector := fastRetries(NewDockerStatsCollector(client, container.ContainerLabels{}))
	stopped := make(chan bool, 1)
	go func() { stopped <- collector.Start(ctx) }()

	select {
	case <-reconnected:
	case <-time.After(5 * time.Second):
		t.Fatal("event stream was never re-established")
	}

	// And the collector itself is still up — the old behaviour tore it down.
	select {
	case <-stopped:
		t.Fatal("collector stopped when the event stream failed")
	case <-time.After(50 * time.Millisecond):
	}
	collector.mu.Lock()
	assert.NotNil(t, collector.stopper, "collector should still be running")
	collector.mu.Unlock()
}

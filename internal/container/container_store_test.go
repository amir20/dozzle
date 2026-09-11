package container

import (
	"context"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/utils"
	"github.com/puzpuzpuz/xsync/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockedClient struct {
	mock.Mock
	Client
}

func (m *mockedClient) ListContainers(ctx context.Context, filter ContainerLabels) ([]Container, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]Container), args.Error(1)
}

func (m *mockedClient) FindContainer(ctx context.Context, id string) (Container, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(Container), args.Error(1)
}

func (m *mockedClient) ContainerEvents(ctx context.Context, events chan<- ContainerEvent) error {
	args := m.Called(ctx, events)
	return args.Error(0)
}

func (m *mockedClient) ContainerStats(ctx context.Context, id string, stats chan<- ContainerStat) error {
	args := m.Called(ctx, id, stats)
	return args.Error(0)
}

func (m *mockedClient) Host() Host {
	args := m.Called()
	return args.Get(0).(Host)
}

func TestContainerStore_List(t *testing.T) {

	client := new(mockedClient)
	client.On("ListContainers", mock.Anything, mock.Anything).Return([]Container{
		{
			ID:   "1234",
			Name: "test",
		},
	}, nil)
	client.On("ContainerEvents", mock.Anything, mock.AnythingOfType("chan<- container.ContainerEvent")).Return(nil).Run(func(args mock.Arguments) {
		ctx := args.Get(0).(context.Context)
		<-ctx.Done()
	})
	client.On("Host").Return(Host{
		ID: "localhost",
	})

	client.On("FindContainer", mock.Anything, "1234").Return(Container{
		ID:    "1234",
		Name:  "test",
		Image: "test",
		Stats: utils.NewRingBuffer[ContainerStat](300),
	}, nil)

	collector := &fakeStatsCollector{}
	store := NewContainerStore(t.Context(), client, collector, ContainerLabels{})
	containers, _ := store.ListContainers(ContainerLabels{})

	assert.Equal(t, containers[0].ID, "1234")
}

func TestContainerStore_die(t *testing.T) {
	client := new(mockedClient)
	client.On("ListContainers", mock.Anything, mock.Anything).Return([]Container{
		{
			ID:    "1234",
			Name:  "test",
			State: "running",
			Stats: utils.NewRingBuffer[ContainerStat](300),
		},
	}, nil)

	ready := make(chan struct{})
	client.On("ContainerEvents", mock.Anything, mock.AnythingOfType("chan<- container.ContainerEvent")).Return(nil).
		Run(func(args mock.Arguments) {
			ctx := args.Get(0).(context.Context)
			events := args.Get(1).(chan<- ContainerEvent)
			<-ready
			events <- ContainerEvent{
				Name:    "die",
				ActorID: "1234",
				Host:    "localhost",
			}
			<-ctx.Done()
		})
	client.On("Host").Return(Host{
		ID: "localhost",
	})

	client.On("ContainerStats", mock.Anything, "1234", mock.AnythingOfType("chan<- container.ContainerStat")).Return(nil)

	client.On("FindContainer", mock.Anything, "1234").Return(Container{
		ID:    "1234",
		Name:  "test",
		Image: "test",
		Stats: utils.NewRingBuffer[ContainerStat](300),
	}, nil)

	store := NewContainerStore(t.Context(), client, &fakeStatsCollector{}, ContainerLabels{})

	// Wait until we get the event
	events := make(chan ContainerEvent)
	store.SubscribeEvents(t.Context(), events)
	close(ready)
	<-events

	containers, _ := store.ListContainers(ContainerLabels{})
	assert.Equal(t, containers[0].State, "exited")
}

func TestContainerStore_rename(t *testing.T) {
	run := func(t *testing.T, initial Container, attributes map[string]string) Container {
		client := new(mockedClient)
		client.On("ListContainers", mock.Anything, mock.Anything).Return([]Container{initial}, nil)
		client.On("FindContainer", mock.Anything, initial.ID).Return(initial, nil)
		client.On("Host").Return(Host{ID: "localhost"})

		ready := make(chan struct{})
		client.On("ContainerEvents", mock.Anything, mock.AnythingOfType("chan<- container.ContainerEvent")).Return(nil).
			Run(func(args mock.Arguments) {
				ctx := args.Get(0).(context.Context)
				events := args.Get(1).(chan<- ContainerEvent)
				<-ready
				events <- ContainerEvent{
					Name:            "rename",
					ActorID:         initial.ID,
					Host:            "localhost",
					ActorAttributes: attributes,
				}
				<-ctx.Done()
			})

		store := NewContainerStore(t.Context(), client, &fakeStatsCollector{}, ContainerLabels{})

		events := make(chan ContainerEvent)
		store.SubscribeEvents(t.Context(), events)
		close(ready)
		<-events

		containers, err := store.ListContainers(ContainerLabels{})
		assert.NoError(t, err)
		assert.Len(t, containers, 1)
		return containers[0]
	}

	t.Run("keeps custom name from dev.dozzle.name label", func(t *testing.T) {
		initial := Container{
			ID:          "1234",
			Name:        "custom-name",
			State:       "running",
			FullyLoaded: true,
			Labels:      map[string]string{"dev.dozzle.name": "custom-name"},
			Stats:       utils.NewRingBuffer[ContainerStat](300),
		}
		result := run(t, initial, map[string]string{"name": "new-docker-name"})
		assert.Equal(t, "custom-name", result.Name)
	})

	t.Run("keeps custom name from coolify.serviceName label", func(t *testing.T) {
		initial := Container{
			ID:          "1234",
			Name:        "coolify-name",
			State:       "running",
			FullyLoaded: true,
			Labels:      map[string]string{"coolify.serviceName": "coolify-name"},
			Stats:       utils.NewRingBuffer[ContainerStat](300),
		}
		result := run(t, initial, map[string]string{"name": "new-docker-name"})
		assert.Equal(t, "coolify-name", result.Name)
	})

	t.Run("follows rename when name comes from docker", func(t *testing.T) {
		initial := Container{
			ID:          "1234",
			Name:        "old-docker-name",
			State:       "running",
			FullyLoaded: true,
			Stats:       utils.NewRingBuffer[ContainerStat](300),
		}
		result := run(t, initial, map[string]string{"name": "new-docker-name"})
		assert.Equal(t, "new-docker-name", result.Name)
	})
}

// TestContainerStore_start_inspect_failure covers a container that starts while
// FindContainer is failing (a busy daemon right after a compose recreate). Nothing
// re-adds it later, so before the list-entry fallback it stayed missing from the store
// for the life of the connection and the UI never saw it update again.
func TestContainerStore_start_inspect_failure(t *testing.T) {
	client := new(mockedClient)

	existing := Container{ID: "1234", Name: "test", State: "running", Stats: utils.NewRingBuffer[ContainerStat](300)}
	started := Container{ID: "5678", Name: "restarted", State: "running", Stats: utils.NewRingBuffer[ContainerStat](300)}

	// the initial store hydration only sees the pre-existing container
	client.On("ListContainers", mock.Anything, mock.Anything).Return([]Container{existing}, nil).Once()
	// by the time the start event lands, docker lists both
	client.On("ListContainers", mock.Anything, mock.Anything).Return([]Container{existing, started}, nil)

	client.On("FindContainer", mock.Anything, "1234").Return(existing, nil)
	// inspect fails for the container that just started
	client.On("FindContainer", mock.Anything, "5678").Return(Container{}, assert.AnError)

	client.On("Host").Return(Host{ID: "localhost"})

	ready := make(chan struct{})
	client.On("ContainerEvents", mock.Anything, mock.AnythingOfType("chan<- container.ContainerEvent")).Return(nil).
		Run(func(args mock.Arguments) {
			ctx := args.Get(0).(context.Context)
			events := args.Get(1).(chan<- ContainerEvent)
			<-ready
			events <- ContainerEvent{Name: "start", ActorID: "5678", Host: "localhost"}
			<-ctx.Done()
		})

	store := NewContainerStore(t.Context(), client, &fakeStatsCollector{}, ContainerLabels{})

	events := make(chan ContainerEvent)
	store.SubscribeEvents(t.Context(), events)
	close(ready)
	<-events

	containers, err := store.ListContainers(ContainerLabels{})
	assert.NoError(t, err)

	ids := make([]string, 0, len(containers))
	for _, c := range containers {
		ids = append(ids, c.ID)
	}
	assert.ElementsMatch(t, []string{"1234", "5678"}, ids, "started container should fall back to its list entry when inspect fails")
}

type fakeStatsCollector struct{}

func (f *fakeStatsCollector) Subscribe(_ context.Context, _ chan<- ContainerStat) {}
func (f *fakeStatsCollector) Start(_ context.Context) bool                        { return true }
func (f *fakeStatsCollector) Stop()                                               {}

// A subscriber that stops reading must cost only its own events. The store runs
// the Docker event stream, stats and every container add on one goroutine, so a
// blocking fan-out lets one wedged reader freeze the host: containers that start
// during the freeze are never added, and Docker does not replay what it sent
// while nobody was reading.
func TestContainerStore_wedgedSubscriberDoesNotStallStore(t *testing.T) {
	client := new(mockedClient)

	existing := Container{ID: "1234", Name: "test", State: "running", Stats: utils.NewRingBuffer[ContainerStat](300)}
	first := Container{ID: "5678", Name: "first", State: "running", Stats: utils.NewRingBuffer[ContainerStat](300)}
	second := Container{ID: "9012", Name: "second", State: "running", Stats: utils.NewRingBuffer[ContainerStat](300)}

	client.On("ListContainers", mock.Anything, mock.Anything).Return([]Container{existing}, nil).Once()
	client.On("ListContainers", mock.Anything, mock.Anything).Return([]Container{existing, first, second}, nil)
	client.On("FindContainer", mock.Anything, "1234").Return(existing, nil)
	client.On("FindContainer", mock.Anything, "5678").Return(first, nil)
	client.On("FindContainer", mock.Anything, "9012").Return(second, nil)
	client.On("Host").Return(Host{ID: "localhost"})

	ready := make(chan struct{})
	client.On("ContainerEvents", mock.Anything, mock.AnythingOfType("chan<- container.ContainerEvent")).Return(nil).
		Run(func(args mock.Arguments) {
			ctx := args.Get(0).(context.Context)
			events := args.Get(1).(chan<- ContainerEvent)
			<-ready
			events <- ContainerEvent{Name: "start", ActorID: "5678", Host: "localhost"}
			events <- ContainerEvent{Name: "start", ActorID: "9012", Host: "localhost"}
			<-ctx.Done()
		})

	store := NewContainerStore(t.Context(), client, &fakeStatsCollector{}, ContainerLabels{})

	// neither of these is ever read from: a handler that deadlocked, a client
	// whose socket wedged. Both contexts stay alive, so nothing unregisters them.
	store.SubscribeEvents(t.Context(), make(chan ContainerEvent))
	store.SubscribeNewContainers(t.Context(), make(chan Container))

	close(ready)

	assert.Eventually(t, func() bool {
		containers, err := store.ListContainers(ContainerLabels{})
		if err != nil {
			return false
		}
		ids := make(map[string]struct{}, len(containers))
		for _, c := range containers {
			ids[c.ID] = struct{}{}
		}
		_, hasFirst := ids["5678"]
		_, hasSecond := ids["9012"]
		return hasFirst && hasSecond
	}, 5*time.Second, 10*time.Millisecond, "containers started behind a wedged subscriber should still reach the store")
}

// The wait belongs to the fan-out, not to each subscriber: a host that has
// collected several wedged readers must not pay a full timeout per reader on
// every event.
func TestContainerStore_broadcastBudgetIsShared(t *testing.T) {
	client := new(mockedClient)
	client.On("Host").Return(Host{ID: "localhost"})

	store := &ContainerStore{
		client:      client,
		subscribers: xsync.NewMap[context.Context, *eventSubscriber](),
	}

	const wedged = 5
	for range wedged {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		store.subscribers.Store(ctx, &eventSubscriber{ch: make(chan ContainerEvent), name: "wedged"})
	}

	start := time.Now()
	store.broadcast(ContainerEvent{Name: "start", ActorID: "1234"})
	elapsed := time.Since(start)

	assert.GreaterOrEqual(t, elapsed, broadcastTimeout, "should have waited on the wedged subscribers")
	assert.Less(t, elapsed, 2*broadcastTimeout, "should have waited once, not once per subscriber")
}

// A stalled subscriber must report the stall as one throttled span. A container with a
// 5s healthcheck emits three exec events per cycle, and a line per lost event buries
// every other thing in the log.
func TestEventSubscriber_dropLoggingIsThrottled(t *testing.T) {
	sub := &eventSubscriber{ch: make(chan ContainerEvent), name: "sse-events"}
	start := time.Now()

	dropped, stalled, shouldLog := sub.recordDrop(start)
	assert.True(t, shouldLog, "the first drop should be logged")
	assert.Equal(t, 1, dropped)
	assert.Equal(t, time.Duration(0), stalled)

	for i := range 20 {
		_, _, shouldLog := sub.recordDrop(start.Add(time.Duration(i) * time.Second / 2))
		assert.False(t, shouldLog, "drops inside the interval should be silent")
	}

	dropped, stalled, shouldLog = sub.recordDrop(start.Add(dropLogInterval))
	assert.True(t, shouldLog, "a drop past the interval should be logged")
	assert.Equal(t, 21, dropped, "should report every drop since the last line, not just this one")
	assert.Equal(t, dropLogInterval, stalled, "should report how long the stall has run")
}

func TestEventSubscriber_reportsRecoveryOnce(t *testing.T) {
	sub := &eventSubscriber{ch: make(chan ContainerEvent), name: "sse-events"}
	start := time.Now()

	sub.recordDrop(start)
	sub.recordDrop(start.Add(time.Second))

	dropped, stalled, recovered := sub.recordDelivered(start.Add(2 * time.Second))
	assert.True(t, recovered)
	assert.Equal(t, 2, dropped, "recovery reports the whole stall, not just the unlogged tail")
	assert.Equal(t, 2*time.Second, stalled)

	_, _, recovered = sub.recordDelivered(start.Add(3 * time.Second))
	assert.False(t, recovered, "a subscriber that never stalled should stay quiet")
}

// The warning has to name the consumer that stalled. Without it there is no way to tell
// an SSE client that wedged from the notification listener or an agent stream.
func TestContainerStore_subscriberIsNamed(t *testing.T) {
	ctx := WithSubscriberName(t.Context(), "sse-events")
	assert.Equal(t, "sse-events", subscriberNameFrom(ctx))
	assert.Equal(t, "unnamed", subscriberNameFrom(t.Context()), "an unlabelled subscription should still be loggable")

	store := &ContainerStore{
		subscribers: xsync.NewMap[context.Context, *eventSubscriber](),
	}
	store.subscribers.Store(ctx, &eventSubscriber{ch: make(chan ContainerEvent, 1), name: subscriberNameFrom(ctx)})

	sub, ok := store.subscribers.Load(ctx)
	assert.True(t, ok, "subscriber should be registered under the context it was passed")
	assert.Equal(t, "sse-events", sub.name)
}

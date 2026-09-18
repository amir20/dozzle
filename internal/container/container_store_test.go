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

func TestContainerStore_updateCreatedToExitedBroadcastsStart(t *testing.T) {
	pending := Container{
		ID:    "default:hello-1:hello",
		Name:  "hello-1/hello",
		State: "created",
		Host:  "localhost",
		Stats: utils.NewRingBuffer[ContainerStat](300),
	}
	succeeded := pending
	succeeded.State = "exited"

	client := new(mockedClient)
	client.On("ListContainers", mock.Anything, mock.Anything).Return([]Container{pending}, nil)

	ready := make(chan struct{})
	client.On("ContainerEvents", mock.Anything, mock.AnythingOfType("chan<- container.ContainerEvent")).Return(nil).
		Run(func(args mock.Arguments) {
			ctx := args.Get(0).(context.Context)
			events := args.Get(1).(chan<- ContainerEvent)
			<-ready
			events <- ContainerEvent{
				Name:      "update",
				ActorID:   pending.ID,
				Host:      "localhost",
				Container: &succeeded,
			}
			<-ctx.Done()
		})
	client.On("Host").Return(Host{ID: "localhost"})
	client.On("ContainerStats", mock.Anything, pending.ID, mock.AnythingOfType("chan<- container.ContainerStat")).Return(nil)
	client.On("FindContainer", mock.Anything, pending.ID).Return(pending, nil)

	store := NewContainerStore(t.Context(), client, &fakeStatsCollector{}, ContainerLabels{})

	events := make(chan ContainerEvent, 2)
	store.SubscribeEvents(t.Context(), events)
	close(ready)

	assert.Equal(t, "start", (<-events).Name)

	containers, _ := store.ListContainers(ContainerLabels{})
	assert.Equal(t, "exited", containers[0].State)
}

// A k8s pod created between the store's first list and the informer's arrives only as
// an update. Before, updates for IDs the store never loaded were dropped for good.
func TestContainerStore_updateForUnknownContainerAddsIt(t *testing.T) {
	missed := Container{ID: "default:web-1:app", Name: "web-1/app", State: "running", Host: "localhost", Stats: utils.NewRingBuffer[ContainerStat](300)}

	client := new(mockedClient)
	client.On("ListContainers", mock.Anything, mock.Anything).Return([]Container{}, nil).Once()
	client.On("ListContainers", mock.Anything, mock.Anything).Return([]Container{missed}, nil)
	client.On("FindContainer", mock.Anything, missed.ID).Return(missed, nil)
	client.On("Host").Return(Host{ID: "localhost"})
	client.On("ContainerStats", mock.Anything, missed.ID, mock.AnythingOfType("chan<- container.ContainerStat")).Return(nil)

	ready := make(chan struct{})
	client.On("ContainerEvents", mock.Anything, mock.AnythingOfType("chan<- container.ContainerEvent")).Return(nil).
		Run(func(args mock.Arguments) {
			ctx := args.Get(0).(context.Context)
			events := args.Get(1).(chan<- ContainerEvent)
			<-ready
			events <- ContainerEvent{Name: "update", ActorID: missed.ID, Host: "localhost", Container: &missed}
			<-ctx.Done()
		})

	store := NewContainerStore(t.Context(), client, &fakeStatsCollector{}, ContainerLabels{})

	events := make(chan ContainerEvent, 2)
	store.SubscribeEvents(t.Context(), events)
	close(ready)

	assert.Equal(t, "start", (<-events).Name)
	containers, _ := store.ListContainers(ContainerLabels{})
	assert.Len(t, containers, 1)
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

// feedEvents makes the mocked event stream forward whatever the test sends, so a test
// can drive the store one event at a time.
func feedEvents(client *mockedClient) chan<- ContainerEvent {
	feed := make(chan ContainerEvent)
	client.On("ContainerEvents", mock.Anything, mock.AnythingOfType("chan<- container.ContainerEvent")).Return(nil).
		Run(func(args mock.Arguments) {
			ctx := args.Get(0).(context.Context)
			events := args.Get(1).(chan<- ContainerEvent)
			for {
				select {
				case e := <-feed:
					select {
					case events <- e:
					case <-ctx.Done():
						return
					}
				case <-ctx.Done():
					return
				}
			}
		})
	return feed
}

// waitForEvent reads the subscriber channel until the store has broadcast name, which
// it does only after it has finished handling the event.
func waitForEvent(t *testing.T, events <-chan ContainerEvent, name string) ContainerEvent {
	t.Helper()
	timeout := time.After(5 * time.Second)
	for {
		select {
		case e := <-events:
			if e.Name == name {
				return e
			}
		case <-timeout:
			t.Fatalf("timed out waiting for %q event", name)
			return ContainerEvent{}
		}
	}
}

func findByID(containers []Container, id string) (Container, bool) {
	for _, c := range containers {
		if c.ID == id {
			return c, true
		}
	}
	return Container{}, false
}

// captureStatsCollector hands the store's stats channel to the test. Start reports
// false so no background Clear races with a test that pushes stats itself.
type captureStatsCollector struct {
	subscribed chan chan<- ContainerStat
}

func newCaptureStatsCollector() *captureStatsCollector {
	return &captureStatsCollector{subscribed: make(chan chan<- ContainerStat, 1)}
}

func (c *captureStatsCollector) Subscribe(_ context.Context, stats chan<- ContainerStat) {
	select {
	case c.subscribed <- stats:
	default:
	}
}
func (c *captureStatsCollector) Start(_ context.Context) bool { return false }
func (c *captureStatsCollector) Stop()                        {}

func loadedContainer(id, state string) Container {
	return Container{ID: id, Name: "c-" + id, State: state, Host: "localhost", FullyLoaded: true, Stats: utils.NewRingBuffer[ContainerStat](300)}
}

func TestContainerStore_lifecycleEvents(t *testing.T) {
	client := new(mockedClient)
	client.On("ListContainers", mock.Anything, mock.Anything).Return([]Container{loadedContainer("1234", "running")}, nil)
	client.On("Host").Return(Host{ID: "localhost"})
	feed := feedEvents(client)

	store := NewContainerStore(t.Context(), client, newCaptureStatsCollector(), ContainerLabels{})
	events := make(chan ContainerEvent, 16)
	store.SubscribeEvents(t.Context(), events)

	state := func() Container {
		containers, err := store.ListContainers(ContainerLabels{})
		assert.NoError(t, err)
		c, _ := findByID(containers, "1234")
		return c
	}

	feed <- ContainerEvent{Name: "pause", ActorID: "1234"}
	waitForEvent(t, events, "pause")
	assert.Equal(t, "paused", state().State)

	feed <- ContainerEvent{Name: "unpause", ActorID: "1234"}
	waitForEvent(t, events, "unpause")
	assert.Equal(t, "running", state().State)

	feed <- ContainerEvent{Name: "health_status: healthy", ActorID: "1234"}
	waitForEvent(t, events, "health_status: healthy")
	assert.Equal(t, "healthy", state().Health)

	feed <- ContainerEvent{Name: "health_status: unhealthy", ActorID: "1234"}
	waitForEvent(t, events, "health_status: unhealthy")
	assert.Equal(t, "unhealthy", state().Health)

	feed <- ContainerEvent{Name: "destroy", ActorID: "1234"}
	waitForEvent(t, events, "destroy")
	containers, err := store.ListContainers(ContainerLabels{})
	assert.NoError(t, err)
	assert.Empty(t, containers)
}

// The store's filter can hold docker filters that are not labels, so the list decides
// membership. A create the list does not report must not reach the map, or the host
// would show containers the operator filtered out.
func TestContainerStore_createRejectedByFilter(t *testing.T) {
	filter := ContainerLabels{"com.example.team": {"payments"}}
	client := new(mockedClient)
	client.On("ListContainers", mock.Anything, mock.Anything).Return([]Container{loadedContainer("1234", "running")}, nil)
	client.On("Host").Return(Host{ID: "localhost"})
	feed := feedEvents(client)

	store := NewContainerStore(t.Context(), client, newCaptureStatsCollector(), filter)
	events := make(chan ContainerEvent, 16)
	store.SubscribeEvents(t.Context(), events)

	feed <- ContainerEvent{Name: "create", ActorID: "5678"}
	waitForEvent(t, events, "create")

	_, ok := store.containers.Load("5678")
	assert.False(t, ok, "a container outside the filter should not be added")
	client.AssertNotCalled(t, "FindContainer", mock.Anything, "5678")
}

// Without a filter the inspect alone proves the container belongs, so a start must not
// pay for a full list of every container on the host.
func TestContainerStore_startWithoutFilterSkipsList(t *testing.T) {
	client := new(mockedClient)
	client.On("ListContainers", mock.Anything, mock.Anything).Return([]Container{}, nil)
	client.On("FindContainer", mock.Anything, "5678").Return(loadedContainer("5678", "running"), nil)
	client.On("Host").Return(Host{ID: "localhost"})
	feed := feedEvents(client)

	store := NewContainerStore(t.Context(), client, newCaptureStatsCollector(), ContainerLabels{})
	events := make(chan ContainerEvent, 16)
	store.SubscribeEvents(t.Context(), events)

	feed <- ContainerEvent{Name: "start", ActorID: "5678"}
	waitForEvent(t, events, "start")

	_, ok := store.containers.Load("5678")
	assert.True(t, ok)
	client.AssertNumberOfCalls(t, "ListContainers", 1) // the initial list only
}

// Docker sends create then start for one container. Announcing both made the log
// streamer emit container-started twice and open two streams for the same container.
func TestContainerStore_createThenStartNotifiesOnce(t *testing.T) {
	client := new(mockedClient)
	client.On("ListContainers", mock.Anything, mock.Anything).Return([]Container{}, nil)
	client.On("FindContainer", mock.Anything, "5678").Return(loadedContainer("5678", "created"), nil).Once()
	client.On("FindContainer", mock.Anything, "5678").Return(loadedContainer("5678", "running"), nil)
	client.On("Host").Return(Host{ID: "localhost"})
	feed := feedEvents(client)

	store := NewContainerStore(t.Context(), client, newCaptureStatsCollector(), ContainerLabels{})
	events := make(chan ContainerEvent, 16)
	store.SubscribeEvents(t.Context(), events)
	started := make(chan Container, 4)
	store.SubscribeNewContainers(t.Context(), started)

	feed <- ContainerEvent{Name: "create", ActorID: "5678"}
	waitForEvent(t, events, "create")
	feed <- ContainerEvent{Name: "start", ActorID: "5678"}
	waitForEvent(t, events, "start")

	assert.Len(t, started, 1)
	c := <-started
	assert.Equal(t, "running", c.State)
}

// K8s sends create for a pending pod and then an update once it runs. The update is
// the only signal the pod started, so it has to be announced there.
func TestContainerStore_k8sUpdateToRunningNotifies(t *testing.T) {
	pending := loadedContainer("default:web-1:app", "created")
	running := pending
	running.State = "running"

	client := new(mockedClient)
	client.On("ListContainers", mock.Anything, mock.Anything).Return([]Container{}, nil)
	client.On("FindContainer", mock.Anything, pending.ID).Return(pending, nil)
	client.On("Host").Return(Host{ID: "localhost"})
	feed := feedEvents(client)

	store := NewContainerStore(t.Context(), client, newCaptureStatsCollector(), ContainerLabels{})
	events := make(chan ContainerEvent, 16)
	store.SubscribeEvents(t.Context(), events)
	started := make(chan Container, 4)
	store.SubscribeNewContainers(t.Context(), started)

	feed <- ContainerEvent{Name: "create", ActorID: pending.ID, Container: &pending}
	waitForEvent(t, events, "create")
	assert.Empty(t, started, "a pending pod has not started yet")

	feed <- ContainerEvent{Name: "update", ActorID: pending.ID, Container: &running}
	waitForEvent(t, events, "update")

	assert.Len(t, started, 1)
	c := <-started
	assert.Equal(t, "running", c.State)
}

func TestContainerStore_FindContainer(t *testing.T) {
	partial := Container{ID: "1234", Name: "test", State: "exited", Host: "localhost", Stats: utils.NewRingBuffer[ContainerStat](300)}
	full := partial
	full.FullyLoaded = true
	full.Image = "nginx"
	full.Stats = utils.NewRingBuffer[ContainerStat](300)
	userLabels := ContainerLabels{"team": {"a"}}

	newStore := func(t *testing.T) (*ContainerStore, *mockedClient) {
		client := new(mockedClient)
		client.On("ListContainers", mock.Anything, userLabels).Return([]Container{}, nil)
		client.On("ListContainers", mock.Anything, mock.Anything).Return([]Container{partial}, nil)
		client.On("FindContainer", mock.Anything, "1234").Return(full, nil)
		client.On("Host").Return(Host{ID: "localhost"})
		feedEvents(client)
		return NewContainerStore(t.Context(), client, newCaptureStatsCollector(), ContainerLabels{}), client
	}

	t.Run("fetches a partial container and broadcasts update", func(t *testing.T) {
		store, _ := newStore(t)
		events := make(chan ContainerEvent, 16)
		store.SubscribeEvents(t.Context(), events)

		c, err := store.FindContainer("1234", ContainerLabels{})
		assert.NoError(t, err)
		assert.True(t, c.FullyLoaded)
		assert.Equal(t, "nginx", c.Image)

		update := waitForEvent(t, events, "update")
		assert.Equal(t, "1234", update.ActorID)
		assert.False(t, update.Time.IsZero(), "update should carry a timestamp")
		assert.True(t, update.Container.FullyLoaded)
	})

	t.Run("keeps the stats history", func(t *testing.T) {
		store, _ := newStore(t)
		_, _ = store.ListContainers(ContainerLabels{})
		stored, ok := store.containers.Load("1234")
		assert.True(t, ok)
		stored.Stats.Push(ContainerStat{CPUPercent: 42})

		c, err := store.FindContainer("1234", ContainerLabels{})
		assert.NoError(t, err)
		assert.True(t, c.FullyLoaded)
		assert.Equal(t, 1, c.Stats.Len(), "a refetch should not reset the stats history")
		assert.Equal(t, 42.0, c.Stats.Data()[0].CPUPercent)
	})

	t.Run("denied by user labels", func(t *testing.T) {
		store, _ := newStore(t)
		_, err := store.FindContainer("1234", userLabels)
		assert.ErrorIs(t, err, ErrContainerNotFound)
	})

	t.Run("unknown id", func(t *testing.T) {
		store, client := newStore(t)
		_, err := store.FindContainer("nope", ContainerLabels{})
		assert.ErrorIs(t, err, ErrContainerNotFound)
		client.AssertNotCalled(t, "FindContainer", mock.Anything, "nope")
	})
}

func TestContainerStore_ListContainersWithUserLabels(t *testing.T) {
	userLabels := ContainerLabels{"team": {"a"}}
	client := new(mockedClient)
	client.On("ListContainers", mock.Anything, userLabels).Return([]Container{loadedContainer("1234", "running")}, nil)
	client.On("ListContainers", mock.Anything, mock.Anything).Return([]Container{loadedContainer("1234", "running"), loadedContainer("5678", "running")}, nil)
	client.On("Host").Return(Host{ID: "localhost"})
	feedEvents(client)

	store := NewContainerStore(t.Context(), client, newCaptureStatsCollector(), ContainerLabels{})

	all, err := store.ListContainers(ContainerLabels{})
	assert.NoError(t, err)
	assert.Len(t, all, 2)

	visible, err := store.ListContainers(userLabels)
	assert.NoError(t, err)
	assert.Len(t, visible, 1)
	assert.Equal(t, "1234", visible[0].ID)
}

// A failed first list used to leave the store empty until the Docker event stream
// happened to drop, because the stream was already marked connected.
func TestContainerStore_initialListFailureIsRetried(t *testing.T) {
	client := new(mockedClient)
	client.On("ListContainers", mock.Anything, mock.Anything).Return([]Container(nil), assert.AnError).Once()
	client.On("ListContainers", mock.Anything, mock.Anything).Return([]Container{loadedContainer("1234", "running")}, nil)
	client.On("Host").Return(Host{ID: "localhost"})
	feedEvents(client)

	store := NewContainerStore(t.Context(), client, newCaptureStatsCollector(), ContainerLabels{})

	containers, err := store.ListContainers(ContainerLabels{})
	assert.NoError(t, err)
	assert.Len(t, containers, 1)

	// fresh now, so the next call does not list again
	_, err = store.ListContainers(ContainerLabels{})
	assert.NoError(t, err)
	client.AssertNumberOfCalls(t, "ListContainers", 2)
}

func TestContainerStore_statsArePushedToTheirContainer(t *testing.T) {
	client := new(mockedClient)
	client.On("ListContainers", mock.Anything, mock.Anything).Return([]Container{loadedContainer("1234", "running")}, nil)
	client.On("Host").Return(Host{ID: "localhost"})
	feedEvents(client)

	collector := newCaptureStatsCollector()
	store := NewContainerStore(t.Context(), client, collector, ContainerLabels{})
	_, _ = store.ListContainers(ContainerLabels{})

	var stats chan<- ContainerStat
	select {
	case stats = <-collector.subscribed:
	case <-time.After(5 * time.Second):
		t.Fatal("store never subscribed to stats")
	}

	stats <- ContainerStat{ID: "unknown", CPUPercent: 1}
	stats <- ContainerStat{ID: "1234", CPUPercent: 7}

	c, _ := store.containers.Load("1234")
	assert.Eventually(t, func() bool { return c.Stats.Len() == 1 }, 5*time.Second, 5*time.Millisecond)
	assert.Equal(t, 7.0, c.Stats.Data()[0].CPUPercent)
	_, ok := store.containers.Load("unknown")
	assert.False(t, ok, "a stat for an unknown container should not create one")
}

func TestContainerStore_cancelledSubscriberIsRemoved(t *testing.T) {
	client := new(mockedClient)
	client.On("ListContainers", mock.Anything, mock.Anything).Return([]Container{}, nil)
	client.On("Host").Return(Host{ID: "localhost"})
	feedEvents(client)

	store := NewContainerStore(t.Context(), client, newCaptureStatsCollector(), ContainerLabels{})
	ctx, cancel := context.WithCancel(t.Context())
	store.SubscribeEvents(ctx, make(chan ContainerEvent, 1))
	assert.Equal(t, 1, store.subscribers.Size())

	cancel()
	assert.Eventually(t, func() bool { return store.subscribers.Size() == 0 }, 5*time.Second, 5*time.Millisecond)
}

func TestContainerStore_applyMountStats(t *testing.T) {
	client := new(mockedClient)
	client.On("Host").Return(Host{ID: "localhost"})
	store := &ContainerStore{
		client:      client,
		containers:  xsync.NewMap[string, *Container](),
		subscribers: xsync.NewMap[context.Context, *eventSubscriber](),
	}
	c := loadedContainer("1234", "running")
	store.containers.Store(c.ID, &c)
	events := make(chan ContainerEvent, 1)
	store.subscribers.Store(t.Context(), &eventSubscriber{ch: events, name: "test"})

	mounts := map[string]MountStat{"/data": {Destination: "/data", Available: true, Total: 100, Free: 40, Used: 60}}
	store.applyMountStats("1234", mounts)

	stored, _ := store.containers.Load("1234")
	assert.Equal(t, mounts, stored.MountStats)
	assert.Nil(t, c.MountStats, "the previous entry should not be mutated")

	update := <-events
	assert.Equal(t, "update", update.Name)
	assert.Equal(t, mounts, update.Container.MountStats)

	store.applyMountStats("unknown", mounts)
	assert.Empty(t, events, "an unknown container should not broadcast")
}

func TestMatchesLabels(t *testing.T) {
	tests := []struct {
		name   string
		labels map[string]string
		filter ContainerLabels
		want   bool
	}{
		{"empty filter", map[string]string{"a": "1"}, ContainerLabels{}, true},
		{"nil labels, empty filter", nil, ContainerLabels{}, true},
		{"match", map[string]string{"a": "1"}, ContainerLabels{"a": {"1"}}, true},
		{"one of several values", map[string]string{"a": "2"}, ContainerLabels{"a": {"1", "2"}}, true},
		{"wrong value", map[string]string{"a": "3"}, ContainerLabels{"a": {"1", "2"}}, false},
		{"missing key", map[string]string{"b": "1"}, ContainerLabels{"a": {"1"}}, false},
		{"all keys required", map[string]string{"a": "1"}, ContainerLabels{"a": {"1"}, "b": {"2"}}, false},
		{"all keys match", map[string]string{"a": "1", "b": "2"}, ContainerLabels{"a": {"1"}, "b": {"2"}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, matchesLabels(tt.labels, tt.filter))
		})
	}
}

// A refresh must not wipe a container the event loop added while the list was in
// flight, and must not keep one the list no longer reports.
func TestContainerStore_refreshReconcilesWithoutClearing(t *testing.T) {
	client := new(mockedClient)
	client.On("Host").Return(Host{ID: "localhost"})
	store := &ContainerStore{
		client:     client,
		containers: xsync.NewMap[string, *Container](),
		ctx:        t.Context(),
	}
	old := loadedContainer("old", "running")
	kept := loadedContainer("kept", "running")
	kept.Stats.Push(ContainerStat{CPUPercent: 3})
	store.containers.Store(old.ID, &old)
	store.containers.Store(kept.ID, &kept)

	client.On("ListContainers", mock.Anything, mock.Anything).Return([]Container{loadedContainer("kept", "running")}, nil).Run(func(mock.Arguments) {
		added := loadedContainer("added", "running")
		store.containers.Store(added.ID, &added)
	})

	assert.NoError(t, store.refresh())

	_, ok := store.containers.Load("old")
	assert.False(t, ok, "a container the list no longer reports should be removed")
	_, ok = store.containers.Load("added")
	assert.True(t, ok, "a container added during the list should survive")
	c, ok := store.containers.Load("kept")
	assert.True(t, ok)
	assert.Equal(t, 1, c.Stats.Len(), "the refresh should keep the stats history")
}

func TestContainerStore_mergeFetched(t *testing.T) {
	newStore := func() *ContainerStore {
		return &ContainerStore{containers: xsync.NewMap[string, *Container]()}
	}
	fetched := loadedContainer("1234", "running")
	fetched.Image = "nginx"

	t.Run("keeps fields the event loop changed during the fetch", func(t *testing.T) {
		store := newStore()
		prev := &Container{ID: "1234", State: "running", Stats: utils.NewRingBuffer[ContainerStat](300)}
		store.containers.Store("1234", prev)
		changed := *prev
		changed.State = "exited"
		changed.Health = "unhealthy"
		store.containers.Store("1234", &changed)

		c, found, updated := store.mergeFetched(prev, fetched)
		assert.True(t, found)
		assert.True(t, updated)
		assert.Equal(t, "exited", c.State)
		assert.Equal(t, "unhealthy", c.Health)
		assert.Equal(t, "nginx", c.Image)
		assert.Same(t, prev.Stats, c.Stats)
	})

	t.Run("keeps a fully loaded entry stored during the fetch", func(t *testing.T) {
		store := newStore()
		prev := &Container{ID: "1234", State: "running"}
		store.containers.Store("1234", prev)
		newer := loadedContainer("1234", "paused")
		store.containers.Store("1234", &newer)

		c, found, updated := store.mergeFetched(prev, fetched)
		assert.True(t, found)
		assert.False(t, updated)
		assert.Equal(t, "paused", c.State)
	})

	t.Run("container removed during the fetch", func(t *testing.T) {
		store := newStore()
		prev := &Container{ID: "1234"}
		_, found, updated := store.mergeFetched(prev, fetched)
		assert.False(t, found)
		assert.False(t, updated)
		_, ok := store.containers.Load("1234")
		assert.False(t, ok)
	})
}

// docker often starts a container before the loop handles its create, so the
// create's inspect already sees it running and the start that follows sees the
// same run. It must be announced once, or each announcement starts a log stream.
func TestContainerStore_createSeenRunningThenStartNotifiesOnce(t *testing.T) {
	running := loadedContainer("5678", "running")
	running.StartedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	client := new(mockedClient)
	client.On("ListContainers", mock.Anything, mock.Anything).Return([]Container{}, nil)
	client.On("FindContainer", mock.Anything, "5678").Return(running, nil)
	client.On("Host").Return(Host{ID: "localhost"})
	feed := feedEvents(client)

	store := NewContainerStore(t.Context(), client, newCaptureStatsCollector(), ContainerLabels{})
	events := make(chan ContainerEvent, 16)
	store.SubscribeEvents(t.Context(), events)
	started := make(chan Container, 4)
	store.SubscribeNewContainers(t.Context(), started)

	feed <- ContainerEvent{Name: "create", ActorID: "5678"}
	waitForEvent(t, events, "create")
	feed <- ContainerEvent{Name: "start", ActorID: "5678"}
	waitForEvent(t, events, "start")
	assert.Len(t, started, 1)

	// a restart is a new run and is announced again
	<-started
	restarted := running
	restarted.StartedAt = running.StartedAt.Add(time.Minute)
	client.ExpectedCalls = nil
	client.On("FindContainer", mock.Anything, "5678").Return(restarted, nil)
	client.On("Host").Return(Host{ID: "localhost"})
	feed <- ContainerEvent{Name: "start", ActorID: "5678"}
	waitForEvent(t, events, "start")
	assert.Len(t, started, 1)
}

// A refresh's list can be taken before a die the event loop has since applied.
// Writing the list entry back unconditionally would resurrect the container.
func TestContainerStore_storeListedKeepsNewerLoopState(t *testing.T) {
	store := &ContainerStore{containers: xsync.NewMap[string, *Container]()}

	before := loadedContainer("1", "running")
	store.containers.Store("1", &before)
	died := before
	died.State = "exited"
	store.containers.Store("1", &died)
	stat := ContainerStat{CPUPercent: 5}
	died.Stats.Push(stat)

	listed := Container{ID: "1", Name: "c-1", State: "running", Stats: utils.NewRingBuffer[ContainerStat](300)}
	store.storeListed(&before, listed)
	got, _ := store.containers.Load("1")
	assert.Equal(t, "exited", got.State, "the die applied during the list is newer")
	assert.Equal(t, 1, got.Stats.Len(), "stats history is kept")

	// added by the loop while the list was in flight: its inspect is newer
	added := loadedContainer("2", "paused")
	store.containers.Store("2", &added)
	store.storeListed(nil, Container{ID: "2", State: "running"})
	got, _ = store.containers.Load("2")
	assert.Same(t, &added, got)

	// untouched during the list: the list wins
	unchanged := loadedContainer("3", "running")
	store.containers.Store("3", &unchanged)
	store.storeListed(&unchanged, Container{ID: "3", State: "exited"})
	got, _ = store.containers.Load("3")
	assert.Equal(t, "exited", got.State)
}

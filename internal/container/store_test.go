package container

import (
	"context"
	"errors"
	"slices"
	"sync"
	"sync/atomic"
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

func TestStore_List(t *testing.T) {

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
	store := NewStore(t.Context(), client, collector, ContainerLabels{})
	containers, _ := store.ListContainers(t.Context(), ContainerLabels{})

	assert.Equal(t, containers[0].ID, "1234")
}

func TestStore_die(t *testing.T) {
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

	store := NewStore(t.Context(), client, &fakeStatsCollector{}, ContainerLabels{})

	// Wait until we get the event
	events := make(chan ContainerEvent)
	store.SubscribeEvents(t.Context(), events)
	close(ready)
	<-events

	containers, _ := store.ListContainers(t.Context(), ContainerLabels{})
	assert.Equal(t, containers[0].State, "exited")
}

func TestStore_updateCreatedToExitedBroadcastsStart(t *testing.T) {
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

	store := NewStore(t.Context(), client, &fakeStatsCollector{}, ContainerLabels{})

	events := make(chan ContainerEvent, 2)
	store.SubscribeEvents(t.Context(), events)
	close(ready)

	assert.Equal(t, "start", (<-events).Name)

	containers, _ := store.ListContainers(t.Context(), ContainerLabels{})
	assert.Equal(t, "exited", containers[0].State)
}

// A k8s pod created between the store's first list and the informer's arrives only as
// an update. Before, updates for IDs the store never loaded were dropped for good.
func TestStore_updateForUnknownContainerAddsIt(t *testing.T) {
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

	store := NewStore(t.Context(), client, &fakeStatsCollector{}, ContainerLabels{})

	events := make(chan ContainerEvent, 2)
	store.SubscribeEvents(t.Context(), events)
	close(ready)

	assert.Equal(t, "start", (<-events).Name)
	containers, _ := store.ListContainers(t.Context(), ContainerLabels{})
	assert.Len(t, containers, 1)
}

func TestStore_rename(t *testing.T) {
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

		store := NewStore(t.Context(), client, &fakeStatsCollector{}, ContainerLabels{})

		events := make(chan ContainerEvent)
		store.SubscribeEvents(t.Context(), events)
		close(ready)
		<-events

		containers, err := store.ListContainers(t.Context(), ContainerLabels{})
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

// TestStore_start_inspect_failure covers a container that starts while
// FindContainer is failing (a busy daemon right after a compose recreate). Nothing
// re-adds it later, so before the list-entry fallback it stayed missing from the store
// for the life of the connection and the UI never saw it update again.
func TestStore_start_inspect_failure(t *testing.T) {
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

	store := NewStore(t.Context(), client, &fakeStatsCollector{}, ContainerLabels{})

	events := make(chan ContainerEvent)
	store.SubscribeEvents(t.Context(), events)
	close(ready)
	<-events

	containers, err := store.ListContainers(t.Context(), ContainerLabels{})
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
func TestStore_wedgedSubscriberDoesNotStallStore(t *testing.T) {
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

	store := NewStore(t.Context(), client, &fakeStatsCollector{}, ContainerLabels{})

	// neither of these is ever read from: a handler that deadlocked, a client
	// whose socket wedged. Both contexts stay alive, so nothing unregisters them.
	store.SubscribeEvents(t.Context(), make(chan ContainerEvent))
	store.SubscribeNewContainers(t.Context(), make(chan Container))

	close(ready)

	assert.Eventually(t, func() bool {
		containers, err := store.ListContainers(t.Context(), ContainerLabels{})
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
func TestStore_broadcastBudgetIsShared(t *testing.T) {
	client := new(mockedClient)
	client.On("Host").Return(Host{ID: "localhost"})

	store := &Store{
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
func TestStore_subscriberIsNamed(t *testing.T) {
	ctx := WithSubscriberName(t.Context(), "sse-events")
	assert.Equal(t, "sse-events", subscriberNameFrom(ctx))
	assert.Equal(t, "unnamed", subscriberNameFrom(t.Context()), "an unlabelled subscription should still be loggable")

	store := &Store{
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

func TestStore_lifecycleEvents(t *testing.T) {
	client := new(mockedClient)
	client.On("ListContainers", mock.Anything, mock.Anything).Return([]Container{loadedContainer("1234", "running")}, nil)
	client.On("Host").Return(Host{ID: "localhost"})
	feed := feedEvents(client)

	store := NewStore(t.Context(), client, newCaptureStatsCollector(), ContainerLabels{})
	events := make(chan ContainerEvent, 16)
	store.SubscribeEvents(t.Context(), events)

	state := func() Container {
		containers, err := store.ListContainers(t.Context(), ContainerLabels{})
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
	containers, err := store.ListContainers(t.Context(), ContainerLabels{})
	assert.NoError(t, err)
	assert.Empty(t, containers)
}

// The store's filter can hold docker filters that are not labels, so the list decides
// membership. A create the list does not report must not reach the map, or the host
// would show containers the operator filtered out.
func TestStore_createRejectedByFilter(t *testing.T) {
	filter := ContainerLabels{"com.example.team": {"payments"}}
	client := new(mockedClient)
	client.On("ListContainers", mock.Anything, mock.Anything).Return([]Container{loadedContainer("1234", "running")}, nil)
	client.On("Host").Return(Host{ID: "localhost"})
	feed := feedEvents(client)

	store := NewStore(t.Context(), client, newCaptureStatsCollector(), filter)
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
func TestStore_startWithoutFilterSkipsList(t *testing.T) {
	client := new(mockedClient)
	client.On("ListContainers", mock.Anything, mock.Anything).Return([]Container{}, nil)
	client.On("FindContainer", mock.Anything, "5678").Return(loadedContainer("5678", "running"), nil)
	client.On("Host").Return(Host{ID: "localhost"})
	feed := feedEvents(client)

	store := NewStore(t.Context(), client, newCaptureStatsCollector(), ContainerLabels{})
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
func TestStore_createThenStartNotifiesOnce(t *testing.T) {
	client := new(mockedClient)
	client.On("ListContainers", mock.Anything, mock.Anything).Return([]Container{}, nil)
	client.On("FindContainer", mock.Anything, "5678").Return(loadedContainer("5678", "created"), nil).Once()
	client.On("FindContainer", mock.Anything, "5678").Return(loadedContainer("5678", "running"), nil)
	client.On("Host").Return(Host{ID: "localhost"})
	feed := feedEvents(client)

	store := NewStore(t.Context(), client, newCaptureStatsCollector(), ContainerLabels{})
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
func TestStore_k8sUpdateToRunningNotifies(t *testing.T) {
	// exited: a short Job that finished before a Running update was seen.
	// restarting: crashed on boot, and the first update seen is already CrashLoopBackOff.
	for _, state := range []string{"running", "exited", "restarting"} {
		t.Run(state, func(t *testing.T) { testK8sUpdateNotifies(t, state) })
	}
}

func testK8sUpdateNotifies(t *testing.T, state string) {
	pending := loadedContainer("default:web-1:app", "created")
	running := pending
	running.State = state

	client := new(mockedClient)
	client.On("ListContainers", mock.Anything, mock.Anything).Return([]Container{}, nil)
	client.On("FindContainer", mock.Anything, pending.ID).Return(pending, nil)
	client.On("Host").Return(Host{ID: "localhost"})
	feed := feedEvents(client)

	store := NewStore(t.Context(), client, newCaptureStatsCollector(), ContainerLabels{})
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
	assert.Equal(t, state, c.State)
}

func TestStore_FindContainer(t *testing.T) {
	partial := Container{ID: "1234", Name: "test", State: "exited", Host: "localhost", Stats: utils.NewRingBuffer[ContainerStat](300)}
	full := partial
	full.FullyLoaded = true
	full.Image = "nginx"
	full.Stats = utils.NewRingBuffer[ContainerStat](300)
	userLabels := ContainerLabels{"team": {"a"}}

	newStore := func(t *testing.T) (*Store, *mockedClient) {
		client := new(mockedClient)
		client.On("ListContainers", mock.Anything, userLabels).Return([]Container{}, nil)
		client.On("ListContainers", mock.Anything, mock.Anything).Return([]Container{partial}, nil)
		client.On("FindContainer", mock.Anything, "1234").Return(full, nil)
		client.On("Host").Return(Host{ID: "localhost"})
		feedEvents(client)
		return NewStore(t.Context(), client, newCaptureStatsCollector(), ContainerLabels{}), client
	}

	t.Run("fetches a partial container and broadcasts update", func(t *testing.T) {
		store, _ := newStore(t)
		events := make(chan ContainerEvent, 16)
		store.SubscribeEvents(t.Context(), events)

		c, err := store.FindContainer(t.Context(), "1234", ContainerLabels{})
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
		_, _ = store.ListContainers(t.Context(), ContainerLabels{})
		stored, ok := store.containers.Load("1234")
		assert.True(t, ok)
		stored.Stats.Push(ContainerStat{CPUPercent: 42})

		c, err := store.FindContainer(t.Context(), "1234", ContainerLabels{})
		assert.NoError(t, err)
		assert.True(t, c.FullyLoaded)
		assert.Equal(t, 1, c.Stats.Len(), "a refetch should not reset the stats history")
		assert.Equal(t, 42.0, c.Stats.Data()[0].CPUPercent)
	})

	t.Run("denied by user labels", func(t *testing.T) {
		store, _ := newStore(t)
		_, err := store.FindContainer(t.Context(), "1234", userLabels)
		assert.ErrorIs(t, err, ErrContainerNotFound)
	})

	t.Run("unknown id", func(t *testing.T) {
		store, client := newStore(t)
		_, err := store.FindContainer(t.Context(), "nope", ContainerLabels{})
		assert.ErrorIs(t, err, ErrContainerNotFound)
		client.AssertNotCalled(t, "FindContainer", mock.Anything, "nope")
	})
}

func TestStore_ListContainersWithUserLabels(t *testing.T) {
	userLabels := ContainerLabels{"team": {"a"}}
	client := new(mockedClient)
	client.On("ListContainers", mock.Anything, userLabels).Return([]Container{loadedContainer("1234", "running")}, nil)
	client.On("ListContainers", mock.Anything, mock.Anything).Return([]Container{loadedContainer("1234", "running"), loadedContainer("5678", "running")}, nil)
	client.On("Host").Return(Host{ID: "localhost"})
	feedEvents(client)

	store := NewStore(t.Context(), client, newCaptureStatsCollector(), ContainerLabels{})

	all, err := store.ListContainers(t.Context(), ContainerLabels{})
	assert.NoError(t, err)
	assert.Len(t, all, 2)

	visible, err := store.ListContainers(t.Context(), userLabels)
	assert.NoError(t, err)
	assert.Len(t, visible, 1)
	assert.Equal(t, "1234", visible[0].ID)
}

// A failed first list used to leave the store empty until the Docker event stream
// happened to drop, because the stream was already marked connected.
func TestStore_initialListFailureIsRetried(t *testing.T) {
	client := new(mockedClient)
	client.On("ListContainers", mock.Anything, mock.Anything).Return([]Container(nil), assert.AnError).Once()
	client.On("ListContainers", mock.Anything, mock.Anything).Return([]Container{loadedContainer("1234", "running")}, nil)
	client.On("Host").Return(Host{ID: "localhost"})
	feedEvents(client)

	store := NewStore(t.Context(), client, newCaptureStatsCollector(), ContainerLabels{})

	containers, err := store.ListContainers(t.Context(), ContainerLabels{})
	assert.NoError(t, err)
	assert.Len(t, containers, 1)

	// fresh now, so the next call does not list again
	_, err = store.ListContainers(t.Context(), ContainerLabels{})
	assert.NoError(t, err)
	client.AssertNumberOfCalls(t, "ListContainers", 2)
}

func TestStore_statsArePushedToTheirContainer(t *testing.T) {
	client := new(mockedClient)
	client.On("ListContainers", mock.Anything, mock.Anything).Return([]Container{loadedContainer("1234", "running")}, nil)
	client.On("Host").Return(Host{ID: "localhost"})
	feedEvents(client)

	collector := newCaptureStatsCollector()
	store := NewStore(t.Context(), client, collector, ContainerLabels{})
	_, _ = store.ListContainers(t.Context(), ContainerLabels{})

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

func TestStore_cancelledSubscriberIsRemoved(t *testing.T) {
	client := new(mockedClient)
	client.On("ListContainers", mock.Anything, mock.Anything).Return([]Container{}, nil)
	client.On("Host").Return(Host{ID: "localhost"})
	feedEvents(client)

	store := NewStore(t.Context(), client, newCaptureStatsCollector(), ContainerLabels{})
	ctx, cancel := context.WithCancel(t.Context())
	store.SubscribeEvents(ctx, make(chan ContainerEvent, 1))
	assert.Equal(t, 1, store.subscribers.Size())

	cancel()
	assert.Eventually(t, func() bool { return store.subscribers.Size() == 0 }, 5*time.Second, 5*time.Millisecond)
}

func TestStore_applyMountStats(t *testing.T) {
	client := new(mockedClient)
	client.On("Host").Return(Host{ID: "localhost"})
	store := &Store{
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
func TestStore_refreshReconcilesWithoutClearing(t *testing.T) {
	client := new(mockedClient)
	client.On("Host").Return(Host{ID: "localhost"})
	store := &Store{
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

	_, err := store.refresh(true)
	assert.NoError(t, err)

	_, ok := store.containers.Load("old")
	assert.False(t, ok, "a container the list no longer reports should be removed")
	_, ok = store.containers.Load("added")
	assert.True(t, ok, "a container added during the list should survive")
	c, ok := store.containers.Load("kept")
	assert.True(t, ok)
	assert.Equal(t, 1, c.Stats.Len(), "the refresh should keep the stats history")
}

func TestStore_mergeFetched(t *testing.T) {
	newStore := func() *Store {
		return &Store{containers: xsync.NewMap[string, *Container]()}
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
func TestStore_createSeenRunningThenStartNotifiesOnce(t *testing.T) {
	running := loadedContainer("5678", "running")
	running.StartedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	client := new(mockedClient)
	client.On("ListContainers", mock.Anything, mock.Anything).Return([]Container{}, nil)
	client.On("FindContainer", mock.Anything, "5678").Return(running, nil)
	client.On("Host").Return(Host{ID: "localhost"})
	feed := feedEvents(client)

	store := NewStore(t.Context(), client, newCaptureStatsCollector(), ContainerLabels{})
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
func TestStore_storeListedKeepsNewerLoopState(t *testing.T) {
	store := &Store{containers: xsync.NewMap[string, *Container]()}

	before := loadedContainer("1", "running")
	store.containers.Store("1", &before)
	died := before
	died.State = "exited"
	store.containers.Store("1", &died)
	stat := ContainerStat{CPUPercent: 5}
	died.Stats.Push(stat)

	listed := Container{ID: "1", Name: "c-1", State: "running", Stats: utils.NewRingBuffer[ContainerStat](300)}
	store.storeListed(&before, listed, true)
	got, _ := store.containers.Load("1")
	assert.Equal(t, "exited", got.State, "the die applied during the list is newer")
	assert.Equal(t, 1, got.Stats.Len(), "stats history is kept")

	// added by the loop while the list was in flight: its inspect is newer
	added := loadedContainer("2", "paused")
	store.containers.Store("2", &added)
	store.storeListed(nil, Container{ID: "2", State: "running"}, true)
	got, _ = store.containers.Load("2")
	assert.Same(t, &added, got)

	// untouched during the list: the list wins
	unchanged := loadedContainer("3", "running")
	store.containers.Store("3", &unchanged)
	store.storeListed(&unchanged, Container{ID: "3", State: "exited"}, true)
	got, _ = store.containers.Load("3")
	assert.Equal(t, "exited", got.State)
}

// A refresh removes what the list no longer reports, but only the entry its snapshot
// saw. One the loop re-stored during the list came back after the list was taken.
func TestStore_refreshKeepsEntryReplacedDuringList(t *testing.T) {
	gone := loadedContainer("gone", "running")
	recreated := loadedContainer("sts-0", "running")

	client := new(mockedClient)
	client.On("Host").Return(Host{ID: "localhost"})
	store := &Store{containers: xsync.NewMap[string, *Container](), client: client, ctx: t.Context()}
	store.containers.Store(gone.ID, &gone)
	store.containers.Store(recreated.ID, &recreated)

	again := loadedContainer("sts-0", "running")
	again.Created = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	client.On("ListContainers", mock.Anything, mock.Anything).Return([]Container{}, nil).Run(func(mock.Arguments) {
		// the loop handles the pod's recreate while the list is in flight
		store.containers.Store(again.ID, &again)
	})

	_, err := store.refresh(true)
	assert.NoError(t, err)
	_, ok := store.containers.Load("gone")
	assert.False(t, ok, "a container the list no longer reports is removed")
	got, ok := store.containers.Load("sts-0")
	assert.True(t, ok, "the entry re-stored during the list survives")
	assert.Same(t, &again, got)
}

// A complete entry the loop stored during the list is newer than the list entry.
func TestStore_storeListedKeepsFullyLoadedLoopEntry(t *testing.T) {
	store := &Store{containers: xsync.NewMap[string, *Container]()}
	before := Container{ID: "1", State: "created"}
	store.containers.Store("1", &before)
	inspected := loadedContainer("1", "running")
	inspected.Image = "nginx"
	store.containers.Store("1", &inspected)

	store.storeListed(&before, Container{ID: "1", State: "running"}, true)
	got, _ := store.containers.Load("1")
	assert.Same(t, &inspected, got)
}

var fastTiming = storeTiming{retryMin: 5 * time.Millisecond, retryMax: 20 * time.Millisecond, subscribeGrace: 5 * time.Millisecond}

// restartingDaemon is a daemon that drops the event stream once. While the stream is
// down a container starts, so only a list taken after the reconnect can see it.
type restartingDaemon struct {
	*mockedClient
	reconnected atomic.Bool
	listFails   atomic.Int32 // lists to fail after the reconnect
	missed      Container
}

func newRestartingDaemon() *restartingDaemon {
	d := &restartingDaemon{
		mockedClient: new(mockedClient),
		missed:       Container{ID: "missed", Name: "missed", State: "running", StartedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), FullyLoaded: true, Stats: utils.NewRingBuffer[ContainerStat](300)},
	}
	d.On("Host").Return(Host{ID: "localhost"})
	d.On("FindContainer", mock.Anything, "missed").Return(d.missed, nil)
	d.On("ContainerEvents", mock.Anything, mock.Anything).Return(assert.AnError).Once()
	d.On("ContainerEvents", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		d.reconnected.Store(true)
		<-args.Get(0).(context.Context).Done()
	})
	return d
}

func (d *restartingDaemon) ListContainers(context.Context, ContainerLabels) ([]Container, error) {
	if !d.reconnected.Load() {
		return []Container{}, nil
	}
	if d.listFails.Add(-1) >= 0 {
		return nil, assert.AnError
	}
	return []Container{d.missed}, nil
}

// A dropped event stream reconnects on its own and refreshes the map, so a container
// that started while it was down shows up without anyone calling ListContainers, and
// is announced so log alerts attach to it.
func TestStore_eventStreamReconnectsAndRefreshes(t *testing.T) {
	daemon := newRestartingDaemon()
	store := newStore(t.Context(), daemon, &fakeStatsCollector{}, ContainerLabels{}, fastTiming)
	started := make(chan Container, 4)
	store.SubscribeNewContainers(t.Context(), started)
	events := make(chan ContainerEvent, 16)
	store.SubscribeEvents(t.Context(), events)

	select {
	case c := <-started:
		assert.Equal(t, "missed", c.ID)
	case <-time.After(5 * time.Second):
		t.Fatal("the container that started while the stream was down was never announced")
	}
	waitForEvent(t, events, "start")
	_, ok := store.containers.Load("missed")
	assert.True(t, ok)
}

// The refresh after a reconnect can fail while the daemon is still coming up. It
// keeps retrying instead of waiting for the next ListContainers.
func TestStore_refreshAfterReconnectRetries(t *testing.T) {
	daemon := newRestartingDaemon()
	daemon.listFails.Store(3)
	store := newStore(t.Context(), daemon, &fakeStatsCollector{}, ContainerLabels{}, fastTiming)

	assert.Eventually(t, func() bool {
		_, ok := store.containers.Load("missed")
		return ok && store.staleGen.Load() == store.freshGen.Load()
	}, 5*time.Second, 5*time.Millisecond)
}

// A caller whose context ends stops waiting on a refresh that is stuck on the daemon.
func TestStore_ListContainersHonoursCallerContext(t *testing.T) {
	client := new(mockedClient)
	client.On("Host").Return(Host{ID: "localhost"})
	unblock := make(chan struct{})
	t.Cleanup(func() { close(unblock) })
	client.On("ListContainers", mock.Anything, mock.Anything).Return([]Container{}, nil).Run(func(mock.Arguments) {
		<-unblock
	})
	client.On("ContainerEvents", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		<-args.Get(0).(context.Context).Done()
	})

	store := NewStore(t.Context(), client, &fakeStatsCollector{}, ContainerLabels{})

	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()
	done := make(chan error, 2)
	go func() { _, err := store.ListContainers(ctx, ContainerLabels{}); done <- err }()
	go func() { _, err := store.FindContainer(ctx, "1234", ContainerLabels{}); done <- err }()
	for range 2 {
		select {
		case err := <-done:
			assert.ErrorIs(t, err, context.DeadlineExceeded)
		case <-time.After(5 * time.Second):
			t.Fatal("caller kept waiting after its context ended")
		}
	}
}

// The user-label list used bare s.ctx, so a hung daemon hung the request forever.
func TestStore_userLabelListHasDeadline(t *testing.T) {
	client := new(mockedClient)
	client.On("Host").Return(Host{ID: "localhost"})
	userLabels := ContainerLabels{"team": {"a"}}
	client.On("ListContainers", mock.Anything, ContainerLabels{}).Return([]Container{}, nil)
	var hadDeadline bool
	client.On("ListContainers", mock.Anything, userLabels).Return([]Container{}, nil).Run(func(args mock.Arguments) {
		_, hadDeadline = args.Get(0).(context.Context).Deadline()
	})
	client.On("ContainerEvents", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		<-args.Get(0).(context.Context).Done()
	})

	store := NewStore(t.Context(), client, &fakeStatsCollector{}, ContainerLabels{})
	_, err := store.ListContainers(context.Background(), userLabels)
	assert.NoError(t, err)
	assert.True(t, hadDeadline)
}

// Once a refresh is running, a caller whose context ends still stops waiting on it,
// and the refresh finishes for everyone else.
func TestStore_callerLeavesRunningRefresh(t *testing.T) {
	client := new(mockedClient)
	client.On("Host").Return(Host{ID: "localhost"})
	client.On("ListContainers", mock.Anything, mock.Anything).Return([]Container{}, nil).Once()
	unblock := make(chan struct{})
	client.On("ListContainers", mock.Anything, mock.Anything).Return([]Container{}, nil).Run(func(mock.Arguments) {
		<-unblock
	})
	client.On("ContainerEvents", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		<-args.Get(0).(context.Context).Done()
	})

	store := NewStore(t.Context(), client, &fakeStatsCollector{}, ContainerLabels{})
	_, err := store.ListContainers(t.Context(), ContainerLabels{})
	assert.NoError(t, err)

	// the stream reconnected: the next list is stuck on the daemon
	store.staleGen.Add(1)
	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()
	done := make(chan error, 1)
	go func() { _, err := store.ListContainers(ctx, ContainerLabels{}); done <- err }()
	select {
	case err := <-done:
		assert.ErrorIs(t, err, context.DeadlineExceeded)
	case <-time.After(5 * time.Second):
		t.Fatal("caller stayed blocked on a refresh after its context ended")
	}

	close(unblock)
	assert.Eventually(t, func() bool {
		return store.staleGen.Load() == store.freshGen.Load()
	}, 5*time.Second, 5*time.Millisecond, "the refresh completes after the caller left")
}

func bareStore(t *testing.T, client Client) *Store {
	return &Store{
		containers:              xsync.NewMap[string, *Container](),
		newContainerSubscribers: xsync.NewMap[context.Context, chan<- Container](),
		subscribers:             xsync.NewMap[context.Context, *eventSubscriber](),
		client:                  client,
		ctx:                     t.Context(),
		timing:                  fastTiming,
	}
}

// After a reconnect the map holds list entries, which have no StartedAt or Health. A
// die during the follow-up inspect must not copy those zero values over the inspect.
func TestStore_mergeFetchedKeepsInspectDataOverListEntry(t *testing.T) {
	store := bareStore(t, nil)
	listEntry := Container{ID: "1", Name: "web", State: "running"}
	store.containers.Store("1", &listEntry)
	died := listEntry
	died.State = "exited"
	died.FinishedAt = time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	store.containers.Store("1", &died)

	startedAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	fetched := Container{ID: "1", Name: "web", State: "running", Health: "healthy", StartedAt: startedAt, FullyLoaded: true}
	got, found, updated := store.mergeFetched(&listEntry, fetched)

	assert.True(t, found)
	assert.True(t, updated)
	assert.Equal(t, "exited", got.State, "the die is newer than the inspect")
	assert.Equal(t, died.FinishedAt, got.FinishedAt)
	assert.Equal(t, startedAt, got.StartedAt, "the list entry never knew StartedAt")
	assert.Equal(t, "healthy", got.Health)
}

// A destroy the loop handled during the list must not be undone by the list entry.
func TestStore_storeListedSkipsDestroyedDuringList(t *testing.T) {
	store := bareStore(t, nil)
	before := loadedContainer("1", "running")

	stored, _ := store.storeListed(&before, Container{ID: "1", State: "running"}, true)
	assert.False(t, stored)
	_, ok := store.containers.Load("1")
	assert.False(t, ok)
}

// applyMountStats swaps the pointer every minute. That is not the loop changing the
// container, so a refresh still removes it when it is gone and still trusts the list.
func TestStore_mountStatsSwapIsNotALoopChange(t *testing.T) {
	client := new(mockedClient)
	client.On("Host").Return(Host{ID: "localhost"})
	store := bareStore(t, client)
	gone := loadedContainer("gone", "running")
	store.containers.Store(gone.ID, &gone)
	client.On("ListContainers", mock.Anything, mock.Anything).Return([]Container{}, nil).Run(func(mock.Arguments) {
		store.applyMountStats("gone", map[string]MountStat{"/data": {}})
	})

	missed, err := store.refresh(true)
	assert.NoError(t, err)
	_, ok := store.containers.Load("gone")
	assert.False(t, ok)
	assert.Equal(t, []string{"destroy:gone"}, eventNames(missed))
}

func eventNames(events []ContainerEvent) []string {
	names := make([]string, 0, len(events))
	for _, e := range events {
		names = append(names, e.Name+":"+e.ActorID)
	}
	slices.Sort(names)
	return names
}

// A refresh reports what the stream never said, so it can be replayed to subscribers:
// what started, restarted, died or went away while the stream was down.
func TestStore_refreshReportsMissedEvents(t *testing.T) {
	t1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	removed := loadedContainer("removed", "running")
	died := loadedContainer("died", "running")
	restarted := loadedContainer("restarted", "running")
	restarted.StartedAt = t1
	steady := loadedContainer("steady", "running")
	steady.StartedAt = t1

	client := new(mockedClient)
	client.On("Host").Return(Host{ID: "localhost"})
	store := bareStore(t, client)
	for _, c := range []*Container{&removed, &died, &restarted, &steady} {
		store.containers.Store(c.ID, c)
	}

	client.On("ListContainers", mock.Anything, mock.Anything).Return([]Container{
		{ID: "died", State: "exited"},
		{ID: "restarted", State: "running"},
		{ID: "steady", State: "running"},
		{ID: "new", State: "running"},
		{ID: "new-stopped", State: "exited"},
	}, nil)
	again := restarted
	again.StartedAt = t1.Add(time.Hour)
	client.On("FindContainer", mock.Anything, "restarted").Return(again, nil)
	client.On("FindContainer", mock.Anything, "steady").Return(steady, nil)
	client.On("FindContainer", mock.Anything, "new").Return(loadedContainer("new", "running"), nil)

	missed, err := store.refresh(true)
	assert.NoError(t, err)
	assert.Equal(t, []string{"destroy:removed", "die:died", "start:new", "start:restarted"}, eventNames(missed))
}

// A light refresh trusts fully loaded entries: no inspect, only the listed state.
func TestStore_lightRefreshDoesNotReinspect(t *testing.T) {
	client := new(mockedClient)
	client.On("Host").Return(Host{ID: "localhost"})
	store := bareStore(t, client)
	kept := loadedContainer("kept", "running")
	kept.Image = "nginx"
	store.containers.Store(kept.ID, &kept)
	client.On("ListContainers", mock.Anything, mock.Anything).Return([]Container{
		{ID: "kept", State: "running"},
		{ID: "new", State: "running"},
	}, nil)
	client.On("FindContainer", mock.Anything, "new").Return(loadedContainer("new", "running"), nil)

	missed, err := store.refresh(false)
	assert.NoError(t, err)
	client.AssertNotCalled(t, "FindContainer", mock.Anything, "kept")
	got, _ := store.containers.Load("kept")
	assert.Same(t, &kept, got)
	assert.Equal(t, []string{"start:new"}, eventNames(missed))
}

// A stale mark that lands while a refresh is running is not covered by it. The
// refresher keeps going until the map is fresh instead of stopping on the first nil.
func TestStore_refresherCoversStaleMarkedDuringRefresh(t *testing.T) {
	client := new(mockedClient)
	client.On("Host").Return(Host{ID: "localhost"})
	store := bareStore(t, client)
	store.refreshWake = make(chan struct{}, 1)
	store.freshGen.Store(1)
	store.staleGen.Store(1)

	var lists atomic.Int32
	client.On("ListContainers", mock.Anything, mock.Anything).Return([]Container{}, nil).Run(func(mock.Arguments) {
		if lists.Add(1) == 1 {
			// the stream reconnects again while this list is in flight
			store.markStale()
		}
	})

	go store.refresher()
	store.markStale()

	assert.Eventually(t, func() bool {
		return store.staleGen.Load() == store.freshGen.Load()
	}, 5*time.Second, 5*time.Millisecond)
	assert.Equal(t, uint64(3), store.freshGen.Load())
	assert.GreaterOrEqual(t, lists.Load(), int32(2))
}

// An announcement nobody received is not recorded, so the start that follows a create
// still announces the container.
func TestStore_undeliveredAnnouncementIsRetried(t *testing.T) {
	client := new(mockedClient)
	client.On("Host").Return(Host{ID: "localhost"})
	store := bareStore(t, client)
	c := loadedContainer("1", "running")
	c.StartedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	store.notifyNewContainer(c) // nobody subscribed yet

	started := make(chan Container, 2)
	store.newContainerSubscribers.Store(t.Context(), started)
	store.notifyNewContainer(c)
	store.notifyNewContainer(c)
	assert.Len(t, started, 1, "announced once it could be delivered, and only once")
}

// Concurrent lookups of a container that is not fully loaded share one inspect.
func TestStore_FindContainerSharesOneInspect(t *testing.T) {
	client := new(mockedClient)
	client.On("Host").Return(Host{ID: "localhost"})
	store := bareStore(t, client)
	store.ready = make(chan struct{})
	close(store.ready)
	partial := Container{ID: "1", State: "exited"}
	store.containers.Store("1", &partial)

	release := make(chan struct{})
	client.On("FindContainer", mock.Anything, "1").Return(loadedContainer("1", "exited"), nil).Run(func(mock.Arguments) {
		<-release
	})

	var wg sync.WaitGroup
	for range 5 {
		wg.Go(func() {
			c, err := store.FindContainer(t.Context(), "1", ContainerLabels{})
			assert.NoError(t, err)
			assert.True(t, c.FullyLoaded)
		})
	}
	time.Sleep(50 * time.Millisecond)
	close(release)
	wg.Wait()
	client.AssertNumberOfCalls(t, "FindContainer", 1)
}

// An inspect that fails on create leaves a list entry, which has no StartedAt. The
// start that follows inspects successfully, and dedupe on StartedAt alone compared a
// zero time against a real one and announced the same start twice.
func TestStore_listFallbackThenStartNotifiesOnce(t *testing.T) {
	running := loadedContainer("5678", "running")
	running.StartedAt = time.Now().Add(-time.Minute)
	listed := Container{ID: "5678", Name: "c-5678", State: "running", Host: "localhost"}

	client := new(mockedClient)
	client.On("ListContainers", mock.Anything, mock.Anything).Return([]Container{listed}, nil)
	client.On("FindContainer", mock.Anything, "5678").Return(Container{}, errors.New("daemon busy")).Once()
	client.On("FindContainer", mock.Anything, "5678").Return(running, nil)
	client.On("Host").Return(Host{ID: "localhost"})
	feed := feedEvents(client)

	store := NewStore(t.Context(), client, newCaptureStatsCollector(), ContainerLabels{})
	events := make(chan ContainerEvent, 16)
	store.SubscribeEvents(t.Context(), events)
	started := make(chan Container, 4)
	store.SubscribeNewContainers(t.Context(), started)

	feed <- ContainerEvent{Name: "create", ActorID: "5678"}
	waitForEvent(t, events, "create")
	feed <- ContainerEvent{Name: "start", ActorID: "5678"}
	waitForEvent(t, events, "start")
	assert.Len(t, started, 1)

	// a restart is still a new run
	<-started
	restarted := running
	restarted.StartedAt = time.Now().Add(time.Minute)
	client.ExpectedCalls = nil
	client.On("FindContainer", mock.Anything, "5678").Return(restarted, nil)
	client.On("Host").Return(Host{ID: "localhost"})
	feed <- ContainerEvent{Name: "start", ActorID: "5678"}
	waitForEvent(t, events, "start")
	assert.Len(t, started, 1)
}

// Only FindContainer used to go through the singleflight, so a refresh filling in a
// list entry and the event loop adding a container each ran their own inspect of the
// same container at the same time.
func TestStore_inspectIsSharedAcrossPaths(t *testing.T) {
	client := new(mockedClient)
	client.On("Host").Return(Host{ID: "localhost"})
	store := bareStore(t, client)
	store.ready = make(chan struct{})
	close(store.ready)
	partial := Container{ID: "1", State: "running"}
	store.containers.Store("1", &partial)

	release := make(chan struct{})
	client.On("FindContainer", mock.Anything, "1").Return(loadedContainer("1", "running"), nil).Run(func(mock.Arguments) {
		<-release
	})

	var wg sync.WaitGroup
	wg.Go(func() {
		c, err := store.FindContainer(t.Context(), "1", ContainerLabels{})
		assert.NoError(t, err)
		assert.True(t, c.FullyLoaded)
	})
	wg.Go(func() {
		store.inspectPartial([]Container{{ID: "1", State: "running"}})
	})
	wg.Go(func() {
		c, ok := store.addContainer("1", time.Second)
		assert.True(t, ok)
		assert.True(t, c.FullyLoaded)
	})

	time.Sleep(50 * time.Millisecond)
	close(release)
	wg.Wait()
	client.AssertNumberOfCalls(t, "FindContainer", 1)
}

// A container inspected while the goroutine sat in the parallelism queue must not be
// inspected again: two overlapping refreshes used to inspect every container twice.
func TestStore_inspectPartialSkipsWhatLoadedWhileQueued(t *testing.T) {
	client := new(mockedClient)
	client.On("Host").Return(Host{ID: "localhost"})
	store := bareStore(t, client)
	first := Container{ID: "1", State: "running"}
	second := Container{ID: "2", State: "running"}
	store.containers.Store("1", &first)
	store.containers.Store("2", &second)

	release := make(chan struct{})
	client.On("FindContainer", mock.Anything, "1").Return(loadedContainer("1", "running"), nil).Run(func(mock.Arguments) {
		// "2" is inspected by someone else while this one holds the only slot
		loaded := loadedContainer("2", "running")
		store.containers.Store("2", &loaded)
		<-release
	})
	client.On("FindContainer", mock.Anything, "2").Return(loadedContainer("2", "running"), nil)

	original := maxFetchParallelism
	maxFetchParallelism = 1
	t.Cleanup(func() { maxFetchParallelism = original })

	done := make(chan struct{})
	go func() {
		store.inspectPartial([]Container{first, second})
		close(done)
	}()
	time.Sleep(50 * time.Millisecond)
	close(release)
	<-done

	client.AssertNumberOfCalls(t, "FindContainer", 1)
}

package container

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// sizedClient is a mockedClient that can also measure sizes.
type sizedClient struct {
	*mockedClient
	mu       sync.Mutex
	all      map[string]int64
	one      map[string]int64
	measured []string
	// batches to fail before one succeeds
	allFails  int
	allCalled int
}

func (c *sizedClient) ContainerSizes(context.Context) (map[string]int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.allCalled++
	if c.allCalled <= c.allFails {
		return nil, errors.New("timed out")
	}
	return c.all, nil
}

func (c *sizedClient) ContainerSize(_ context.Context, id string) (int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.measured = append(c.measured, id)
	size, ok := c.one[id]
	if !ok {
		return 0, errors.New("no size")
	}
	return size, nil
}

func (c *sizedClient) measuredIDs() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string(nil), c.measured...)
}

func queued(m *sizeMonitor) bool {
	select {
	case id := <-m.queue:
		m.pending.Delete(id)
		return true
	default:
		return false
	}
}

func TestSizeMonitor_nilWithoutSizeReader(t *testing.T) {
	m := newSizeMonitor(bareStore(t, nil), new(mockedClient))
	assert.Nil(t, m, "k8s has no sizes to read")

	c := loadedContainer("1", "running")
	// every method is a no-op on nil, so the store calls them unconditionally
	m.start(t.Context())
	m.measureAll(t.Context())
	m.observe(&c, ContainerStat{ID: "1"})
	m.died("1")
	m.forget("1")
}

func TestSizeMonitor_observe(t *testing.T) {
	m := newSizeMonitor(bareStore(t, nil), &sizedClient{mockedClient: new(mockedClient)})
	c := loadedContainer("1", "running")

	m.observe(&c, ContainerStat{ID: "1", DiskWriteTotal: 500})
	assert.True(t, queued(m), "a container never measured is measured on its first stat")

	size := int64(10)
	c.SizeRw = &size
	tr, _ := m.trackers.Load("1")
	tr.lastCheckNanos.Store(time.Now().UnixNano())

	m.observe(&c, ContainerStat{ID: "1", DiskWriteTotal: 500 + 1<<20})
	assert.False(t, queued(m), "a small write right after a measurement waits")

	m.observe(&c, ContainerStat{ID: "1", DiskWriteTotal: 500 + sizeWriteThreshold})
	assert.True(t, queued(m), "a large write is measured right away")

	tr.lastCheckNanos.Store(time.Now().Add(-sizeRefreshInterval).UnixNano())
	m.observe(&c, ContainerStat{ID: "1", DiskWriteTotal: 500})
	assert.False(t, queued(m), "a container that wrote nothing is not walked again")

	m.observe(&c, ContainerStat{ID: "1", DiskWriteTotal: 501})
	assert.True(t, queued(m), "any write is measured once the interval has passed")

	c.SizeRw = nil
	m.observe(&c, ContainerStat{ID: "1", DiskWriteTotal: 500})
	assert.True(t, queued(m), "a failed measurement is retried on the interval")
}

func TestSizeMonitor_observeAfterRestart(t *testing.T) {
	m := newSizeMonitor(bareStore(t, nil), &sizedClient{mockedClient: new(mockedClient)})
	c := loadedContainer("1", "running")
	size := int64(10)
	c.SizeRw = &size

	m.observe(&c, ContainerStat{ID: "1", DiskWriteTotal: sizeWriteThreshold * 3})
	queued(m)
	tr, _ := m.trackers.Load("1")
	tr.lastCheckNanos.Store(time.Now().UnixNano())

	// the counter starts over; the new total is the delta
	m.observe(&c, ContainerStat{ID: "1", DiskWriteTotal: sizeWriteThreshold})
	assert.True(t, queued(m))
}

func TestSizeMonitor_measure(t *testing.T) {
	client := &sizedClient{mockedClient: new(mockedClient), one: map[string]int64{"1": 4096}}
	store := bareStore(t, client)
	c := loadedContainer("1", "exited")
	store.containers.Store("1", &c)
	events := make(chan ContainerEvent, 2)
	store.subscribers.Store(t.Context(), &eventSubscriber{ch: events, name: "test"})

	m := newSizeMonitor(store, client)
	m.measure(t.Context(), "1")

	stored, _ := store.containers.Load("1")
	if assert.NotNil(t, stored.SizeRw) {
		assert.Equal(t, int64(4096), *stored.SizeRw)
	}
	assert.Nil(t, c.SizeRw, "the previous entry should not be mutated")
	update := <-events
	assert.Equal(t, "update", update.Name)
	assert.Equal(t, int64(4096), *update.Container.SizeRw)

	m.measure(t.Context(), "1")
	assert.Empty(t, events, "an unchanged size does not broadcast")

	m.measure(t.Context(), "unknown")
	assert.Equal(t, []string{"1", "1"}, client.measuredIDs(), "a container the store does not know is not measured")
}

func TestSizeMonitor_measureAll(t *testing.T) {
	client := &sizedClient{mockedClient: new(mockedClient), all: map[string]int64{"1": 100, "2": 0, "gone": 5}}
	store := bareStore(t, client)
	for _, id := range []string{"1", "2"} {
		c := loadedContainer(id, "exited")
		store.containers.Store(id, &c)
	}

	newSizeMonitor(store, client).measureAll(t.Context())

	one, _ := store.containers.Load("1")
	two, _ := store.containers.Load("2")
	assert.Equal(t, int64(100), *one.SizeRw)
	assert.Equal(t, int64(0), *two.SizeRw, "an empty layer is a size, not unknown")
	_, ok := store.containers.Load("gone")
	assert.False(t, ok)
}

// DiskWriteTotal counts from the container's start. The batch creates the tracker
// before any stat arrives, and the first stat must still be read as the baseline,
// or a database's lifetime of writes would re-walk it right after the batch did.
func TestSizeMonitor_firstStatAfterBatchIsBaseline(t *testing.T) {
	client := &sizedClient{mockedClient: new(mockedClient), all: map[string]int64{"1": 100}}
	store := bareStore(t, client)
	c := loadedContainer("1", "running")
	store.containers.Store("1", &c)
	m := newSizeMonitor(store, client)

	m.measureAll(t.Context())
	measured, _ := store.containers.Load("1")

	m.observe(measured, ContainerStat{ID: "1", DiskWriteTotal: 50 * sizeWriteThreshold})
	assert.False(t, queued(m), "lifetime writes before the batch are not new writes")

	tr, _ := m.trackers.Load("1")
	tr.lastCheckNanos.Store(time.Now().Add(-sizeRefreshInterval).UnixNano())
	m.observe(measured, ContainerStat{ID: "1", DiskWriteTotal: 50 * sizeWriteThreshold})
	assert.False(t, queued(m), "nor do they count as writing something by the interval")
}

func TestSizeMonitor_measureAllRetries(t *testing.T) {
	client := &sizedClient{mockedClient: new(mockedClient), all: map[string]int64{"1": 100}, allFails: sizeBatchAttempts - 1}
	store := bareStore(t, client)
	c := loadedContainer("1", "exited")
	store.containers.Store("1", &c)

	newSizeMonitor(store, client).measureAll(t.Context())

	stored, _ := store.containers.Load("1")
	if assert.NotNil(t, stored.SizeRw, "a stopped container has no other chance to be measured") {
		assert.Equal(t, int64(100), *stored.SizeRw)
	}

	client.allFails, client.allCalled = sizeBatchAttempts, 0
	newSizeMonitor(store, client).measureAll(t.Context())
	assert.Equal(t, sizeBatchAttempts, client.allCalled, "each retry walks the whole host, so they are bounded")
}

func TestCarryOverStatsKeepsSize(t *testing.T) {
	size := int64(42)
	from := Container{ID: "1", SizeRw: &size}
	to := Container{ID: "1"}
	carryOverStats(&from, &to)
	assert.Equal(t, &size, to.SizeRw, "a list or inspect result has no size and must not wipe it")
}

func TestStore_measuresAtBootAndOnDie(t *testing.T) {
	base := new(mockedClient)
	base.On("ListContainers", mock.Anything, mock.Anything).Return([]Container{
		loadedContainer("running", "running"),
		loadedContainer("stopped", "exited"),
	}, nil)
	base.On("Host").Return(Host{ID: "localhost"})
	events := feedEvents(base)
	client := &sizedClient{
		mockedClient: base,
		all:          map[string]int64{"running": 1, "stopped": 2},
		one:          map[string]int64{"running": 3},
	}

	store := NewStore(t.Context(), client, newCaptureStatsCollector(), ContainerLabels{})
	assert.Eventually(t, func() bool {
		c, _ := store.containers.Load("stopped")
		return c != nil && c.SizeRw != nil && *c.SizeRw == 2
	}, 5*time.Second, 5*time.Millisecond, "a stopped container is measured once after the first list")

	events <- ContainerEvent{Name: "die", ActorID: "running", Host: "localhost"}
	assert.Eventually(t, func() bool {
		c, _ := store.containers.Load("running")
		return c != nil && c.SizeRw != nil && *c.SizeRw == 3
	}, 5*time.Second, 5*time.Millisecond, "a container is measured again when it dies")
}

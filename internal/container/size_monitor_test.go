package container

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
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

	volumes       map[string][]VolumeUsage
	reclaimable   Reclaimable
	volumesCalled atomic.Int32
}

func (c *sizedClient) DiskUsage(context.Context) (DiskUsage, error) {
	c.volumesCalled.Add(1)
	c.mu.Lock()
	defer c.mu.Unlock()
	return DiskUsage{Volumes: c.volumes, Reclaimable: c.reclaimable}, nil
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

func TestSizeMonitor_measureVolumes(t *testing.T) {
	data := VolumeUsage{Name: "clickhouse_data", Destination: "/var/lib/clickhouse", Size: 23 << 30, Links: 1}
	client := &sizedClient{mockedClient: new(mockedClient), volumes: map[string][]VolumeUsage{"db": {data}}}
	store := bareStore(t, client)
	for _, id := range []string{"db", "web"} {
		c := loadedContainer(id, "running")
		store.containers.Store(id, &c)
	}
	events := make(chan ContainerEvent, 4)
	store.subscribers.Store(t.Context(), &eventSubscriber{ch: events, name: "test"})
	m := newSizeMonitor(store, client)

	assert.True(t, m.claimVolumes())
	m.measureVolumes(t.Context())

	db, _ := store.containers.Load("db")
	web, _ := store.containers.Load("web")
	assert.Equal(t, []VolumeUsage{data}, db.Volumes)
	assert.Nil(t, web.Volumes)
	assert.Len(t, events, 1, "only the container whose volumes changed broadcasts")

	// the volume is removed along with its container's mount
	client.volumes = nil
	m.lastVolumesNanos.Store(0)
	assert.True(t, m.claimVolumes())
	m.measureVolumes(t.Context())
	db, _ = store.containers.Load("db")
	assert.Nil(t, db.Volumes)
}

func TestSizeMonitor_volumesAtMostOncePerInterval(t *testing.T) {
	client := &sizedClient{mockedClient: new(mockedClient)}
	m := newSizeMonitor(bareStore(t, client), client)

	assert.True(t, m.claimVolumes(), "the first walk is due")
	assert.False(t, m.claimVolumes(), "a walk already running is not started again")
	m.volumesRunning.Store(false)
	assert.False(t, m.claimVolumes(), "a walk that just ran is not due")

	m.lastVolumesNanos.Store(time.Now().Add(-volumeRefreshInterval).UnixNano())
	assert.True(t, m.claimVolumes(), "due again after the interval")
}

func TestSizeMonitor_statsTriggerOneVolumeWalk(t *testing.T) {
	client := &sizedClient{mockedClient: new(mockedClient)}
	m := newSizeMonitor(bareStore(t, client), client)
	c := loadedContainer("1", "running")

	m.observe(&c, ContainerStat{ID: "1"})
	assert.False(t, m.volumesRunning.Load(), "the first walk waits for the layer batch, not for a stat")
	assert.Equal(t, int32(0), client.volumesCalled.Load())

	// the batch ran its walk a while ago
	m.lastVolumesNanos.Store(time.Now().Add(-volumeRefreshInterval).UnixNano())
	for range 5 {
		m.observe(&c, ContainerStat{ID: "1"})
	}
	assert.Eventually(t, func() bool { return client.volumesCalled.Load() == 1 && !m.volumesRunning.Load() }, 5*time.Second, 5*time.Millisecond)
	m.observe(&c, ContainerStat{ID: "1"})
	assert.Equal(t, int32(1), client.volumesCalled.Load(), "stats inside the interval walk nothing")
}

// The layer batch and the first volume walk both walk the whole host, so they run one
// after the other, and a batch that gives up still leaves volumes measured.
func TestSizeMonitor_volumesFollowTheBatch(t *testing.T) {
	client := &sizedClient{
		mockedClient: new(mockedClient),
		allFails:     sizeBatchAttempts,
		volumes:      map[string][]VolumeUsage{"1": {{Name: "data", Size: 5, Links: 1}}},
	}
	store := bareStore(t, client)
	c := loadedContainer("1", "exited")
	store.containers.Store("1", &c)

	newSizeMonitor(store, client).measureAll(t.Context())

	assert.Equal(t, sizeBatchAttempts, client.allCalled)
	assert.Equal(t, int32(1), client.volumesCalled.Load())
	stored, _ := store.containers.Load("1")
	assert.Len(t, stored.Volumes, 1)
}

// The engine reports images, volumes and build cache. Stopped containers come from the
// layer sizes the store already has, so their layers are not walked a second time.
func TestSizeMonitor_reclaimable(t *testing.T) {
	client := &sizedClient{mockedClient: new(mockedClient), reclaimable: Reclaimable{Images: 3, ImagesSize: 300, BuildCacheSize: 7}}
	store := bareStore(t, client)
	size := func(n int64) *int64 { return &n }
	for _, c := range []Container{
		{ID: "running", State: "running", SizeRw: size(1000)},
		{ID: "paused", State: "paused", SizeRw: size(1000)},
		{ID: "exited", State: "exited", SizeRw: size(40)},
		{ID: "created", State: "created", SizeRw: size(2)},
		{ID: "unmeasured", State: "exited"},
	} {
		store.containers.Store(c.ID, &c)
	}
	m := newSizeMonitor(store, client)
	assert.Nil(t, store.Reclaimable(), "nothing is known before the first walk")

	assert.True(t, m.claimVolumes())
	m.measureVolumes(t.Context())

	assert.Equal(t, &Reclaimable{Images: 3, ImagesSize: 300, Containers: 2, ContainersSize: 42, BuildCacheSize: 7}, store.Reclaimable())
}

func TestCarryOverStatsKeepsVolumes(t *testing.T) {
	from := Container{ID: "1", Volumes: []VolumeUsage{{Name: "data", Size: 1}}}
	to := Container{ID: "1"}
	carryOverStats(&from, &to)
	assert.Equal(t, from.Volumes, to.Volumes)
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
		volumes:      map[string][]VolumeUsage{"stopped": {{Name: "data", Size: 9, Links: 1}}},
	}

	store := NewStore(t.Context(), client, newCaptureStatsCollector(), ContainerLabels{})
	assert.Eventually(t, func() bool {
		c, _ := store.containers.Load("stopped")
		return c != nil && c.SizeRw != nil && *c.SizeRw == 2
	}, 5*time.Second, 5*time.Millisecond, "a stopped container is measured once after the first list")
	assert.Eventually(t, func() bool {
		c, _ := store.containers.Load("stopped")
		return len(c.Volumes) == 1
	}, 5*time.Second, 5*time.Millisecond, "volumes are measured right after the batch, stopped containers included")

	events <- ContainerEvent{Name: "die", ActorID: "running", Host: "localhost"}
	assert.Eventually(t, func() bool {
		c, _ := store.containers.Load("running")
		return c != nil && c.SizeRw != nil && *c.SizeRw == 3
	}, 5*time.Second, 5*time.Millisecond, "a container is measured again when it dies")
}

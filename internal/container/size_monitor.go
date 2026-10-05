package container

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/puzpuzpuz/xsync/v4"
	"github.com/rs/zerolog/log"
)

const (
	// sizeWriteThreshold re-measures a running container once it has written this
	// much since its last measurement, so one filling the disk shows up quickly.
	sizeWriteThreshold uint64 = 100 << 20 // 100 MiB
	// sizeRefreshInterval re-measures a container that wrote anything at all since
	// its last measurement. One that wrote nothing is never walked again.
	sizeRefreshInterval = 5 * time.Minute
	sizeQueueSize       = 64
	sizeTimeout         = 30 * time.Second
	// sizeBatchTimeout is longer: the batch walks every layer on the host, and on a
	// big one that alone takes minutes.
	sizeBatchTimeout = 2 * time.Minute
	// sizeBatchAttempts bounds the retries, since each one walks the whole host again.
	sizeBatchAttempts = 3
)

// sizeTracker is shared between observe (the event loop) and the worker.
// lastCheckNanos is the last attempt, as UnixNano; zero means never. baselined is
// set once lastWriteTotal holds a real reading: DiskWriteTotal counts from the
// container's start, so before that a whole lifetime of writes would read as new.
type sizeTracker struct {
	lastWriteTotal atomic.Uint64
	lastCheckNanos atomic.Int64
	baselined      atomic.Bool
}

// sizeMonitor keeps Container.SizeRw current without walking anything nobody is
// looking at. Every container is measured once in one batch after the first list.
// After that a running one is re-measured from its stats, which only flow while a
// subscriber is connected, and a stopped one once more when it dies, since its
// layer cannot change after that.
//
// A nil *sizeMonitor is a client that cannot measure sizes, and every method is a no-op.
type sizeMonitor struct {
	store    *Store
	reader   SizeReader
	queue    chan string
	pending  *xsync.Map[string, struct{}]
	trackers *xsync.Map[string, *sizeTracker]
}

func newSizeMonitor(store *Store, client Client) *sizeMonitor {
	reader, ok := client.(SizeReader)
	if !ok {
		return nil
	}
	return &sizeMonitor{
		store:    store,
		reader:   reader,
		queue:    make(chan string, sizeQueueSize),
		pending:  xsync.NewMap[string, struct{}](),
		trackers: xsync.NewMap[string, *sizeTracker](),
	}
}

// start runs one worker: measurements are walks on the daemon, and running them
// side by side would only make each slower.
func (m *sizeMonitor) start(ctx context.Context) {
	if m == nil {
		return
	}
	go m.worker(ctx)
}

// measureAll measures every container in one call. It is the only measurement a
// stopped container gets, so a failure is retried a few times.
func (m *sizeMonitor) measureAll(ctx context.Context) {
	if m == nil {
		return
	}
	backoff := m.store.timing.retryMin
	for attempt := 1; ; attempt++ {
		err := m.tryMeasureAll(ctx)
		if err == nil {
			return
		}
		if attempt == sizeBatchAttempts {
			log.Warn().Err(err).Msg("could not measure container sizes, stopped containers will show none")
			return
		}
		log.Debug().Err(err).Dur("retry_in", backoff).Msg("could not measure container sizes, retrying")
		if !SleepOrDone(ctx, backoff) {
			return
		}
		backoff = NextBackoff(backoff, m.store.timing.retryMax)
	}
}

func (m *sizeMonitor) tryMeasureAll(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, sizeBatchTimeout)
	defer cancel()
	sizes, err := m.reader.ContainerSizes(ctx)
	if err != nil {
		return err
	}
	now := time.Now().UnixNano()
	for id, size := range sizes {
		m.store.applySize(id, size)
		t, _ := m.trackers.LoadOrCompute(id, func() (*sizeTracker, bool) { return &sizeTracker{}, false })
		t.lastCheckNanos.Store(now)
	}
	return nil
}

// observe is called for every stat. It queues a measurement when the container has
// never been measured, has written a lot since, or has written anything and is due.
func (m *sizeMonitor) observe(c *Container, stat ContainerStat) {
	if m == nil {
		return
	}
	t, _ := m.trackers.LoadOrCompute(c.ID, func() (*sizeTracker, bool) {
		return &sizeTracker{}, false
	})
	// The first stat is the baseline, not a write, even when the batch created the
	// tracker earlier. Only the event loop calls observe, so this cannot race itself.
	if !t.baselined.Load() {
		t.lastWriteTotal.Store(stat.DiskWriteTotal)
		t.baselined.Store(true)
	}

	last := t.lastWriteTotal.Load()
	delta := stat.DiskWriteTotal - last
	if stat.DiskWriteTotal < last {
		// the counter starts over when the container restarts
		delta = stat.DiskWriteTotal
	}

	lastCheck := t.lastCheckNanos.Load()
	if lastCheck == 0 {
		m.enqueue(c.ID)
		return
	}
	elapsed := time.Since(time.Unix(0, lastCheck))
	// a failed measurement leaves SizeRw nil and is retried on the interval
	if delta >= sizeWriteThreshold || (elapsed >= sizeRefreshInterval && (delta > 0 || c.SizeRw == nil)) {
		m.enqueue(c.ID)
	}
}

// died measures a container one last time: nothing writes to its layer once it stops.
func (m *sizeMonitor) died(id string) {
	if m == nil {
		return
	}
	m.enqueue(id)
}

func (m *sizeMonitor) forget(id string) {
	if m == nil {
		return
	}
	m.trackers.Delete(id)
}

func (m *sizeMonitor) enqueue(id string) {
	if _, loaded := m.pending.LoadOrStore(id, struct{}{}); loaded {
		return
	}
	select {
	case m.queue <- id:
	default:
		// full; a later stat or event queues it again
		m.pending.Delete(id)
	}
}

func (m *sizeMonitor) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case id := <-m.queue:
			m.pending.Delete(id)
			m.measure(ctx, id)
		}
	}
}

func (m *sizeMonitor) measure(ctx context.Context, id string) {
	c, ok := m.store.containers.Load(id)
	if !ok {
		m.trackers.Delete(id)
		return
	}

	// Mark the attempt before the call, so the stats that arrive while the daemon
	// walks the layer do not queue it again.
	t, _ := m.trackers.LoadOrCompute(id, func() (*sizeTracker, bool) { return &sizeTracker{}, false })
	if c.Stats != nil {
		if data := c.Stats.Data(); len(data) > 0 {
			t.lastWriteTotal.Store(data[len(data)-1].DiskWriteTotal)
			t.baselined.Store(true)
		}
	}
	t.lastCheckNanos.Store(time.Now().UnixNano())

	ctx, cancel := context.WithTimeout(ctx, sizeTimeout)
	defer cancel()
	size, err := m.reader.ContainerSize(ctx, id)
	if err != nil {
		log.Debug().Err(err).Str("id", id).Msg("could not measure container size")
		return
	}
	m.store.applySize(id, size)
}

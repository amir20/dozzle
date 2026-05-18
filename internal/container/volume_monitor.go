package container

import (
	"context"
	"time"

	"github.com/puzpuzpuz/xsync/v4"
	"github.com/rs/zerolog/log"
)

const (
	// writeThresholdBytes triggers a volume refresh once a container has
	// written this many bytes since the last refresh.
	writeThresholdBytes uint64 = 1 << 20 // 1 MiB
	// idleRefreshInterval forces a periodic refresh even when no writes are
	// happening, so newly started containers get an initial reading and
	// out-of-band host changes are eventually picked up.
	idleRefreshInterval = 60 * time.Second
	volumeWorkerCount   = 2
	volumeQueueSize     = 64
)

type volumeTracker struct {
	lastWriteTotal uint64
	lastCheck      time.Time
}

type volumeMonitor struct {
	store    *ContainerStore
	queue    chan string
	pending  *xsync.Map[string, struct{}]
	trackers *xsync.Map[string, *volumeTracker]
}

func newVolumeMonitor(store *ContainerStore) *volumeMonitor {
	return &volumeMonitor{
		store:    store,
		queue:    make(chan string, volumeQueueSize),
		pending:  xsync.NewMap[string, struct{}](),
		trackers: xsync.NewMap[string, *volumeTracker](),
	}
}

func (v *volumeMonitor) start(ctx context.Context) {
	for i := 0; i < volumeWorkerCount; i++ {
		go v.worker(ctx)
	}
}

// observe is called for every incoming container stat. It decides whether to
// enqueue a volume refresh for the container.
func (v *volumeMonitor) observe(c *Container, stat ContainerStat) {
	if len(c.Mounts) == 0 {
		return
	}

	t, _ := v.trackers.LoadOrCompute(c.ID, func() (*volumeTracker, bool) {
		// Initialize with the current write total so the first refresh is
		// driven by the idle timer, not a phantom delta.
		return &volumeTracker{lastWriteTotal: stat.DiskWriteTotal}, false
	})

	delta := stat.DiskWriteTotal - t.lastWriteTotal
	if stat.DiskWriteTotal < t.lastWriteTotal {
		// Counter reset (container restarted with same ID? unlikely but defend).
		delta = stat.DiskWriteTotal
	}

	now := time.Now()
	idle := t.lastCheck.IsZero() || now.Sub(t.lastCheck) >= idleRefreshInterval
	if !idle && delta < writeThresholdBytes {
		return
	}

	v.enqueue(c.ID)
}

func (v *volumeMonitor) enqueue(id string) {
	if _, loaded := v.pending.LoadOrStore(id, struct{}{}); loaded {
		return
	}
	select {
	case v.queue <- id:
	default:
		// Queue is full; drop and let the next tick try again.
		v.pending.Delete(id)
	}
}

func (v *volumeMonitor) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case id := <-v.queue:
			v.pending.Delete(id)
			v.refresh(id)
		}
	}
}

func (v *volumeMonitor) refresh(id string) {
	c, ok := v.store.containers.Load(id)
	if !ok {
		v.trackers.Delete(id)
		return
	}

	mounts := c.Mounts
	if len(mounts) == 0 {
		return
	}

	stats := make(map[string]MountStat, len(mounts))
	for _, m := range mounts {
		ms := MountStat{
			Destination: m.Destination,
			LastChecked: time.Now(),
		}
		if m.Source == "" {
			stats[m.Destination] = ms
			continue
		}
		total, free, err := statfs(m.Source)
		if err != nil {
			log.Debug().Err(err).Str("id", c.ID).Str("source", m.Source).Str("dest", m.Destination).Msg("statfs failed")
			stats[m.Destination] = ms
			continue
		}
		ms.Available = true
		ms.Total = total
		ms.Free = free
		if total > free {
			ms.Used = total - free
		}
		stats[m.Destination] = ms
	}

	// Latest stat may have moved on; read it back from the container ring.
	var latestWrite uint64
	if data := c.Stats.Data(); len(data) > 0 {
		latestWrite = data[len(data)-1].DiskWriteTotal
	}
	v.trackers.Store(id, &volumeTracker{lastWriteTotal: latestWrite, lastCheck: time.Now()})

	v.store.applyMountStats(id, stats)
}

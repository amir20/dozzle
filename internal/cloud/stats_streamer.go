package cloud

import (
	"cmp"
	"context"
	"slices"
	"time"

	"github.com/amir20/dozzle/internal/container"
	pb "github.com/amir20/dozzle/proto/cloud"
	"github.com/rs/zerolog/log"
)

// StatSample pairs a raw container stat with the host it came from.
// container.ContainerStat carries only an ID, and the cloud host service fans
// stats in from several local client services, so the host has to be stamped
// at the fan-in point or it is lost.
type StatSample struct {
	Stat   container.ContainerStat
	HostID string
}

// StatsStreamHostService is the subset of the host service the stats streamer
// needs. It is type-asserted at connect time; when the running host service
// doesn't satisfy it (k8s) the streamer is skipped, exactly as
// LogStreamHostService is today.
type StatsStreamHostService interface {
	ToolHostService
	SubscribeStats(ctx context.Context, samples chan<- StatSample)
}

const (
	// statsWindow is the aggregation window. Docker pushes stats at roughly
	// 1Hz, so this folds ~30 samples per container into one entry.
	statsWindow = 30 * time.Second
	// statsMaxEntries caps a single batch. A host with more running containers
	// than this reports a truncated batch and logs the drop — never silently.
	statsMaxEntries = 2000
	// statsSampleChanBuf absorbs a tick's worth of samples from a busy host
	// without blocking the collector's fan-out loop.
	statsSampleChanBuf = 512
	// statsMaxStaleTicks is how many windows an accumulator may survive without
	// its container resolving in the metadata snapshot before it is discarded.
	// One tick of grace covers a container that started mid-window; beyond that
	// it is gone and holding the samples only leaks memory.
	statsMaxStaleTicks = 2
)

// seriesKey identifies an accumulator. Keyed on container ID because that is
// what a raw stat carries; the ID is resolved to a name at emit time and never
// leaves this process — see the StatsBatchEntry comment in cloud.proto for why
// the wire format is keyed on name instead.
type seriesKey struct {
	hostID      string
	containerID string
}

// statAcc folds a window's samples for one container. Gauges accumulate a sum
// (for the mean) and a running max; counters are monotonic totals, so only the
// most recent value matters.
type statAcc struct {
	cpuSum, cpuMax float64
	memSum, memMax float64
	memUsageSum    float64
	samples        uint32

	networkRx, networkTx uint64
	diskRead, diskWrite  uint64

	staleTicks int
}

// containerMeta is the per-window metadata snapshot needed to turn raw stats
// into a wire entry: the container's name, the core count to normalise CPU by,
// and whether the user opted the container out.
type containerMeta struct {
	name     string
	cores    float64
	disabled bool
}

// statsStreamer aggregates raw container stats into fixed windows and pushes
// one StatsBatch per window to Dozzle Cloud. Unlike logStreamer it runs a
// single goroutine for every container on the instance — stats already arrive
// multiplexed on one channel, so there is nothing to parallelise.
type statsStreamer struct {
	hostService StatsStreamHostService
	labels      container.ContainerLabels
	send        func(resp *pb.ToolResponse) error

	acc map[seriesKey]*statAcc

	// meta is the last-good container metadata snapshot. It is owned entirely
	// by run()'s goroutine: refreshes happen off-loop and hand the new snapshot
	// back over metaCh, so there is no shared state and no lock.
	meta map[seriesKey]containerMeta
	// metaCh delivers a completed async refresh. Buffered so a refresh that
	// finishes after run() has returned doesn't leak its goroutine.
	metaCh chan map[seriesKey]containerMeta
	// refreshing guards against piling up refreshes when the host service is
	// slower than the window.
	refreshing bool

	// window is the flush interval. A field rather than a bare const so tests
	// can drive real windows through run() instead of reaching into state the
	// run goroutine owns.
	window time.Duration
}

func newStatsStreamer(hostService StatsStreamHostService, labels container.ContainerLabels, send func(resp *pb.ToolResponse) error) *statsStreamer {
	return &statsStreamer{
		hostService: hostService,
		labels:      labels,
		send:        send,
		acc:         make(map[seriesKey]*statAcc),
		meta:        make(map[seriesKey]containerMeta),
		metaCh:      make(chan map[seriesKey]containerMeta, 1),
		window:      statsWindow,
	}
}

// run blocks until ctx is cancelled, folding samples into per-container
// accumulators and flushing one batch per window.
func (ss *statsStreamer) run(ctx context.Context) {
	samples := make(chan StatSample, statsSampleChanBuf)
	ss.hostService.SubscribeStats(ctx, samples)

	// Prime the metadata map so the first window can already resolve names
	// rather than spending its grace tick on a cold cache. Async, like every
	// later refresh — see startMetaRefresh for why this must never block.
	ss.startMetaRefresh(ctx)

	ticker := time.NewTicker(ss.window)
	defer ticker.Stop()

	log.Debug().Dur("window", ss.window).Msg("stats streamer: started")
	defer log.Debug().Msg("stats streamer: stopped")

	for {
		select {
		case <-ctx.Done():
			return
		case s := <-samples:
			ss.observe(s)
		case m := <-ss.metaCh:
			ss.meta = m
			ss.refreshing = false
		case now := <-ticker.C:
			// Kick the next refresh before flushing so the snapshot is as fresh
			// as possible for the NEXT window; this one flushes against the
			// last-good snapshot.
			ss.startMetaRefresh(ctx)
			if err := ss.flush(now); err != nil {
				log.Debug().Err(err).Msg("stats streamer: send failed")
				return
			}
		}
	}
}

// observe folds one raw sample into its accumulator. Samples for a container
// the user disabled are dropped here so they never occupy memory.
func (ss *statsStreamer) observe(s StatSample) {
	key := seriesKey{hostID: s.HostID, containerID: s.Stat.ID}
	if m, ok := ss.meta[key]; ok && m.disabled {
		return
	}

	a, ok := ss.acc[key]
	if !ok {
		a = &statAcc{}
		ss.acc[key] = a
	}

	a.cpuSum += s.Stat.CPUPercent
	a.cpuMax = max(a.cpuMax, s.Stat.CPUPercent)
	a.memSum += s.Stat.MemoryPercent
	a.memMax = max(a.memMax, s.Stat.MemoryPercent)
	a.memUsageSum += s.Stat.MemoryUsage
	a.samples++

	// Counters are cumulative since container start — keep the newest.
	a.networkRx = s.Stat.NetworkRxTotal
	a.networkTx = s.Stat.NetworkTxTotal
	a.diskRead = s.Stat.DiskReadTotal
	a.diskWrite = s.Stat.DiskWriteTotal
}

// flush converts every resolvable accumulator into a wire entry and pushes them
// as one batch, against the last-good metadata snapshot. Emitted and abandoned
// accumulators are removed, so a container that stops simply drops out of the
// next window.
//
// Deliberately does NO I/O: it runs on the same goroutine that drains the sample
// channel, and blocking here backs up into the stats collector (see
// startMetaRefresh).
func (ss *statsStreamer) flush(now time.Time) error {
	entries := make([]*pb.StatsBatchEntry, 0, len(ss.acc))
	tsNs := now.UnixNano()

	for key, a := range ss.acc {
		if a.samples == 0 {
			delete(ss.acc, key)
			continue
		}

		m, ok := ss.meta[key]
		if !ok {
			// Container hasn't shown up in a snapshot yet (started mid-window)
			// or has already gone. Hold the samples for one more window, then
			// give up rather than accumulate forever.
			a.staleTicks++
			if a.staleTicks >= statsMaxStaleTicks {
				delete(ss.acc, key)
			}
			continue
		}
		delete(ss.acc, key)
		if m.disabled {
			continue
		}

		n := float64(a.samples)
		cores := m.cores
		if cores <= 0 {
			cores = 1
		}
		entries = append(entries, &pb.StatsBatchEntry{
			HostId:        key.hostID,
			ContainerName: m.name,
			TimestampNs:   tsNs,
			// Stat.CPUPercent is per-core (100% == one full core). Normalise by
			// the core count so Cloud sees overall load, matching the UI and the
			// metric-alert path in internal/notification/processing.go.
			CpuPercent:       a.cpuSum / n / cores,
			CpuPercentMax:    a.cpuMax / cores,
			MemoryPercent:    a.memSum / n,
			MemoryPercentMax: a.memMax,
			MemoryUsageBytes: a.memUsageSum / n,
			NetworkRxTotal:   a.networkRx,
			NetworkTxTotal:   a.networkTx,
			DiskReadTotal:    a.diskRead,
			DiskWriteTotal:   a.diskWrite,
			Samples:          a.samples,
			CpuCores:         cores,
		})
	}

	if len(entries) == 0 {
		return nil
	}

	// Sort before any truncation so a host over the cap reports a stable subset
	// rather than a different random slice every window.
	slices.SortFunc(entries, func(a, b *pb.StatsBatchEntry) int {
		if c := cmp.Compare(a.HostId, b.HostId); c != 0 {
			return c
		}
		return cmp.Compare(a.ContainerName, b.ContainerName)
	})
	if len(entries) > statsMaxEntries {
		log.Warn().
			Int("total", len(entries)).
			Int("sent", statsMaxEntries).
			Msg("stats streamer: batch exceeds entry cap, truncating")
		entries = entries[:statsMaxEntries]
	}

	return ss.send(&pb.ToolResponse{Type: &pb.ToolResponse_StatsBatch{StatsBatch: &pb.StatsBatch{Entries: entries}}})
}

// startMetaRefresh rebuilds the metadata snapshot on its own goroutine and
// hands the result back over metaCh.
//
// This MUST stay off run()'s goroutine. ListAllContainers walks every local host
// sequentially with a 5s timeout each, so one slow or unreachable peer would
// stall the sample drain for N*5s. That stall does not stay local: the stats
// collector's dispatch loop blocks on `stats <- stat` for each subscriber in
// turn (internal/docker/stats_collector.go), so once our 512-deep sample buffer
// filled, a wedged cloud consumer would starve the LIVE UI stats broadcast on
// that host too. Metrics shipping must never be able to degrade the UI.
//
// At most one refresh is in flight; if the host service is slower than the
// window we keep serving the last-good snapshot rather than queueing work.
func (ss *statsStreamer) startMetaRefresh(ctx context.Context) {
	if ss.refreshing {
		return
	}
	ss.refreshing = true
	go func() {
		m := ss.buildMeta()
		select {
		case ss.metaCh <- m:
		case <-ctx.Done():
		}
	}()
}

// refreshMeta rebuilds the snapshot synchronously. Only for tests and callers
// that are not on the sample-draining goroutine.
func (ss *statsStreamer) refreshMeta() {
	ss.meta = ss.buildMeta()
}

// buildMeta resolves every container's name, CPU divisor and opt-out state.
// Called once per window rather than per sample — it is a host round-trip, and
// there are ~30 samples per container per window.
func (ss *statsStreamer) buildMeta() map[seriesKey]containerMeta {
	containers, errs := ss.hostService.ListAllContainers(ss.labels)
	for _, err := range errs {
		if err != nil {
			log.Debug().Err(err).Msg("stats streamer: error listing containers from host")
		}
	}

	ncpuByHost := make(map[string]int)
	for _, h := range ss.hostService.Hosts() {
		ncpuByHost[h.ID] = h.NCPU
	}

	meta := make(map[seriesKey]containerMeta, len(containers))
	for _, c := range containers {
		// A container the user opted out of log streaming with the min_level
		// label shouldn't quietly keep reporting metrics either.
		_, disabled, valid := parseMinLevel(c.Labels[cloudMinLevelLabel])
		if !valid {
			// logStreamer already logs the invalid value loudly for this
			// container; don't double-report it every 30 seconds.
			disabled = false
		}

		cores := c.CPULimit
		if cores <= 0 {
			cores = float64(ncpuByHost[c.Host])
		}
		if cores <= 0 {
			cores = 1
		}

		meta[seriesKey{hostID: c.Host, containerID: c.ID}] = containerMeta{
			name:     c.Name,
			cores:    cores,
			disabled: disabled,
		}
	}
	return meta
}

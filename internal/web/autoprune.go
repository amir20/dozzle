package web

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/amir20/dozzle/internal/config"
	"github.com/amir20/dozzle/internal/container"
	"github.com/rs/zerolog/log"
)

// autoPruneOptions is what a scheduled clean up deletes. Never volumes: an
// unused named volume can be a database between redeploys, and the popover
// only deletes one after someone ticks a box nobody is there to tick at 4am.
var autoPruneOptions = container.PruneOptions{Images: true, Containers: true, BuildCache: true}

// autoPruneRun is the last scheduled clean up, kept in memory for the
// reclaimable popover. It is gone after a restart, which is fine: it only
// says the schedule is doing something.
type autoPruneRun struct {
	At             time.Time `json:"at"`
	SpaceReclaimed uint64    `json:"spaceReclaimed"`
	// Failed counts hosts the run could not clean, whose errors are logged.
	Failed int `json:"failed,omitempty"`
}

var autoPruneLast struct {
	sync.Mutex
	run *autoPruneRun
}

func lastAutoPrune() *autoPruneRun {
	autoPruneLast.Lock()
	defer autoPruneLast.Unlock()
	return autoPruneLast.run
}

// autoPruneFrom reads the clean up schedule out of a dozzle.yml the caller
// already read. It shares autoUpdateSettings, and with it due. Anything
// invalid falls back to the default rather than failing.
func autoPruneFrom(file config.File) autoUpdateSettings {
	s := autoUpdateSettings{Mode: config.AutoUpdateOff, Time: config.DefaultAutoPruneTime}
	if file.AutoPrune != nil && config.ValidAutoUpdateMode(*file.AutoPrune) {
		s.Mode = *file.AutoPrune
	}
	if file.AutoPruneTime != nil && config.ValidAutoUpdateTime(*file.AutoPruneTime) {
		s.Time = *file.AutoPruneTime
	}
	return s
}

// autoPruneScheduler checks once a minute whether it is time to clean up every
// host. dozzle.yml is read on every tick, so a schedule saved from the popover
// applies without a restart.
type autoPruneScheduler struct {
	hostService HostService
	now         func() time.Time
	after       func(time.Duration) <-chan time.Time
	lastRun     string
}

// RunAutoPruneScheduler blocks until ctx is done. Like auto-update it runs in
// server mode only: in swarm every replica would clean every node, and k8s has
// nothing to prune.
func RunAutoPruneScheduler(ctx context.Context, hostService HostService, cfg Config) {
	if cfg.Mode != "server" || !cfg.EnableActions {
		log.Debug().Str("mode", cfg.Mode).Bool("actions", cfg.EnableActions).Msg("auto prune: scheduler not started")
		return
	}
	s := &autoPruneScheduler{hostService: hostService, now: time.Now, after: time.After}
	s.run(ctx)
}

func (s *autoPruneScheduler) run(ctx context.Context) {
	for {
		// Wake just past each minute boundary, so no minute is skipped or seen twice.
		now := s.now()
		next := now.Truncate(time.Minute).Add(time.Minute + time.Second)
		select {
		case <-ctx.Done():
			return
		case <-s.after(next.Sub(now)):
		}
		s.tick(ctx, s.now())
	}
}

func (s *autoPruneScheduler) tick(ctx context.Context, now time.Time) {
	file, err := config.Load(setupConfigPath)
	if err != nil {
		log.Warn().Err(err).Msg("auto prune: could not read dozzle.yml")
	}
	if !autoPruneFrom(file).due(now) {
		return
	}
	day := now.Format("2006-01-02")
	if s.lastRun == day {
		return
	}
	s.lastRun = day

	// A prune in the middle of an update could delete the image it just pulled
	// before the new container holds it.
	select {
	case <-bulkUpdates.idle():
	case <-ctx.Done():
		return
	}

	run := s.pruneAll(ctx)
	run.At = now
	autoPruneLast.Lock()
	autoPruneLast.run = &run
	autoPruneLast.Unlock()
}

// pruneAll cleans every available host in turn and adds up what they freed. A
// host that fails is logged and counted, and the rest still run.
func (s *autoPruneScheduler) pruneAll(ctx context.Context) autoPruneRun {
	var run autoPruneRun
	pruner, ok := s.hostService.(hostPruner)
	if !ok {
		return run
	}
	for _, host := range s.hostService.Hosts() {
		if !host.Available {
			continue
		}
		pruneCtx, cancel := context.WithTimeout(ctx, pruneTimeout)
		result, _, err := pruner.PruneHost(pruneCtx, host.ID, autoPruneOptions)
		cancel()
		switch {
		case errors.Is(err, errors.ErrUnsupported):
			log.Debug().Str("host", host.Name).Msg("auto prune: host skipped, it cannot prune")
		case err != nil:
			run.Failed++
			log.Warn().Err(err).Str("host", host.Name).Msg("auto prune: could not clean up host")
		default:
			run.SpaceReclaimed += result.SpaceReclaimed
			log.Info().Str("host", host.Name).Uint64("freed", result.SpaceReclaimed).Msg("auto prune: host cleaned up")
		}
	}
	return run
}

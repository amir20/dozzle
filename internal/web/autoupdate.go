package web

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/amir20/dozzle/internal/config"
	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/imagecheck"
	"github.com/amir20/dozzle/internal/selfupdate"
	docker_types "github.com/moby/moby/api/types/container"
	"github.com/rs/zerolog/log"
)

// Reasons auto-update is unsupported, as GET /api/setup reports them. All but
// actions-off come from selfupdate. A --rm container is supported: the
// replacement holds every volume by name before the old one stops.
const (
	autoUpdateNotServer   = selfupdate.ReasonNotServer
	autoUpdateNoContainer = selfupdate.ReasonNoContainer
	autoUpdatePinnedTag   = selfupdate.ReasonPinnedTag
	autoUpdateActionsOff  = "actions-off"
)

// selfImage is what auto-update needs to know about Dozzle's own container.
type selfImage struct {
	// Ref is the reference the container was created from, e.g. amir20/dozzle:latest.
	Ref     string
	ImageID string
	// Swarm is true for a swarm service task, which selfupdate.Start refuses.
	Swarm       bool
	RepoDigests []string
}

// Seams for tests, so nothing here reaches docker or a registry.
var (
	selfUpdateStart   = selfupdate.Start
	selfUpdateInspect = inspectSelf
	selfUpdateCheck   = func(ctx context.Context, image string, digests []string) imagecheck.Result {
		// Forced: this runs at most once a day, and a six hour old digest would
		// quietly push the update to the next one.
		return imagecheck.Shared().Check(ctx, image, digests, true)
	}
)

// selfUpdateMu stops a manual update and the scheduler from launching two
// helpers at once.
var selfUpdateMu sync.Mutex

type selfInspector interface {
	ContainerInspect(ctx context.Context, containerID string) (docker_types.InspectResponse, error)
	ImageRepoDigests(ctx context.Context, imageID string) ([]string, error)
}

// inspectSelf goes straight to the local docker clients rather than
// FindContainer, whose label filters may well exclude Dozzle itself.
func inspectSelf(ctx context.Context, hostService HostService, id string) (selfImage, error) {
	if hostService == nil {
		return selfImage{}, errors.New("no host service")
	}
	var lastErr error = errors.New("own container not found on any local client")
	for _, client := range hostService.LocalClients() {
		inspector, ok := client.(selfInspector)
		if !ok {
			continue
		}
		inspect, err := inspector.ContainerInspect(ctx, id)
		if err != nil {
			lastErr = err
			continue
		}
		self := selfImage{ImageID: inspect.Image}
		if inspect.Config != nil {
			self.Ref = selfupdate.ImageRef(inspect.Config)
			self.Swarm = selfupdate.SwarmTask(inspect.Config.Labels)
		}
		if digests, err := inspector.ImageRepoDigests(ctx, inspect.Image); err == nil {
			self.RepoDigests = digests
		}
		return self, nil
	}
	return selfImage{}, lastErr
}

// pinnedReference reports whether pulling ref again can never bring a newer
// image. Floating tags like latest or v8 still move.
func pinnedReference(ref string) bool {
	return selfupdate.Pinned(ref)
}

type autoUpdateSettings struct {
	Mode string
	Time string
}

// effectiveAutoUpdate reads dozzle.yml, with flag and env values winning.
// Anything invalid falls back to the default rather than failing.
func effectiveAutoUpdate(setup SetupConfig) (autoUpdateSettings, error) {
	file, err := config.Load(setupConfigPath)
	s := autoUpdateSettings{Mode: config.AutoUpdateOff, Time: config.DefaultAutoUpdateTime}
	if file.AutoUpdate != nil {
		s.Mode = *file.AutoUpdate
	}
	if file.AutoUpdateTime != nil {
		s.Time = *file.AutoUpdateTime
	}
	if setup.AutoUpdateMode != nil {
		s.Mode = *setup.AutoUpdateMode
	}
	if setup.AutoUpdateTime != nil {
		s.Time = *setup.AutoUpdateTime
	}
	if !config.ValidAutoUpdateMode(s.Mode) {
		s.Mode = config.AutoUpdateOff
	}
	if !config.ValidAutoUpdateTime(s.Time) {
		s.Time = config.DefaultAutoUpdateTime
	}
	return s, err
}

type autoUpdateSupport struct {
	Supported bool
	Reason    string
	Image     string
	self      selfImage
	selfID    string
}

func checkAutoUpdateSupport(ctx context.Context, cfg *Config, hostService HostService) autoUpdateSupport {
	if cfg.Mode != "server" {
		return autoUpdateSupport{Reason: autoUpdateNotServer}
	}
	id := setupSelfID()
	if id == "" {
		return autoUpdateSupport{Reason: autoUpdateNoContainer}
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	self, err := selfUpdateInspect(ctx, hostService, id)
	if err != nil {
		log.Debug().Err(err).Str("container", id).Msg("auto update: could not inspect own container")
		return autoUpdateSupport{Reason: autoUpdateNoContainer}
	}
	s := autoUpdateSupport{Image: self.Ref, self: self, selfID: id}
	switch {
	case self.Swarm:
		s.Reason = autoUpdateNotServer
	case !cfg.EnableActions:
		s.Reason = autoUpdateActionsOff
	case pinnedReference(self.Ref):
		s.Reason = autoUpdatePinnedTag
	default:
		s.Supported = true
	}
	return s
}

// autoUpdateScheduler checks once a minute whether it is time to update
// Dozzle's own container. dozzle.yml is read on every tick, so a schedule saved
// from the wizard applies without a restart.
type autoUpdateScheduler struct {
	config      *Config
	hostService HostService
	now         func() time.Time
	after       func(time.Duration) <-chan time.Time
	lastRun     string
}

// RunAutoUpdateScheduler blocks until ctx is done. It does nothing outside
// server mode or without actions, both of which are fixed for the process.
func RunAutoUpdateScheduler(ctx context.Context, hostService HostService, cfg Config) {
	if cfg.Mode != "server" || !cfg.EnableActions {
		log.Debug().Str("mode", cfg.Mode).Bool("actions", cfg.EnableActions).Msg("auto update: scheduler not started")
		return
	}
	s := &autoUpdateScheduler{config: &cfg, hostService: hostService, now: time.Now, after: time.After}
	s.run(ctx)
}

func (s *autoUpdateScheduler) run(ctx context.Context) {
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

// due reports whether settings schedule an update in now's minute.
func (settings autoUpdateSettings) due(now time.Time) bool {
	if settings.Mode == config.AutoUpdateOff {
		return false
	}
	if settings.Mode == config.AutoUpdateWeekly && now.Weekday() != time.Sunday {
		return false
	}
	return now.Format("15:04") == settings.Time
}

func (s *autoUpdateScheduler) tick(ctx context.Context, now time.Time) {
	settings, err := effectiveAutoUpdate(s.config.Setup)
	if err != nil {
		log.Warn().Err(err).Msg("auto update: could not read dozzle.yml")
	}
	if !settings.due(now) {
		return
	}
	day := now.Format("2006-01-02")
	if s.lastRun == day {
		log.Debug().Str("day", day).Msg("auto update: already ran today")
		return
	}
	s.lastRun = day

	support := checkAutoUpdateSupport(ctx, s.config, s.hostService)
	if !support.Supported {
		log.Debug().Str("reason", support.Reason).Msg("auto update: skipped, not supported")
		return
	}

	checkCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	result := selfUpdateCheck(checkCtx, support.self.Ref, support.self.RepoDigests)
	cancel()
	if !result.UpdateAvailable() {
		if result.Status == imagecheck.StatusUpToDate {
			// Whatever was attempted last stuck.
			_ = os.Remove(autoUpdateAttemptPath())
		}
		log.Debug().Str("image", support.self.Ref).Str("status", string(result.Status)).Str("reason", result.Reason).Msg("auto update: no update available")
		return
	}

	// A scheduled update that was rolled back leaves the tag on the broken image,
	// so every later tick would see the same update and repeat the outage. The
	// remote digest is written down before launching; still being offered that
	// digest afterwards means the attempt did not stick.
	attempt := autoUpdateAttemptPath()
	if result.RemoteDigest != "" {
		if prev, err := os.ReadFile(attempt); err == nil && strings.TrimSpace(string(prev)) == result.RemoteDigest {
			log.Warn().Str("image", support.self.Ref).Str("remote", result.RemoteDigest).Msg("auto update: skipped, an update to this image already failed and was rolled back")
			return
		}
		if err := os.WriteFile(attempt, []byte(result.RemoteDigest+"\n"), 0644); err != nil {
			log.Warn().Err(err).Msg("auto update: could not record the attempted image")
		}
	}

	log.Info().Str("image", support.self.Ref).Str("remote", result.RemoteDigest).Msg("auto update: newer image available, updating dozzle")
	updated, err := runSelfUpdate(ctx, support.selfID, func(p container.UpdateProgress) {
		if p.Status == "error" {
			log.Error().Str("error", p.Error).Msg("auto update: progress")
		} else if p.Status != "pulling" {
			log.Debug().Str("status", p.Status).Msg("auto update: progress")
		}
	})
	if !updated {
		// No helper ran, so nothing was rolled back: the next tick may try again.
		_ = os.Remove(attempt)
	}
	switch {
	case err != nil:
		log.Error().Err(err).Msg("auto update failed")
	case updated:
		log.Info().Msg("auto update: helper launched, dozzle will restart on the new image")
	default:
		log.Info().Msg("auto update: already up to date after pulling")
	}
}

// autoUpdateAttemptPath holds the remote digest of the last scheduled update
// that launched a helper, next to dozzle.yml.
func autoUpdateAttemptPath() string {
	return filepath.Join(filepath.Dir(setupConfigPath), "auto-update-attempt")
}

var errSelfUpdateBusy = errors.New("an update of dozzle is already in progress")

// runSelfUpdate serializes calls to selfupdate.Start.
func runSelfUpdate(ctx context.Context, id string, progress func(container.UpdateProgress)) (bool, error) {
	if !selfUpdateMu.TryLock() {
		return false, errSelfUpdateBusy
	}
	defer selfUpdateMu.Unlock()
	updated, err := selfUpdateStart(ctx, id, progress)
	if err != nil {
		return updated, fmt.Errorf("self update: %w", err)
	}
	return updated, nil
}

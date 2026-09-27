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

	"github.com/amir20/dozzle/internal/analytics"
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
	autoUpdateSwarmWorker = selfupdate.ReasonSwarmWorker
	autoUpdateActionsOff  = "actions-off"
)

// selfImage is what auto-update needs to know about Dozzle's own container.
type selfImage struct {
	// Ref is the reference the container follows, e.g. amir20/dozzle:latest. For
	// a swarm task any digest swarm pinned it to is dropped.
	Ref     string
	ImageID string
	// Swarm is true for a swarm service task, which updates through the manager.
	Swarm bool
	// ServiceID is the swarm service a task belongs to.
	ServiceID string
	// SecondaryReplica is a swarm replica other than the first, which leaves the
	// schedule to that one so the service is not rolled once per replica.
	SecondaryReplica bool
	RepoDigests      []string
}

// Seams for tests, so nothing here reaches docker or a registry.
var (
	selfUpdateStart        = selfupdate.Start
	selfUpdateSwarmManager = selfupdate.SwarmManager
	selfUpdateInspect      = inspectSelf
	selfUpdateCheck        = func(ctx context.Context, image string, digests []string, force bool) imagecheck.Result {
		return imagecheck.Shared().Check(ctx, image, digests, force)
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
			self.Ref = selfupdate.SelfRef(inspect.Config)
			self.Swarm = selfupdate.SwarmTask(inspect.Config.Labels)
			if self.Swarm {
				self.ServiceID = selfupdate.SwarmServiceID(inspect.Config.Labels)
				self.SecondaryReplica = !selfupdate.SwarmPrimary(inspect.Config.Labels)
			}
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
	return autoUpdateFrom(setup, file), err
}

// autoUpdateFrom is effectiveAutoUpdate for a dozzle.yml the caller already read.
func autoUpdateFrom(setup SetupConfig, file config.File) autoUpdateSettings {
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
	return s
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
	case self.Swarm && !selfUpdateSwarmManager(ctx, self.ServiceID):
		s.Reason = autoUpdateSwarmWorker
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
	// flushUsage sends the day's usage before an update replaces the process.
	flushUsage func()
}

// RunAutoUpdateScheduler blocks until ctx is done. It does nothing outside
// server mode or without actions, both of which are fixed for the process.
// flushUsage is Server.FlushUsage.
func RunAutoUpdateScheduler(ctx context.Context, hostService HostService, cfg Config, flushUsage func()) {
	if cfg.Mode != "server" || !cfg.EnableActions {
		log.Debug().Str("mode", cfg.Mode).Bool("actions", cfg.EnableActions).Msg("auto update: scheduler not started")
		return
	}
	s := &autoUpdateScheduler{config: &cfg, hostService: hostService, now: time.Now, after: time.After, flushUsage: flushUsage}
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

	// Labelled containers first: updating Dozzle ends this process.
	s.updateLabelledContainers(ctx)

	support := checkAutoUpdateSupport(ctx, s.config, s.hostService)
	if !support.Supported {
		log.Debug().Str("reason", support.Reason).Msg("auto update: skipped, not supported")
		return
	}
	if support.self.SecondaryReplica {
		log.Debug().Str("service", support.self.ServiceID).Msg("auto update: skipped, another replica of this service runs the schedule")
		return
	}

	checkCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	// Forced: this runs at most once a day, and a six hour old digest would
	// quietly push the update to the next one.
	result := selfUpdateCheck(checkCtx, support.self.Ref, support.self.RepoDigests, true)
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
	updated, err := runSelfUpdate(ctx, support.selfID, s.flushUsage, func(p container.UpdateProgress) {
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

// AutoUpdateLabel opts a container into the auto-update schedule. It is opt in
// on purpose: a database on a floating tag should never move on its own.
const AutoUpdateLabel = "dev.dozzle.auto-update"

func autoUpdateEnabled(labels map[string]string) bool {
	switch strings.ToLower(strings.TrimSpace(labels[AutoUpdateLabel])) {
	case "true", "on", "yes", "1":
		return true
	default:
		return false
	}
}

// updateLabelledContainers updates every labelled container whose registry
// serves a newer image, and waits for them to finish. Nothing is pulled for a
// container that is up to date: the check is a HEAD request that does not count
// against Docker Hub's rate limit, and a pull does.
func (s *autoUpdateScheduler) updateLabelledContainers(ctx context.Context) {
	if s.hostService == nil {
		return
	}
	// A manual update that is running is waited out rather than costing the
	// labelled containers a whole day. The list is only taken afterwards: that
	// job may have recreated some of them under new ids. One can still start
	// between the wait and Start, so a busy updater means wait again.
	for range 5 {
		select {
		case <-bulkUpdates.idle():
		case <-ctx.Done():
			return
		}
		outdated, selfService := s.outdatedLabelledContainers(ctx)
		if len(outdated) == 0 {
			return
		}
		done, err := bulkUpdates.Start(outdated, "schedule", selfService, "", s.flushUsage)
		if errors.Is(err, errBulkUpdateBusy) {
			continue
		}
		if err != nil {
			log.Warn().Err(err).Msg("auto update: skipped containers")
			return
		}
		log.Info().Int("count", len(outdated)).Msg("auto update: updating labelled containers")
		select {
		case <-done:
		case <-ctx.Done():
		}
		return
	}
	log.Warn().Msg("auto update: skipped containers, other updates kept running")
}

// outdatedLabelledContainers returns the labelled containers with a newer
// image, and Dozzle's own swarm service for Start.
func (s *autoUpdateScheduler) outdatedLabelledContainers(ctx context.Context) ([]*container.ContainerService, string) {
	containers, errs := s.hostService.ListAllContainers(s.config.Labels)
	for _, err := range errs {
		log.Warn().Err(err).Msg("auto update: host unavailable, its containers are skipped")
	}

	selfService := selfSwarmService(containers)
	var outdated []*container.ContainerService
	for _, c := range containers {
		if c.State == "deleted" || !autoUpdateEnabled(c.Labels) {
			continue
		}
		// Dozzle's own container follows the schedule by itself, with the
		// rollback guard below. A labelled replica of its swarm service would
		// roll this one too, in the middle of everything else.
		if isSelfContainer(c, selfService) {
			continue
		}
		service, err := s.hostService.FindContainer(c.Host, c.ID, s.config.Labels)
		if err != nil {
			continue
		}
		checkCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		// Forced, for the same reason as Dozzle's own check below.
		result, err := service.CheckImageUpdate(checkCtx, true)
		cancel()
		if err != nil || !result.UpdateAvailable() {
			log.Debug().Err(err).Str("container", c.Name).Str("status", string(result.Status)).Msg("auto update: container not updated")
			continue
		}
		outdated = append(outdated, service)
	}
	return outdated, selfService
}

// autoUpdateAttemptPath holds the remote digest of the last scheduled update
// that launched a helper, next to dozzle.yml.
func autoUpdateAttemptPath() string {
	return filepath.Join(filepath.Dir(setupConfigPath), "auto-update-attempt")
}

var errSelfUpdateBusy = errors.New("an update of dozzle is already in progress")

// runSelfUpdate serializes calls to selfupdate.Start. flushUsage, when set, sends
// the counted usage before the container is replaced.
func runSelfUpdate(ctx context.Context, id string, flushUsage func(), progress func(container.UpdateProgress)) (bool, error) {
	if !selfUpdateMu.TryLock() {
		return false, errSelfUpdateBusy
	}
	defer selfUpdateMu.Unlock()
	updated, err := selfUpdateStart(ctx, id, func(p container.UpdateProgress) {
		// "recreating" is the point where a newer image was pulled and the helper is
		// about to replace this container, usually without a clean shutdown, so it is
		// the last chance for the day's counters, this update included. An image that
		// is already current or a failed pull never gets here.
		if p.Status == "recreating" {
			analytics.Count("image.update")
			if flushUsage != nil {
				flushUsage()
			}
		}
		if progress != nil {
			progress(p)
		}
	})
	if err != nil {
		return updated, fmt.Errorf("self update: %w", err)
	}
	return updated, nil
}

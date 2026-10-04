// Package swap replaces a container with one built from its own config, and
// puts the old one back if the replacement does not come up. Self-update and
// the ordinary container Update action both recreate through it.
//
// The order is rename old, create new, stop old, start new, verify, remove old.
// Only a running container is swapped: one that is stopped is refused with
// container.ErrNotRunning, since starting its replacement would run something
// someone stopped.
// Creating before stopping is what makes --rm containers safe: the engine
// removes a --rm container on stop and deletes its anonymous volumes with it,
// but skips any volume another container still references (checked against
// Docker 29: the volume survives). The replacement references every volume by
// name, and a container that names a volume never deletes it on removal, so
// the data outlives both the old container and a replacement that crashes.
// For a --rm container there is nothing to rename back after the stop, so the
// rollback recreates it from its inspect on its old image id.
package swap

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/amir20/dozzle/internal/container"
	cerrdefs "github.com/containerd/errdefs"
	dcontainer "github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/image"
	"github.com/moby/moby/client"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Timings are variables so tests do not wait on them.
var (
	// StableFor is how long the replacement has to stay up before the old
	// container is thrown away.
	StableFor = 10 * time.Second
	// HealthTimeout bounds the wait for a healthcheck to report healthy.
	HealthTimeout = 3 * time.Minute
	// GoneTimeout bounds the wait for a --rm container to be removed after stop.
	GoneTimeout  = 30 * time.Second
	PollInterval = time.Second
)

// API is the slice of the moby client a swap and its image cleanup use.
type API interface {
	ImageAPI
	ContainerInspect(ctx context.Context, containerID string, options client.ContainerInspectOptions) (client.ContainerInspectResult, error)
	ContainerCreate(ctx context.Context, options client.ContainerCreateOptions) (client.ContainerCreateResult, error)
	ContainerStart(ctx context.Context, containerID string, options client.ContainerStartOptions) (client.ContainerStartResult, error)
	ContainerStop(ctx context.Context, containerID string, options client.ContainerStopOptions) (client.ContainerStopResult, error)
	ContainerRename(ctx context.Context, containerID string, options client.ContainerRenameOptions) (client.ContainerRenameResult, error)
	ContainerRemove(ctx context.Context, containerID string, options client.ContainerRemoveOptions) (client.ContainerRemoveResult, error)
}

// Options tune one swap. The zero value replaces the container on whatever its
// tag resolves to now, keeping everything else.
type Options struct {
	// OldImage is the inspect of the image the old container runs. Settings it
	// supplied (env, labels, cmd, ...) are dropped so the new image's own
	// defaults apply. Nil keeps them.
	OldImage *image.InspectResponse
	// NetworkMode, when set, replaces the old container's network mode.
	NetworkMode string
	// Image, when set, is what the replacement runs instead of the reference
	// the old container follows: a rollback runs the previous image by id. The
	// reference itself is kept in ImageRefLabel, so the replacement still
	// follows its tag for update checks and the next update.
	Image string
	// Labels are set on the replacement only. An empty value removes the label.
	Labels map[string]string
	// OnVerifying is called once the replacement has started, before it is
	// watched for StableFor.
	OnVerifying func()
	// LogPrefix starts every log line, "update" when empty.
	LogPrefix string
}

// Result is what a swap did.
type Result struct {
	// NewID is the replacement, set once the swap committed.
	NewID string
	// RolledBack is true when the replacement failed and the old container
	// was put back. The error says why the replacement failed.
	RolledBack bool
	// RestoredID is the container running the old version after a rollback:
	// the old container itself, or a recreation of it for a --rm one.
	RestoredID string
	// OldStopped is true when the old container was stopped at some point.
	// Containers sharing its network namespace lost it then, even if a
	// rollback restarted it.
	OldStopped bool
}

type swap struct {
	cli     API
	old     dcontainer.InspectResponse
	opts    Options
	name    string
	tmpName string
	logger  zerolog.Logger
	prefix  string

	renamed    bool
	stopped    bool
	oldGone    bool
	newID      string
	restoredID string
}

// Swap replaces old with a container created from its config, verifies the
// replacement stays up, and only then removes old. If any step fails, the
// replacement is removed and old is put back. The rollback runs even when ctx
// is cancelled. old must be running, or nothing is touched and the error is
// container.ErrNotRunning.
func Swap(ctx context.Context, cli API, old dcontainer.InspectResponse, opts Options) (Result, error) {
	if old.Config == nil || old.HostConfig == nil {
		return Result{}, fmt.Errorf("inspect %s: incomplete container config", shortID(old.ID))
	}
	if !Running(old.State) {
		return Result{}, container.ErrNotRunning
	}
	s := newSwap(cli, old, opts)

	if err := s.forward(ctx); err != nil {
		s.logger.Error().Err(err).Msg(s.prefix + ": failed, rolling back")
		if rbErr := s.rollback(context.WithoutCancel(ctx)); rbErr != nil {
			s.logger.Error().Err(rbErr).Msg(s.prefix + ": ROLLBACK FAILED, manual recovery needed")
			return Result{OldStopped: s.stopped}, fmt.Errorf("%w; rollback failed: %v", err, rbErr)
		}
		s.logger.Info().Msg(s.prefix + ": rolled back to the previous container")
		return Result{RolledBack: true, RestoredID: s.restoredID, OldStopped: s.stopped}, err
	}
	return Result{NewID: s.newID, OldStopped: s.stopped}, nil
}

func newSwap(cli API, old dcontainer.InspectResponse, opts Options) *swap {
	name := strings.TrimPrefix(old.Name, "/")
	prefix := opts.LogPrefix
	if prefix == "" {
		prefix = "update"
	}
	return &swap{
		cli:     cli,
		old:     old,
		opts:    opts,
		name:    name,
		tmpName: OldName(name, old.ID),
		logger:  log.With().Str("container", name).Str("id", shortID(old.ID)).Logger(),
		prefix:  prefix,
	}
}

// spec is the container to create in place of old: on image when set, pinned
// by id with the reference it followed kept in ImageRefLabel, and with labels
// applied (an empty value removes one). oldImage is as Options.OldImage.
func (s *swap) spec(oldImage *image.InspectResponse, image string, labels map[string]string) client.ContainerCreateOptions {
	spec := ReplacementSpec(s.old, oldImage, s.name)
	if s.opts.NetworkMode != "" {
		spec.HostConfig.NetworkMode = dcontainer.NetworkMode(s.opts.NetworkMode)
	}
	if image != "" {
		if ref := spec.Config.Image; ref != "" && !IsImageID(ref) {
			if spec.Config.Labels == nil {
				spec.Config.Labels = map[string]string{}
			}
			spec.Config.Labels[ImageRefLabel] = ref
		}
		spec.Config.Image = image
	}
	for k, v := range labels {
		if v == "" {
			delete(spec.Config.Labels, k)
			continue
		}
		if spec.Config.Labels == nil {
			spec.Config.Labels = map[string]string{}
		}
		spec.Config.Labels[k] = v
	}
	return spec
}

func (s *swap) forward(ctx context.Context) error {
	s.logger.Info().Str("to", s.tmpName).Msg(s.prefix + ": renaming old container")
	if _, err := s.cli.ContainerRename(ctx, s.old.ID, client.ContainerRenameOptions{NewName: s.tmpName}); err != nil {
		return fmt.Errorf("rename old container: %w", err)
	}
	s.renamed = true

	s.logger.Info().Msg(s.prefix + ": creating replacement")
	created, err := s.cli.ContainerCreate(ctx, s.spec(s.opts.OldImage, s.opts.Image, s.opts.Labels))
	if err != nil {
		return fmt.Errorf("create replacement: %w", err)
	}
	s.newID = created.ID

	s.logger.Info().Msg(s.prefix + ": stopping old container")
	if _, err := s.cli.ContainerStop(ctx, s.old.ID, client.ContainerStopOptions{}); err != nil && !isNotFound(err) {
		// A stop that errors may still have stopped it, so the rollback starts
		// it unless it is plainly still running.
		s.stopped = !s.running(ctx, s.old.ID)
		return fmt.Errorf("stop old container: %w", err)
	}
	s.stopped = true

	if s.old.HostConfig.AutoRemove {
		if err := s.waitGone(ctx, s.old.ID); err != nil {
			return err
		}
		s.oldGone = true
	}

	s.logger.Info().Str("newId", shortID(s.newID)).Msg(s.prefix + ": starting replacement")
	if _, err := s.cli.ContainerStart(ctx, s.newID, client.ContainerStartOptions{}); err != nil {
		return fmt.Errorf("start replacement: %w", err)
	}

	if s.opts.OnVerifying != nil {
		s.opts.OnVerifying()
	}
	s.logger.Info().Dur("for", StableFor).Msg(s.prefix + ": waiting for replacement to stay up")
	if err := WaitStable(ctx, s.cli, s.newID); err != nil {
		return err
	}

	if !s.oldGone {
		s.removeOld(ctx)
	}
	return nil
}

// running reports whether the daemon says id is running. An inspect that
// fails reads as not running.
func (s *swap) running(ctx context.Context, id string) bool {
	result, err := s.cli.ContainerInspect(ctx, id, client.ContainerInspectOptions{})
	return err == nil && result.Container.State != nil && result.Container.State.Running
}

// removeOld commits the swap by removing the old container. Failing to is
// only logged: the update itself worked, and a leftover stopped container is
// untidy, not a reason to roll back.
func (s *swap) removeOld(ctx context.Context) {
	s.logger.Info().Msg(s.prefix + ": removing old container (volumes kept)")
	if _, err := s.cli.ContainerRemove(ctx, s.old.ID, client.ContainerRemoveOptions{}); err != nil && !isNotFound(err) {
		s.logger.Warn().Err(err).Str("old", s.tmpName).Msg(s.prefix + ": unable to remove old container, remove it by hand")
	}
}

func (s *swap) rollback(ctx context.Context) error {
	var errs []error

	// A --rm container whose removal outlasted the wait deletes its anonymous
	// volumes as it finishes, unless something else still references them. The
	// replacement is that something, so it stays until the old one is gone.
	if s.newID != "" && s.stopped && !s.oldGone && s.old.HostConfig.AutoRemove {
		if err := s.waitGone(ctx, s.old.ID); err != nil {
			return s.startOldUnderTmpName(ctx, fmt.Errorf("old container still being removed, keeping replacement %s so its volumes survive: %w", shortID(s.newID), err))
		}
		s.oldGone = true
	}

	if s.newID != "" {
		s.logger.Info().Str("newId", shortID(s.newID)).Msg("rollback: removing replacement (volumes kept)")
		if _, err := s.cli.ContainerRemove(ctx, s.newID, client.ContainerRemoveOptions{Force: true}); err != nil && !isNotFound(err) {
			// A --rm replacement that exited may already be removing itself
			// ("removal already in progress"), so wait for it to go before
			// giving up. Until it does the name is taken and nothing below can
			// succeed.
			if waitErr := s.waitGone(ctx, s.newID); waitErr != nil {
				return s.startOldUnderTmpName(ctx, fmt.Errorf("remove replacement: %w", err))
			}
		}
	}

	// Trust the engine over bookkeeping: a --rm container can finish removing
	// itself after a wait for it gave up.
	if !s.oldGone {
		if _, err := s.cli.ContainerInspect(ctx, s.old.ID, client.ContainerInspectOptions{}); isNotFound(err) {
			s.oldGone = true
		}
	}

	if s.oldGone {
		// A --rm container removed itself on stop. Rebuild it on the image it
		// was running; its volumes are still there, held by name.
		spec := s.spec(nil, s.old.Image, nil)
		s.logger.Info().Str("image", shortID(s.old.Image)).Msg("rollback: recreating old container")
		created, err := s.cli.ContainerCreate(ctx, spec)
		if err != nil {
			return fmt.Errorf("recreate old container: %w", err)
		}
		s.restoredID = created.ID
		if _, err := s.cli.ContainerStart(ctx, created.ID, client.ContainerStartOptions{}); err != nil {
			return fmt.Errorf("start recreated container: %w", err)
		}
		return nil
	}

	s.restoredID = s.old.ID
	if s.renamed {
		s.logger.Info().Msg("rollback: restoring old container name")
		if _, err := s.cli.ContainerRename(ctx, s.old.ID, client.ContainerRenameOptions{NewName: s.name}); err != nil {
			errs = append(errs, fmt.Errorf("rename old container back: %w", err))
		}
	}

	if s.stopped {
		s.logger.Info().Msg("rollback: starting old container")
		if _, err := s.cli.ContainerStart(ctx, s.old.ID, client.ContainerStartOptions{}); err != nil {
			errs = append(errs, fmt.Errorf("start old container: %w", err))
		}
	}
	return errors.Join(errs...)
}

// startOldUnderTmpName is for a rollback that has to give up while the
// replacement still holds the name: the old container, if it is still there,
// is started under its temporary name, so the service at least runs again.
func (s *swap) startOldUnderTmpName(ctx context.Context, err error) error {
	if !s.stopped || s.oldGone {
		return err
	}
	s.logger.Info().Str("name", s.tmpName).Msg("rollback: starting old container under its temporary name")
	if _, startErr := s.cli.ContainerStart(ctx, s.old.ID, client.ContainerStartOptions{}); startErr != nil {
		return fmt.Errorf("%w; start old container: %v", err, startErr)
	}
	return err
}

func (s *swap) waitGone(ctx context.Context, id string) error {
	deadline := time.Now().Add(GoneTimeout)
	for {
		_, err := s.cli.ContainerInspect(ctx, id, client.ContainerInspectOptions{})
		if isNotFound(err) {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("container %s was not removed within %s", shortID(id), GoneTimeout)
		}
		if err := sleep(ctx, PollInterval); err != nil {
			return err
		}
	}
}

// WaitStable requires the container to keep running, without restarting, for
// StableFor, and then to report healthy if it has a healthcheck.
func WaitStable(ctx context.Context, cli API, id string) error {
	started := ""
	check := func() (*dcontainer.State, error) {
		result, err := cli.ContainerInspect(ctx, id, client.ContainerInspectOptions{})
		if isNotFound(err) {
			return nil, fmt.Errorf("replacement exited and was removed")
		}
		if err != nil {
			return nil, fmt.Errorf("inspect replacement: %w", err)
		}
		state := result.Container.State
		if !Running(state) {
			if state == nil {
				return nil, fmt.Errorf("replacement is not running")
			}
			return nil, fmt.Errorf("replacement is not running (status %s, exit code %d%s)", state.Status, state.ExitCode, errSuffix(state.Error))
		}
		if started == "" {
			started = state.StartedAt
		} else if state.StartedAt != started {
			return nil, fmt.Errorf("replacement restarted")
		}
		return state, nil
	}

	deadline := time.Now().Add(StableFor)
	state, err := check()
	for err == nil && time.Now().Before(deadline) {
		if err = sleep(ctx, PollInterval); err == nil {
			state, err = check()
		}
	}
	if err != nil {
		return err
	}

	if state.Health == nil {
		return nil
	}
	deadline = time.Now().Add(HealthTimeout)
	for {
		switch state.Health.Status {
		case dcontainer.Healthy:
			return nil
		case dcontainer.Unhealthy:
			return fmt.Errorf("replacement is unhealthy")
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("replacement did not become healthy within %s", HealthTimeout)
		}
		if err := sleep(ctx, PollInterval); err != nil {
			return err
		}
		if state, err = check(); err != nil {
			return err
		}
		if state.Health == nil {
			return nil
		}
	}
}

// Running reports whether Swap treats a container as running: one that is not
// is refused.
func Running(state *dcontainer.State) bool {
	return state != nil && state.Running && !state.Restarting
}

func isNotFound(err error) bool {
	return err != nil && cerrdefs.IsNotFound(err)
}

func errSuffix(msg string) string {
	if msg == "" {
		return ""
	}
	return ": " + msg
}

func sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

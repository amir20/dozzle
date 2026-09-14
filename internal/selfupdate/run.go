package selfupdate

import (
	"context"
	"errors"
	"fmt"
	"time"

	dcontainer "github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/image"
	"github.com/moby/moby/client"
	"github.com/rs/zerolog/log"
)

// Timings are variables so tests do not wait on them.
var (
	// stableFor is how long the replacement has to stay up before the old
	// container is thrown away.
	stableFor = 10 * time.Second
	// healthTimeout bounds the wait for a healthcheck to report healthy.
	healthTimeout = 3 * time.Minute
	// goneTimeout bounds the wait for a --rm container to be removed after stop.
	goneTimeout  = 30 * time.Second
	pollInterval = time.Second
)

// Run recreates targetID from the image its tag now points to, keeping its
// config, networks and volumes, and puts the old container back if the new one
// does not come up.
//
// The order is rename old, create new, stop old, start new, verify, remove old.
// Creating before stopping is what makes --rm containers safe: the engine
// removes a --rm container on stop and deletes its anonymous volumes with it,
// but skips any volume another container still references (checked against
// Docker 29: the volume survives). The replacement references every volume by
// name, and a container that names a volume never deletes it on removal, so
// the data outlives both the old container and a replacement that crashes.
// For a --rm container there is nothing to rename back after the stop, so the
// rollback recreates it from its inspect on its old image id.
func Run(ctx context.Context, targetID string) error {
	cli, err := newClient(ctx)
	if err != nil {
		return err
	}
	defer cli.Close()
	return run(ctx, cli, targetID)
}

type swap struct {
	cli      dockerAPI
	old      dcontainer.InspectResponse
	oldImage *image.InspectResponse
	name     string
	tmpName  string

	renamed bool
	stopped bool
	oldGone bool
	newID   string
}

func run(ctx context.Context, cli dockerAPI, targetID string) error {
	result, err := cli.ContainerInspect(ctx, targetID, client.ContainerInspectOptions{})
	if err != nil {
		return fmt.Errorf("inspect %s: %w", targetID, err)
	}
	old := result.Container
	if old.Config == nil || old.HostConfig == nil {
		return fmt.Errorf("inspect %s: incomplete container config", targetID)
	}
	if old.Config.Labels[swarmLabel] != "" {
		return ErrSwarm
	}

	s := &swap{
		cli:     cli,
		old:     old,
		name:    trimName(old.Name),
		tmpName: oldName(trimName(old.Name), old.ID),
	}
	logger := log.With().Str("container", s.name).Str("id", shortID(old.ID)).Logger()

	ref := ImageRef(old.Config)
	newImage, err := cli.ImageInspect(ctx, ref)
	if err != nil {
		return fmt.Errorf("resolve image %s: %w", ref, err)
	}
	if newImage.ID == old.Image {
		logger.Info().Str("image", ref).Msg("self-update: already on the latest image, nothing to do")
		return nil
	}
	if oldImage, err := cli.ImageInspect(ctx, old.Image); err == nil {
		s.oldImage = &oldImage.InspectResponse
	} else {
		logger.Warn().Err(err).Msg("self-update: old image not inspectable, keeping its defaults in the new config")
	}

	logger.Info().
		Str("image", ref).
		Str("from", shortID(old.Image)).
		Str("to", shortID(newImage.ID)).
		Bool("autoRemove", old.HostConfig.AutoRemove).
		Msg("self-update: starting")

	if err := s.forward(ctx); err != nil {
		logger.Error().Err(err).Msg("self-update: failed, rolling back")
		if rbErr := s.rollback(context.WithoutCancel(ctx)); rbErr != nil {
			logger.Error().Err(rbErr).Msg("self-update: ROLLBACK FAILED, manual recovery needed")
			return fmt.Errorf("%w; rollback failed: %v", err, rbErr)
		}
		logger.Info().Msg("self-update: rolled back to the previous container")
		return err
	}

	logger.Info().Str("newId", shortID(s.newID)).Msg("self-update: done")
	return nil
}

func (s *swap) forward(ctx context.Context) error {
	logger := log.With().Str("container", s.name).Logger()

	logger.Info().Str("to", s.tmpName).Msg("self-update: renaming old container")
	if _, err := s.cli.ContainerRename(ctx, s.old.ID, client.ContainerRenameOptions{NewName: s.tmpName}); err != nil {
		return fmt.Errorf("rename old container: %w", err)
	}
	s.renamed = true

	logger.Info().Msg("self-update: creating replacement")
	created, err := s.cli.ContainerCreate(ctx, replacementSpec(s.old, s.oldImage, s.name))
	if err != nil {
		return fmt.Errorf("create replacement: %w", err)
	}
	s.newID = created.ID

	logger.Info().Msg("self-update: stopping old container")
	if _, err := s.cli.ContainerStop(ctx, s.old.ID, client.ContainerStopOptions{}); err != nil && !isNotFound(err) {
		return fmt.Errorf("stop old container: %w", err)
	}
	s.stopped = true

	if s.old.HostConfig.AutoRemove {
		if err := s.waitGone(ctx, s.old.ID); err != nil {
			return err
		}
		s.oldGone = true
	}

	logger.Info().Str("id", shortID(s.newID)).Msg("self-update: starting replacement")
	if _, err := s.cli.ContainerStart(ctx, s.newID, client.ContainerStartOptions{}); err != nil {
		return fmt.Errorf("start replacement: %w", err)
	}

	logger.Info().Dur("for", stableFor).Msg("self-update: waiting for replacement to stay up")
	if err := waitStable(ctx, s.cli, s.newID); err != nil {
		return err
	}

	if !s.oldGone {
		logger.Info().Msg("self-update: removing old container (volumes kept)")
		if _, err := s.cli.ContainerRemove(ctx, s.old.ID, client.ContainerRemoveOptions{}); err != nil && !isNotFound(err) {
			// The update itself worked; a leftover stopped container is untidy,
			// not a reason to roll back.
			logger.Warn().Err(err).Str("old", s.tmpName).Msg("self-update: unable to remove old container, remove it by hand")
		}
	}
	return nil
}

func (s *swap) rollback(ctx context.Context) error {
	var errs []error
	logger := log.With().Str("container", s.name).Logger()

	// A --rm container whose removal outlasted the wait deletes its anonymous
	// volumes as it finishes, unless something else still references them. The
	// replacement is that something, so it stays until the old one is gone.
	if s.newID != "" && s.stopped && !s.oldGone && s.old.HostConfig.AutoRemove {
		if err := s.waitGone(ctx, s.old.ID); err != nil {
			return fmt.Errorf("old container still being removed, keeping replacement %s so its volumes survive: %w", shortID(s.newID), err)
		}
		s.oldGone = true
	}

	if s.newID != "" {
		logger.Info().Str("id", shortID(s.newID)).Msg("rollback: removing replacement (volumes kept)")
		if _, err := s.cli.ContainerRemove(ctx, s.newID, client.ContainerRemoveOptions{Force: true}); err != nil && !isNotFound(err) {
			// A --rm replacement that exited may already be removing itself
			// ("removal already in progress"), so wait for it to go before
			// giving up. Until it does the name is taken and nothing below can
			// succeed.
			if waitErr := s.waitGone(ctx, s.newID); waitErr != nil {
				return fmt.Errorf("remove replacement: %w", err)
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
		spec := replacementSpec(s.old, nil, s.name)
		if ref := spec.Config.Image; !imageIDRef.MatchString(ref) {
			if spec.Config.Labels == nil {
				spec.Config.Labels = map[string]string{}
			}
			spec.Config.Labels[imageRefLabel] = ref
		}
		spec.Config.Image = s.old.Image
		logger.Info().Str("image", shortID(s.old.Image)).Msg("rollback: recreating old container")
		created, err := s.cli.ContainerCreate(ctx, spec)
		if err != nil {
			return fmt.Errorf("recreate old container: %w", err)
		}
		if _, err := s.cli.ContainerStart(ctx, created.ID, client.ContainerStartOptions{}); err != nil {
			return fmt.Errorf("start recreated container: %w", err)
		}
		return nil
	}

	if s.renamed {
		logger.Info().Msg("rollback: restoring old container name")
		if _, err := s.cli.ContainerRename(ctx, s.old.ID, client.ContainerRenameOptions{NewName: s.name}); err != nil {
			errs = append(errs, fmt.Errorf("rename old container back: %w", err))
		}
	}

	if s.stopped && running(s.old.State) {
		logger.Info().Msg("rollback: starting old container")
		if _, err := s.cli.ContainerStart(ctx, s.old.ID, client.ContainerStartOptions{}); err != nil {
			errs = append(errs, fmt.Errorf("start old container: %w", err))
		}
	}
	return errors.Join(errs...)
}

func (s *swap) waitGone(ctx context.Context, id string) error {
	deadline := time.Now().Add(goneTimeout)
	for {
		_, err := s.cli.ContainerInspect(ctx, id, client.ContainerInspectOptions{})
		if isNotFound(err) {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("container %s was not removed within %s", shortID(id), goneTimeout)
		}
		if err := sleep(ctx, pollInterval); err != nil {
			return err
		}
	}
}

// waitStable requires the container to keep running, without restarting, for
// stableFor, and then to report healthy if it has a healthcheck.
func waitStable(ctx context.Context, cli dockerAPI, id string) error {
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
		if !running(state) {
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

	deadline := time.Now().Add(stableFor)
	state, err := check()
	for err == nil && time.Now().Before(deadline) {
		if err = sleep(ctx, pollInterval); err == nil {
			state, err = check()
		}
	}
	if err != nil {
		return err
	}

	if state.Health == nil {
		return nil
	}
	deadline = time.Now().Add(healthTimeout)
	for {
		switch state.Health.Status {
		case dcontainer.Healthy:
			return nil
		case dcontainer.Unhealthy:
			return fmt.Errorf("replacement is unhealthy")
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("replacement did not become healthy within %s", healthTimeout)
		}
		if err := sleep(ctx, pollInterval); err != nil {
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

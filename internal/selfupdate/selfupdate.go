// Package selfupdate replaces the container Dozzle itself runs in with one on a
// newer image. A process cannot stop its own container and keep going, so the
// work is split in two: Start runs inside Dozzle and only pulls the image and
// launches a short-lived helper container from it; Run is what that helper
// executes (dozzle self-update) to swap the containers and roll back on failure.
package selfupdate

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/container/docker/swap"
	"github.com/amir20/dozzle/internal/imagecheck"
	"github.com/moby/moby/client"
	"github.com/rs/zerolog/log"
)

// Reasons Support gives for an unsupported self-update.
const (
	ReasonNotServer   = "not-server"
	ReasonNoContainer = "no-container"
	ReasonPinnedTag   = "pinned-tag"
)

// ErrSwarm is returned by Run for a swarm task: the helper never swaps one,
// since Start updates those through the swarm manager instead.
var ErrSwarm = errors.New("selfupdate: container is managed by a swarm service")

// versionTag matches a full release tag such as v8.12.0 or 8.12.0-beta. Those
// never move, so pulling them again can never produce anything newer.
var versionTag = regexp.MustCompile(`^v?\d+\.\d+\.\d+([-+].*)?$`)

// SwarmTask reports whether a container with these labels belongs to a swarm
// service, which Start updates through the manager rather than the helper.
func SwarmTask(labels map[string]string) bool {
	return labels[container.SwarmServiceNameLabel] != ""
}

// Pinned reports whether pulling ref again can never bring a newer image: a
// digest, a bare image id, or a full version tag.
func Pinned(ref string) bool {
	if swap.IsImageID(ref) {
		return true
	}
	parsed, err := imagecheck.ParseReference(ref)
	if err != nil {
		return true
	}
	return parsed.Pinned() || versionTag.MatchString(parsed.Tag)
}

// Support reports whether scheduled self-update can do anything for the
// container selfID, and the image reference it runs. Mode (server/swarm/k8s)
// and whether actions are enabled are for the caller to decide.
func Support(ctx context.Context, selfID string) (supported bool, reason string, image string) {
	if selfID == "" {
		return false, ReasonNoContainer, ""
	}
	cli, err := newClient(ctx)
	if err != nil {
		return false, ReasonNoContainer, ""
	}
	defer cli.Close()
	return support(ctx, cli, selfID)
}

func support(ctx context.Context, cli dockerAPI, selfID string) (bool, string, string) {
	result, err := cli.ContainerInspect(ctx, selfID, client.ContainerInspectOptions{})
	if err != nil || result.Container.Config == nil {
		return false, ReasonNoContainer, ""
	}
	self := result.Container
	image := SelfRef(self.Config)
	if SwarmTask(self.Config.Labels) && !swarmManager(ctx, cli, self.Config.Labels[container.SwarmServiceIDLabel]) {
		return false, ReasonSwarmWorker, image
	}
	if Pinned(image) {
		return false, ReasonPinnedTag, image
	}
	return true, "", image
}

// Start pulls the image that container selfID runs, reporting progress the same
// way container updates do. If the tag now resolves to a different image it
// launches the helper and returns updated=true; Dozzle goes away shortly after.
// It returns updated=false when the image is already current.
func Start(ctx context.Context, selfID string, progress func(container.UpdateProgress)) (updated bool, err error) {
	if progress == nil {
		progress = func(container.UpdateProgress) {}
	}
	cli, err := newClient(ctx)
	if err != nil {
		progress(container.UpdateProgress{Status: container.UpdateError, Error: err.Error()})
		return false, err
	}
	defer cli.Close()
	return start(ctx, cli, selfID, progress)
}

// startMu serializes every caller in this process (the wizard, the scheduler and
// the container update action), so two starts never race over the helper.
var startMu sync.Mutex

func start(ctx context.Context, cli dockerAPI, selfID string, progress func(container.UpdateProgress)) (bool, error) {
	fail := func(format string, args ...any) (bool, error) {
		err := fmt.Errorf(format, args...)
		progress(container.UpdateProgress{Status: container.UpdateError, Error: err.Error()})
		return false, err
	}

	if selfID == "" {
		return fail("self-update: Dozzle is not running in a container")
	}

	if !startMu.TryLock() {
		return fail("a self-update is already in progress")
	}
	defer startMu.Unlock()

	result, err := cli.ContainerInspect(ctx, selfID, client.ContainerInspectOptions{})
	if err != nil {
		return fail("inspect failed: %w", err)
	}
	self := result.Container
	if self.Config == nil {
		return fail("inspect failed: container has no config")
	}
	swarmTask := SwarmTask(self.Config.Labels)

	ref := SelfRef(self.Config)
	if swap.IsImageID(ref) {
		// Created from an image id: there is no tag to pull.
		progress(container.UpdateProgress{Status: container.UpdateUpToDate})
		return false, nil
	}

	reader, err := cli.ImagePull(ctx, ref, client.ImagePullOptions{})
	if err != nil {
		return fail("pull failed: %w", err)
	}
	err = swap.ReadPull(reader, progress)
	reader.Close()
	if err != nil {
		return fail("%w", err)
	}

	img, err := cli.ImageInspect(ctx, ref)
	if err != nil {
		return fail("unable to resolve pulled image: %w", err)
	}
	if img.ID == self.Image {
		log.Info().Str("image", ref).Msg("self-update: already running the latest image")
		progress(container.UpdateProgress{Status: container.UpdateUpToDate})
		return false, nil
	}

	if swarmTask {
		return updateService(ctx, cli, self.Config.Labels[container.SwarmServiceIDLabel], ref, progress)
	}

	progress(container.UpdateProgress{Status: container.UpdateRecreating})
	spec := helperSpec(self, img.ID)
	if err := launchHelper(ctx, cli, spec); err != nil {
		return fail("%w", err)
	}

	log.Info().Str("helper", spec.Name).Str("image", ref).Str("newImage", img.ID).Msg("self-update: helper started, Dozzle will be replaced shortly")
	progress(container.UpdateProgress{Status: container.UpdateDone})
	return true, nil
}

// StartRejoin launches the helper to recreate container selfID joined to
// networkMode, on the image it already runs. Dozzle calls it after recreating
// the container whose network namespace it shares, since its own namespace
// went with the old one.
func StartRejoin(ctx context.Context, selfID string, networkMode string) error {
	cli, err := newClient(ctx)
	if err != nil {
		return err
	}
	defer cli.Close()
	return startRejoin(ctx, cli, selfID, networkMode)
}

func startRejoin(ctx context.Context, cli dockerAPI, selfID string, networkMode string) error {
	if !startMu.TryLock() {
		return fmt.Errorf("a self-update is already in progress")
	}
	defer startMu.Unlock()

	result, err := cli.ContainerInspect(ctx, selfID, client.ContainerInspectOptions{})
	if err != nil {
		return fmt.Errorf("inspect failed: %w", err)
	}
	spec := helperSpec(result.Container, result.Container.Image)
	spec.Config.Cmd = append(spec.Config.Cmd, "--network", networkMode)
	if err := launchHelper(ctx, cli, spec); err != nil {
		return err
	}
	log.Info().Str("helper", spec.Name).Str("networkMode", networkMode).Msg("self-update: helper started, Dozzle will rejoin its network shortly")
	return nil
}

// launchHelper creates and starts the helper container spec describes.
func launchHelper(ctx context.Context, cli dockerAPI, spec client.ContainerCreateOptions) error {
	created, err := cli.ContainerCreate(ctx, spec)
	if isConflict(err) {
		// A helper by that name already exists. A running one is an update in
		// progress; anything else is left over and can go.
		existing, inspectErr := cli.ContainerInspect(ctx, spec.Name, client.ContainerInspectOptions{})
		if inspectErr == nil && existing.Container.State != nil && existing.Container.State.Running {
			return fmt.Errorf("a self-update is already in progress (%s)", spec.Name)
		}
		// Not forced: the engine refuses to remove a helper that started since.
		if _, rmErr := cli.ContainerRemove(ctx, spec.Name, client.ContainerRemoveOptions{}); rmErr != nil && !isNotFound(rmErr) {
			return fmt.Errorf("remove stale helper failed: %w", rmErr)
		}
		created, err = cli.ContainerCreate(ctx, spec)
	}
	if err != nil {
		return fmt.Errorf("create helper failed: %w", err)
	}

	// A client that disconnects must not cancel a start the daemon may already
	// have acted on, and a helper that did start is never force-removed.
	ctx = context.WithoutCancel(ctx)
	if _, err := cli.ContainerStart(ctx, created.ID, client.ContainerStartOptions{}); err != nil {
		if _, rmErr := cli.ContainerRemove(ctx, created.ID, client.ContainerRemoveOptions{}); rmErr != nil {
			log.Warn().Err(rmErr).Str("helper", spec.Name).Msg("self-update: unable to remove helper that failed to start")
		}
		return fmt.Errorf("start helper failed: %w", err)
	}
	return nil
}

func trimName(name string) string {
	return strings.TrimPrefix(name, "/")
}

package docker

import (
	"context"
	"fmt"
	"strings"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/container/swap"
	docker_types "github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/image"
	"github.com/rs/zerolog/log"
)

// RollbackContainer swaps c back to the image it ran before its last update,
// with the same swap an update uses: the current container is kept until the
// one on the previous image has stayed up, and is put back if it does not.
//
// The target is the previous image this container's own update replaced, from
// the host's update events or else from the dev.dozzle.previous-image label.
// An image that is no longer on the host is pulled again by its digest, never
// by a tag: a tag names whatever was pushed last, which is the image being
// rolled back from.
func (d *Service) RollbackContainer(ctx context.Context, c container.Container, opts container.RollbackOptions, progressCh chan<- container.UpdateProgress) error {
	defer close(progressCh)

	// See UpdateContainer: an unguarded send outlives the request.
	reqCtx := ctx
	progress := func(p container.UpdateProgress) {
		select {
		case progressCh <- p:
		case <-reqCtx.Done():
		}
	}
	fail := func(err error) error {
		progress(container.UpdateProgress{Status: container.UpdateError, Error: err.Error()})
		return err
	}

	inspectResp, err := d.client.ContainerInspect(ctx, c.ID)
	if err != nil {
		return fail(fmt.Errorf("inspect failed: %w", err))
	}
	if inspectResp.Config == nil {
		return fail(fmt.Errorf("inspect failed: container has no config"))
	}
	if inspectResp.Config.Labels["com.docker.swarm.service.name"] != "" || c.Labels["com.docker.swarm.service.name"] != "" {
		return fail(fmt.Errorf("%w for swarm services", container.ErrRollbackUnsupported))
	}
	if isSelf(c.ID) || mayBeSelf(inspectResp) {
		return fail(fmt.Errorf("%w for Dozzle's own container, update it instead", container.ErrRollbackUnsupported))
	}

	ref := swap.ImageRef(inspectResp.Config)
	var fromImage *image.InspectResponse
	if img, err := d.client.ImageInspect(ctx, inspectResp.Image); err == nil {
		fromImage = &img
	} else {
		log.Warn().Err(err).Str("image", inspectResp.Image).Msg("rollback: could not inspect the current image")
	}
	fromDigest := swap.PreviousRef(fromImage, ref)

	if opts.ExpectedFromDigest != "" && !runsDigest(inspectResp.Image, fromImage, opts.ExpectedFromDigest) {
		running := fromDigest
		if running == "" {
			running = inspectResp.Image
		}
		return fail(fmt.Errorf("%w: it runs %s, not %s", container.ErrDigestMismatch, running, opts.ExpectedFromDigest))
	}

	target, err := d.rollbackTarget(inspectResp, opts.ToImageID)
	if err != nil {
		return fail(err)
	}

	imageID, err := d.ensureImage(ctx, target, progress)
	if err != nil {
		return fail(err)
	}

	progress(container.UpdateProgress{Status: container.UpdateRecreating})

	containerName := strings.TrimPrefix(inspectResp.Name, "/")
	dependents, err := d.client.NetworkDependents(ctx, inspectResp.ID, containerName)
	if err != nil {
		return fail(fmt.Errorf("list dependents failed: %w", err))
	}

	// As with an update, nothing may stop the swap halfway.
	ctx = context.WithoutCancel(ctx)

	result, err := swap.Swap(ctx, d.client.SwapAPI(), inspectResp, swap.Options{
		// The current image's own settings are dropped, so the previous
		// image's defaults apply again.
		OldImage: fromImage,
		Image:    imageID,
		Labels: map[string]string{
			// What a rollback replaced is the image it rolled back from. The
			// update event reads the source off the new container, and the
			// rollback source is what says it has no target of its own.
			container.PreviousImageLabel: inspectResp.Image,
			container.PreviousRefLabel:   fromDigest,
			container.UpdateSourceLabel:  container.UpdateSourceRollback,
			container.UpdateRunLabel:     "",
		},
		OnVerifying: func() { progress(container.UpdateProgress{Status: container.UpdateVerifying}) },
		LogPrefix:   "rollback",
	})
	if err != nil {
		if result.OldStopped && result.RolledBack {
			if rejoinErr := d.rejoinDependents(ctx, dependents, inspectResp.ID, result.RestoredID, true); rejoinErr != nil {
				err = fmt.Errorf("%w; %v", err, rejoinErr)
			}
		}
		if result.RolledBack {
			d.recordRolledBack(ctx, c, inspectResp, fromImage, ref, imageID, result, container.UpdateOptions{Source: container.UpdateSourceRollback})
			progress(container.UpdateProgress{Status: container.UpdateRolledBack, Error: err.Error()})
			return fmt.Errorf("rollback undone, the container still runs its current image: %w", err)
		}
		return fail(err)
	}

	// As with an update, a container that was not running is left stopped,
	// and so are its dependents.
	if err := d.rejoinDependents(ctx, dependents, inspectResp.ID, result.NewID, swap.Running(inspectResp.State)); err != nil {
		progress(container.UpdateProgress{Status: container.UpdateError, Error: err.Error()})
		return err
	}

	// Once someone rolled back, the image rolled back from is no longer worth
	// keeping for a second thought. Usually its tag still names it, and then
	// it stays.
	swap.RemoveLeftoverImage(ctx, d.client, inspectResp, inspectResp.Image, "the image rolled back from")

	log.Info().Str("container", containerName).Str("from", inspectResp.Image).Str("to", imageID).Msg("rollback: done")
	progress(container.UpdateProgress{Status: container.UpdateDone})
	return nil
}

// rollbackTarget is the image the container ran before the update that
// created it, as container.RollbackTargetOf works it out from this host's
// update events and the container's labels. The UI asks the same question
// through the container service, so it offers what this would pick.
//
// want, when set, must be that target; it is what the UI showed the user.
// Only the one target is accepted: an older image the labels still name is
// never one, since rolling back to it would skip a version.
func (d *Service) rollbackTarget(inspect docker_types.InspectResponse, want string) (container.RollbackTarget, error) {
	target, err := container.RollbackTargetOf(inspect.ID, inspect.Image, inspect.Config.Labels, d.RecentUpdates())
	if err != nil {
		return container.RollbackTarget{}, err
	}
	if want != "" && !container.SameImageID(target.ImageID, want) {
		return container.RollbackTarget{}, fmt.Errorf("%w: %s is not the image this container ran before", container.ErrNoRollbackTarget, want)
	}
	return target, nil
}

// ensureImage makes sure target is in the local store and returns its id. An
// image that was pruned is pulled again by digest. A tag is never pulled: it
// names the newest image, the very one being rolled back from.
func (d *Service) ensureImage(ctx context.Context, target container.RollbackTarget, progress func(container.UpdateProgress)) (string, error) {
	if img, err := d.client.ImageInspect(ctx, target.ImageID); err == nil {
		return img.ID, nil
	}
	if !strings.Contains(target.Ref, "@sha256:") {
		return "", fmt.Errorf("the previous image %s is no longer on this host, and it has no registry digest to pull it by", shortImageID(target.ImageID))
	}
	log.Info().Str("ref", target.Ref).Msg("rollback: previous image is gone, pulling it by digest")
	if err := d.pull(ctx, target.Ref, progress); err != nil {
		return "", fmt.Errorf("the previous image is no longer on this host and pulling %s failed: %w", target.Ref, err)
	}
	id, err := d.client.ImageID(ctx, target.Ref)
	if err != nil {
		return "", fmt.Errorf("resolve pulled image %s: %w", target.Ref, err)
	}
	return id, nil
}

// runsDigest reports whether a container on imageID (inspected as img, nil if
// that failed) runs expected: one of its repo digests, or its image id.
func runsDigest(imageID string, img *image.InspectResponse, expected string) bool {
	want := container.DigestOf(strings.TrimSpace(expected))
	if want == "" {
		return false
	}
	if container.SameImageID(imageID, want) {
		return true
	}
	if img == nil {
		return false
	}
	for _, d := range img.RepoDigests {
		if container.DigestOf(d) == want {
			return true
		}
	}
	return false
}

func shortImageID(id string) string {
	id = strings.TrimPrefix(id, "sha256:")
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

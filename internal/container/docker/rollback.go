package docker

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/container/swap"
	docker_types "github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/image"
	"github.com/rs/zerolog/log"
)

// rollbackTarget is an image a container ran before its last update.
type rollbackTarget struct {
	imageID string
	// ref is the image as repo@sha256:digest, to pull it again by digest once
	// it is gone locally. Empty for an image built locally.
	ref string
}

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

	target, err := d.rollbackTarget(c, inspectResp, opts.ToImageID)
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
// created it. The host's update events know that for any update Dozzle saw,
// whoever made it. The previous-image label covers an update made before
// Dozzle started, but it only says what Dozzle's own last update replaced:
// Watchtower copies every label, so after a later Watchtower update the label
// may name an older image, which is why the events win.
//
// want, when set, must be that target; it is what the UI showed the user.
func (d *Service) rollbackTarget(c container.Container, inspect docker_types.InspectResponse, want string) (rollbackTarget, error) {
	var candidates []rollbackTarget
	labels := inspect.Config.Labels
	madeByRollback := labels[container.UpdateSourceLabel] == container.UpdateSourceRollback

	if d.store != nil && !madeByRollback {
		events := d.store.RecentUpdates()
		for _, e := range slices.Backward(events) {
			if e.Name != c.Name || e.RolledBack || e.NewID != shortContainerID(inspect.ID) {
				continue
			}
			if e.Source == container.UpdateSourceRollback {
				madeByRollback = true
			} else if e.FromImageID != "" && e.FromImageID != inspect.Image {
				candidates = append(candidates, rollbackTarget{imageID: e.FromImageID, ref: e.FromDigest})
			}
			break
		}
	}
	if madeByRollback {
		return rollbackTarget{}, fmt.Errorf("%w: the container was already rolled back", container.ErrNoRollbackTarget)
	}
	if previous := labels[container.PreviousImageLabel]; previous != "" && previous != inspect.Image {
		candidates = append(candidates, rollbackTarget{imageID: previous, ref: labels[container.PreviousRefLabel]})
	}

	if len(candidates) == 0 {
		return rollbackTarget{}, container.ErrNoRollbackTarget
	}
	if want == "" {
		return candidates[0], nil
	}
	for _, candidate := range candidates {
		if sameImageID(candidate.imageID, want) {
			return candidate, nil
		}
	}
	return rollbackTarget{}, fmt.Errorf("%w: %s is not the image this container ran before", container.ErrNoRollbackTarget, want)
}

// ensureImage makes sure target is in the local store and returns its id. An
// image that was pruned is pulled again by digest. A tag is never pulled: it
// names the newest image, the very one being rolled back from.
func (d *Service) ensureImage(ctx context.Context, target rollbackTarget, progress func(container.UpdateProgress)) (string, error) {
	if img, err := d.client.ImageInspect(ctx, target.imageID); err == nil {
		return img.ID, nil
	}
	if !strings.Contains(target.ref, "@sha256:") {
		return "", fmt.Errorf("the previous image %s is no longer on this host, and it has no registry digest to pull it by", shortImageID(target.imageID))
	}
	log.Info().Str("ref", target.ref).Msg("rollback: previous image is gone, pulling it by digest")
	if err := d.pull(ctx, target.ref, progress); err != nil {
		return "", fmt.Errorf("the previous image is no longer on this host and pulling %s failed: %w", target.ref, err)
	}
	id, err := d.client.ImageID(ctx, target.ref)
	if err != nil {
		return "", fmt.Errorf("resolve pulled image %s: %w", target.ref, err)
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
	if sameImageID(imageID, want) {
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

// sameImageID compares image ids with or without their sha256: prefix.
func sameImageID(a, b string) bool {
	a, b = strings.TrimPrefix(a, "sha256:"), strings.TrimPrefix(b, "sha256:")
	return a != "" && a == b
}

func shortImageID(id string) string {
	id = strings.TrimPrefix(id, "sha256:")
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

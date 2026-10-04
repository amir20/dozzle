package docker

import (
	"context"
	"fmt"
	"strings"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/container/swap"
	"github.com/moby/moby/api/types/image"
	"github.com/rs/zerolog/log"
)

// RollbackContainer swaps c back to the image it ran before its last update,
// with the same swap an update uses: the current container is kept until the
// one on the previous image has stayed up, and is put back if it does not.
//
// The target is the dev.dozzle.previous-image label the last update left. It
// must still be on the host: nothing is pulled, since the tag names the newest
// image, the very one being rolled back from. The new container is stamped
// with dev.dozzle.rolled-back-from, so the auto-update schedule does not bring
// that image straight back.
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

	labels := inspectResp.Config.Labels
	if labels[container.RolledBackFromLabel] != "" {
		return fail(fmt.Errorf("%w: the container was already rolled back", container.ErrNoRollbackTarget))
	}
	previous := strings.TrimSpace(labels[container.PreviousImageLabel])
	if previous == "" || container.SameImageID(previous, inspectResp.Image) {
		return fail(container.ErrNoRollbackTarget)
	}
	target, err := d.client.ImageInspect(ctx, previous)
	if err != nil {
		return fail(fmt.Errorf("%w: the previous image %s is no longer on this host", container.ErrNoRollbackTarget, shortImageID(previous)))
	}

	progress(container.UpdateProgress{Status: container.UpdateRecreating})

	containerName := strings.TrimPrefix(inspectResp.Name, "/")
	dependents, err := d.client.NetworkDependents(ctx, inspectResp.ID, containerName)
	if err != nil {
		return fail(fmt.Errorf("list dependents failed: %w", err))
	}

	// As with an update, nothing may stop the swap halfway.
	ctx = context.WithoutCancel(ctx)

	rolledBackFrom := fromDigest
	if rolledBackFrom == "" {
		rolledBackFrom = inspectResp.Image
	}
	result, err := swap.Swap(ctx, d.client.SwapAPI(), inspectResp, swap.Options{
		// The current image's own settings are dropped, so the previous
		// image's defaults apply again.
		OldImage: fromImage,
		Image:    target.ID,
		Labels: map[string]string{
			// A rolled back container has no rollback target of its own: its
			// previous image is the newer one it rolled back from.
			container.PreviousImageLabel:  "",
			container.PreviousRefLabel:    "",
			container.RolledBackFromLabel: rolledBackFrom,
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
			// The rollback itself was undone: the container still runs the
			// image it ran before, so there is no change to record.
			progress(container.UpdateProgress{Status: container.UpdateRolledBack, Error: err.Error()})
			return fmt.Errorf("rollback undone, the container still runs its current image: %w", err)
		}
		return fail(err)
	}

	done := container.UpdateResult{
		OldID:       shortContainerID(inspectResp.ID),
		NewID:       shortContainerID(result.NewID),
		FromImageID: inspectResp.Image,
		ToImageID:   target.ID,
		FromDigest:  fromDigest,
		ToDigest:    swap.PreviousRef(&target, ref),
	}

	// As with an update, a container that was not running is left stopped,
	// and so are its dependents.
	if err := d.rejoinDependents(ctx, dependents, inspectResp.ID, result.NewID, swap.Running(inspectResp.State)); err != nil {
		progress(container.UpdateProgress{Status: container.UpdateError, Error: err.Error(), Result: &done})
		return err
	}

	// Once someone rolled back, the image rolled back from is no longer worth
	// keeping. Usually its tag still names it, and then it stays.
	swap.RemoveLeftoverImage(ctx, d.client, inspectResp, inspectResp.Image, "the image rolled back from")

	log.Info().Str("container", containerName).Str("from", inspectResp.Image).Str("to", target.ID).Msg("rollback: done")
	progress(container.UpdateProgress{Status: container.UpdateDone, Result: &done})
	return nil
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

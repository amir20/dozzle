package selfupdate

import (
	"context"
	"fmt"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/container/docker/swap"
	"github.com/moby/moby/api/types/image"
	"github.com/moby/moby/client"
	"github.com/rs/zerolog/log"
)

// Run recreates targetID from the image its tag now points to, keeping its
// config, networks and volumes, and puts the old container back if the new one
// does not come up. The swap itself, and why its order keeps --rm containers
// safe, is in internal/container/docker/swap.
func Run(ctx context.Context, targetID string) error {
	cli, err := newClient(ctx)
	if err != nil {
		return err
	}
	defer cli.Close()
	return run(ctx, cli, targetID, "")
}

// Rejoin recreates targetID the same way Run does, but joined to networkMode
// (container:<id>) and even when its image has not changed. A container that
// shares another's network namespace by id is cut off for good once that one
// is recreated, and restarting it fails because the id no longer exists.
func Rejoin(ctx context.Context, targetID string, networkMode string) error {
	cli, err := newClient(ctx)
	if err != nil {
		return err
	}
	defer cli.Close()
	return run(ctx, cli, targetID, networkMode)
}

func run(ctx context.Context, cli dockerAPI, targetID string, networkMode string) error {
	result, err := cli.ContainerInspect(ctx, targetID, client.ContainerInspectOptions{})
	if err != nil {
		return fmt.Errorf("inspect %s: %w", targetID, err)
	}
	old := result.Container
	if old.Config == nil || old.HostConfig == nil {
		return fmt.Errorf("inspect %s: incomplete container config", targetID)
	}
	if old.Config.Labels[container.SwarmServiceNameLabel] != "" {
		return ErrSwarm
	}

	logger := log.With().Str("container", trimName(old.Name)).Str("id", shortID(old.ID)).Logger()

	ref := swap.ImageRef(old.Config)
	newImage, err := cli.ImageInspect(ctx, ref)
	if err != nil {
		return fmt.Errorf("resolve image %s: %w", ref, err)
	}
	if newImage.ID == old.Image && networkMode == "" {
		logger.Info().Str("image", ref).Msg("self-update: already on the latest image, nothing to do")
		return nil
	}
	var oldImage *image.InspectResponse
	if result, err := cli.ImageInspect(ctx, old.Image); err == nil {
		oldImage = &result.InspectResponse
	} else {
		logger.Warn().Err(err).Msg("self-update: old image not inspectable, keeping its defaults in the new config")
	}

	logger.Info().
		Str("image", ref).
		Str("from", shortID(old.Image)).
		Str("to", shortID(newImage.ID)).
		Bool("autoRemove", old.HostConfig.AutoRemove).
		Str("networkMode", networkMode).
		Msg("self-update: starting")

	// A new image is stamped and cleaned up exactly as any other container
	// update does it (see docker.Service.UpdateContainer). A rejoin onto the
	// image Dozzle already runs is not an update: the replacement keeps the
	// old container's labels and no image is removed.
	imageChanged := newImage.ID != old.Image
	var labels map[string]string
	if imageChanged {
		labels = swap.PreviousLabels(old, oldImage, ref)
	}

	swapped, err := swap.Swap(ctx, cli, old, swap.Options{
		OldImage:    oldImage,
		NetworkMode: networkMode,
		Labels:      labels,
		LogPrefix:   "self-update",
	})
	if err != nil {
		return err
	}
	logger.Info().Str("newId", shortID(swapped.NewID)).Msg("self-update: done")

	// Only after the swap committed: a rollback returned above. Swarm tasks
	// never get here (ErrSwarm), since the swarm manager replaces them and
	// each node keeps its own images.
	if imageChanged {
		swap.CleanupImage(ctx, cli, old)
	}
	return nil
}

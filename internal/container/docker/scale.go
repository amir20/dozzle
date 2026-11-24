package docker

import (
	"context"
	"fmt"

	"github.com/amir20/dozzle/internal/container"
	"github.com/moby/moby/client"
	"github.com/rs/zerolog/log"
)

// ScaleService sets a replicated swarm service's replica count, the same as
// `docker service scale name=n`. Nothing else in the spec changes, and the
// inspected version is sent back so a concurrent update makes this one fail
// rather than silently undoing it.
func (d *Client) ScaleService(ctx context.Context, name string, replicas uint64) error {
	if !d.info.Swarm.ControlAvailable {
		return container.ErrNotSwarmManager
	}

	result, err := d.cli.ServiceInspect(ctx, name, client.ServiceInspectOptions{})
	if err != nil {
		return fmt.Errorf("could not inspect service %s: %w", name, err)
	}
	svc := result.Service
	if svc.Spec.Mode.Replicated == nil {
		return container.ErrServiceNotReplicated
	}

	svc.Spec.Mode.Replicated.Replicas = &replicas
	// The manager has to answer before this returns, so a client that
	// disconnects must not cancel an update the daemon may already have begun.
	options := client.ServiceUpdateOptions{Version: svc.Version, Spec: svc.Spec}
	if _, err := d.cli.ServiceUpdate(context.WithoutCancel(ctx), svc.ID, options); err != nil {
		return fmt.Errorf("could not scale service %s: %w", name, err)
	}

	log.Info().Str("service", name).Uint64("replicas", replicas).Msg("scaled swarm service")
	return nil
}

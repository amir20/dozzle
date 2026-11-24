package docker

import (
	"context"
	"fmt"

	"github.com/amir20/dozzle/internal/container"
	cerrdefs "github.com/containerd/errdefs"
	"github.com/moby/moby/client"
	"github.com/rs/zerolog/log"
)

// ScaleService sets a replicated swarm service's replica count, the same as
// `docker service scale`. Nothing else in the spec changes, and the inspected
// version is sent back so a concurrent update makes this one fail rather than
// silently undoing it.
//
// Whether this engine can do it is the engine's answer, not the Info read when
// the client was built: a node promoted or demoted since then would be wrong.
// A worker or a node outside any swarm answers 503, and a manager of another
// swarm 404, and both mean the caller should ask the next host.
func (d *Client) ScaleService(ctx context.Context, id string, replicas uint64) error {
	result, err := d.cli.ServiceInspect(ctx, id, client.ServiceInspectOptions{})
	if cerrdefs.IsUnavailable(err) || cerrdefs.IsNotFound(err) {
		return fmt.Errorf("%w: %w", container.ErrNotSwarmManager, err)
	}
	if err != nil {
		return fmt.Errorf("could not inspect service %s: %w", id, err)
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
		return fmt.Errorf("could not scale service %s: %w", svc.Spec.Name, err)
	}

	log.Info().Str("service", svc.Spec.Name).Uint64("replicas", replicas).Msg("scaled swarm service")
	return nil
}

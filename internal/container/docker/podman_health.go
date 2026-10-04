package docker

import (
	"context"
	"time"

	"github.com/moby/moby/client"
)

const podmanHealthAction = "health_status"

// healthAction turns Podman's health event into Docker's encoding.
//
// Docker bakes the result into the action itself:
//
//	{"Action": "health_status: healthy", ...}
//
// Podman's Docker-compatible endpoint sends the bare action and carries the
// result in a field of its own, beside the embedded Docker message
// (pkg/domain/entities/types/events.go):
//
//	{"Action": "health_status", ..., "HealthStatus": "healthy"}
//
// events.Message has no field for it, so the client drops it while decoding, and
// response hooks are not allowed to touch the body to rewrite it first. Asking the
// container for its health keeps that one difference here, so the store, the SSE
// stream, the notification rules and the agent all go on matching a single
// encoding. Docker never sends the bare action, so it never pays for the inspect.
func (d *Client) healthAction(ctx context.Context, action string, id string) string {
	if action != podmanHealthAction {
		return action
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result, err := d.cli.ContainerInspect(ctx, id, client.ContainerInspectOptions{})
	if err != nil || result.Container.State == nil || result.Container.State.Health == nil || result.Container.State.Health.Status == "" {
		return action
	}
	return podmanHealthAction + ": " + string(result.Container.State.Health.Status)
}

package selfupdate

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/amir20/dozzle/internal/container"
	dcontainer "github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/client"
	"github.com/rs/zerolog/log"
)

// A swarm task is never swapped by the helper: the manager would reschedule the
// task it thinks is missing and fight the replacement. Swarm already knows how
// to roll a service onto a new image, with its own restart and rollback
// policies, so Dozzle just asks it to.

const (
	swarmServiceIDLabel = "com.docker.swarm.service.id"
	swarmTaskNameLabel  = "com.docker.swarm.task.name"

	// ReasonSwarmWorker means Dozzle is a swarm task on a worker node. Only a
	// manager can update a service, and a worker's engine refuses the call.
	ReasonSwarmWorker = "swarm-worker"
)

// recentServiceUpdate is how long after a service was last updated another
// update to the same image is treated as a duplicate. Every task of a global
// service, or several replicas, can reach the scheduled minute together.
const recentServiceUpdate = 10 * time.Minute

// taskImageRef is the reference a swarm task should be updated to. Swarm
// usually resolves the tag to a digest when it deploys, so the task's image
// reads amir20/dozzle:latest@sha256:... and would look pinned. The service
// follows the tag, so the digest is dropped and the manager resolves it again.
func taskImageRef(ref string) string {
	if at := strings.Index(ref, "@"); at > 0 {
		return ref[:at]
	}
	return ref
}

// SelfRef is the reference to pull and compare for a container, whether it is
// a plain container or a swarm task.
func SelfRef(cfg *dcontainer.Config) string {
	ref := ImageRef(cfg)
	if cfg != nil && SwarmTask(cfg.Labels) {
		return taskImageRef(ref)
	}
	return ref
}

// SwarmPrimary reports whether this task should be the one that runs the
// schedule. A replicated service's tasks are named service.slot.id, and only
// slot 1 goes ahead. A global service's tasks are named service.node.id, with
// no slot to choose by, so each goes ahead and the recent-update guard in the
// update itself keeps it to one rollout.
func SwarmPrimary(labels map[string]string) bool {
	parts := strings.Split(labels[swarmTaskNameLabel], ".")
	if len(parts) < 3 {
		return true
	}
	slot := parts[len(parts)-2]
	for _, r := range slot {
		if r < '0' || r > '9' {
			return true
		}
	}
	return slot == "1"
}

// updateService rolls the service onto ref. ForceUpdate makes swarm replace
// the tasks even when the spec's image string did not change, which is the
// normal case for a moving tag.
func updateService(ctx context.Context, cli dockerAPI, serviceID, ref string, progress func(container.UpdateProgress)) (bool, error) {
	fail := func(format string, args ...any) (bool, error) {
		err := fmt.Errorf(format, args...)
		progress(container.UpdateProgress{Status: "error", Error: err.Error()})
		return false, err
	}

	if serviceID == "" {
		return fail("service update failed: task has no service id")
	}
	result, err := cli.ServiceInspect(ctx, serviceID, client.ServiceInspectOptions{})
	if err != nil {
		return fail("service update failed: %w", err)
	}
	svc := result.Service
	if svc.UpdateStatus != nil && svc.UpdateStatus.State == swarm.UpdateStateUpdating {
		return fail("an update of this service is already in progress")
	}
	if svc.Spec.TaskTemplate.ContainerSpec == nil {
		return fail("service update failed: service has no container spec")
	}
	if taskImageRef(svc.Spec.TaskTemplate.ContainerSpec.Image) == ref && time.Since(svc.UpdatedAt) < recentServiceUpdate {
		log.Info().Str("service", serviceID).Time("updatedAt", svc.UpdatedAt).Msg("self-update: service was updated moments ago, not starting another rollout")
		progress(container.UpdateProgress{Status: "done"})
		return true, nil
	}

	progress(container.UpdateProgress{Status: "recreating"})
	svc.Spec.TaskTemplate.ContainerSpec.Image = ref
	svc.Spec.TaskTemplate.ForceUpdate++
	// The manager has to answer before this returns, so a client that
	// disconnects must not cancel a rollout the daemon may already have begun.
	if _, err := cli.ServiceUpdate(context.WithoutCancel(ctx), serviceID, client.ServiceUpdateOptions{Version: svc.Version, Spec: svc.Spec}); err != nil {
		return fail("service update failed: %w", err)
	}
	log.Info().Str("service", serviceID).Str("image", ref).Msg("self-update: asked the swarm manager to roll the service onto the new image")
	progress(container.UpdateProgress{Status: "done"})
	return true, nil
}

// SwarmServiceID is the id of the service a swarm task belongs to, or "".
func SwarmServiceID(labels map[string]string) string {
	return labels[swarmServiceIDLabel]
}

// SwarmManager reports whether the local engine can update the service, i.e.
// whether it is a manager.
func SwarmManager(ctx context.Context, serviceID string) bool {
	cli, err := newClient(ctx)
	if err != nil {
		return false
	}
	defer cli.Close()
	return swarmManager(ctx, cli, serviceID)
}

// swarmManager reports whether this engine can update services, i.e. whether
// it is a manager. A worker answers a service inspect with an error.
func swarmManager(ctx context.Context, cli dockerAPI, serviceID string) bool {
	if serviceID == "" {
		return false
	}
	_, err := cli.ServiceInspect(ctx, serviceID, client.ServiceInspectOptions{})
	return err == nil
}

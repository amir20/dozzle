package container

import (
	"context"
	"time"

	"github.com/amir20/dozzle/internal/imagecheck"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

// Labels the swarm manager puts on every task container.
const (
	SwarmServiceIDLabel   = "com.docker.swarm.service.id"
	SwarmServiceNameLabel = "com.docker.swarm.service.name"
)

// Updatable reports whether c is worth checking for an update. A stopped
// standalone container still is, so it shows that an update is waiting, but it
// is only updated once someone starts it (see ErrNotRunning). An exited swarm task is not: it is history the orchestrator left behind after
// replacing it, and its service is updated through the task that is running.
func Updatable(c Container) bool {
	if c.State == "deleted" {
		return false
	}
	return c.Labels[SwarmServiceIDLabel] == "" || c.State == "running"
}

// ContainerFinder resolves a listed container to the service that can check it.
type ContainerFinder interface {
	FindContainer(host string, id string, labels ContainerLabels) (*ContainerService, error)
}

// ImageUpdate is one container's update check, with the service that can act
// on it.
type ImageUpdate struct {
	Container Container
	Service   *ContainerService
	Result    imagecheck.Result
}

// checkTimeout bounds a single container's check. Each one inspects the
// container on its own host first, which is the slow half for a remote agent.
const checkTimeout = 30 * time.Second

// CheckImageUpdates checks every updatable container in containers, in their
// order. containers must already be filtered to what the caller may see: the
// finder is asked without labels, since passing them again would re-list the
// host for every container. The checker looks each image up once however many
// containers run it.
func CheckImageUpdates(ctx context.Context, finder ContainerFinder, containers []Container, force bool) []ImageUpdate {
	results := make([]ImageUpdate, len(containers))
	checked := make([]bool, len(containers))

	// Each goroutine writes only its own index, so the slices need no lock.
	var group errgroup.Group
	group.SetLimit(8)
	for i, c := range containers {
		if !Updatable(c) {
			continue
		}
		group.Go(func() error {
			update := ImageUpdate{Container: c}
			service, err := finder.FindContainer(c.Host, c.ID, nil)
			if err != nil {
				log.Debug().Err(err).Str("container", c.Name).Msg("image update check: container not found")
				return nil
			}
			update.Service = service

			checkCtx, cancel := context.WithTimeout(ctx, checkTimeout)
			result, err := service.CheckImageUpdate(checkCtx, force)
			cancel()
			if err != nil {
				// Reported rather than dropped, so an earlier "update available"
				// does not outlive the check that could no longer confirm it.
				log.Debug().Err(err).Str("container", c.Name).Msg("image update check failed")
				result = imagecheck.Result{Image: c.Image, Status: imagecheck.StatusUnknown, Reason: err.Error(), CheckedAt: time.Now()}
			}
			update.Result = result

			results[i] = update
			checked[i] = true
			return nil
		})
	}
	_ = group.Wait()

	out := results[:0]
	for i, update := range results {
		if checked[i] {
			out = append(out, update)
		}
	}
	return out
}

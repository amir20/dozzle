package docker

import (
	"context"
	"maps"
	"sync"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/moby/moby/client"
	"github.com/rs/zerolog/log"
)

// Swarm writes a compose file's `deploy.labels` onto the *service*, never onto the task
// containers, so inspecting a task never sees them. That is where the ecosystem puts its
// labels: traefik's swarm provider reads service labels and its docs tell you to use
// `deploy.labels`, so a swarm user writes `dev.dozzle.url` there too. Without merging
// them back onto the container every `dev.dozzle.*` label is silently ignored on swarm.
//
// Listing services is manager-only. Agents on worker nodes skip this and their containers
// keep only their own labels.

// Service labels change on `docker service update` and nothing else, so a short window
// turns one API call per container into one per refresh.
const serviceLabelTTL = 30 * time.Second

type serviceLabelCache struct {
	mu      sync.Mutex
	labels  map[string]map[string]string
	fetched time.Time
}

// all returns the labels of every swarm service keyed by service id, refreshing when
// stale. A failed refresh keeps whatever is already cached rather than dropping labels
// out of the UI on a transient error.
func (s *serviceLabelCache) all(ctx context.Context, cli DockerCLI) map[string]map[string]string {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Keyed off the attempt, not the result, so a node that cannot list services backs off
	// for a window instead of retrying on every container.
	if !s.fetched.IsZero() && time.Since(s.fetched) < serviceLabelTTL {
		return s.labels
	}

	result, err := cli.ServiceList(ctx, client.ServiceListOptions{})
	if err != nil {
		log.Debug().Err(err).Msg("could not list swarm services for labels")
		s.fetched = time.Now()
		return s.labels
	}

	labels := make(map[string]map[string]string, len(result.Items))
	for _, service := range result.Items {
		if len(service.Spec.Labels) > 0 {
			labels[service.ID] = service.Spec.Labels
		}
	}

	s.labels = labels
	s.fetched = time.Now()
	return s.labels
}

// mergeServiceLabels folds each swarm service's labels into its task containers. A label
// set on the container itself is the more specific of the two and always wins.
func (d *DockerClient) mergeServiceLabels(ctx context.Context, containers ...*container.Container) {
	if !d.info.Swarm.ControlAvailable {
		return
	}

	var byService map[string]map[string]string
	fetched := false

	for _, c := range containers {
		serviceID := c.Labels["com.docker.swarm.service.id"]
		if serviceID == "" {
			continue
		}
		if !fetched {
			byService = d.serviceLabels.all(ctx, d.cli)
			fetched = true
		}

		serviceLabels := byService[serviceID]
		if len(serviceLabels) == 0 {
			continue
		}

		// Copy on write. The container's label map came straight off the API response and
		// is shared with whatever else holds that container.
		merged := make(map[string]string, len(serviceLabels)+len(c.Labels))
		maps.Copy(merged, serviceLabels)
		maps.Copy(merged, c.Labels)
		c.Labels = merged

		// Name and group are derived from labels at construction, before this ran. Redo
		// that derivation on the merged set, using the same precedence.
		if name := merged["dev.dozzle.name"]; name != "" {
			c.Name = name
		} else if name := coolifyName(merged); name != "" {
			c.Name = name
		}
		if group := merged["dev.dozzle.group"]; group != "" {
			c.Group = group
		} else if group := merged["coolify.projectName"]; group != "" {
			c.Group = group
		}
	}
}

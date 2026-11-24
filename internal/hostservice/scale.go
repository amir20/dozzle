package hostservice

import (
	"context"
	"errors"
	"slices"

	"github.com/amir20/dozzle/internal/container"
)

// ErrServiceNotFound means none of the service's tasks are visible to the caller.
var ErrServiceNotFound = errors.New("service not found")

// ScaleService sets a swarm service's replica count. The caller must see at
// least one of its tasks, so a label filter that hides a service also keeps it
// from being scaled.
//
// The service is addressed by the ids on those visible tasks, never by name.
// Hosts from two swarms can each run a service of the same name, and an id
// only resolves in the swarm it belongs to, so a scale cannot land on the one
// the caller's filter hides. The view merges same-named services, so each
// visible one is scaled.
//
// Only a manager's engine can update a service, and the node serving this
// request is often a worker: Dozzle usually runs as a global service behind the
// routing mesh. So each host is asked in turn, this one first, and a host that
// cannot do it (a worker, another swarm's manager) moves on to the next.
func (m *MultiHostService) ScaleService(ctx context.Context, name string, replicas uint64, labels container.ContainerLabels) error {
	containers, _ := m.ListAllContainers(labels)
	var ids []string
	for _, c := range containers {
		id := c.Labels[container.SwarmServiceIDLabel]
		if c.Labels[container.SwarmServiceNameLabel] == name && id != "" && !slices.Contains(ids, id) {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return ErrServiceNotFound
	}

	clients := append(m.manager.LocalClientServices(), m.manager.List()...)
	for _, id := range ids {
		if err := scaleOnAManager(ctx, clients, id, replicas); err != nil {
			return err
		}
	}
	return nil
}

func scaleOnAManager(ctx context.Context, clients []container.ClientService, id string, replicas uint64) error {
	tried := make(map[container.ClientService]bool)
	for _, client := range clients {
		if tried[client] {
			continue
		}
		tried[client] = true

		scaler, ok := client.(container.ServiceScaler)
		if !ok {
			continue
		}
		if err := scaler.ScaleService(ctx, id, replicas); !errors.Is(err, container.ErrNotSwarmManager) {
			return err
		}
	}
	return container.ErrNotSwarmManager
}

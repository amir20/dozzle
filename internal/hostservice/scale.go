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
// Only a manager's engine can update a service, and the node serving this
// request is often a worker: Dozzle usually runs as a global service behind the
// routing mesh. So each host is asked in turn, this one first, and a worker's
// refusal moves on to the next. The refusal is the manager check, which keeps
// it right after a node is promoted or demoted.
func (m *MultiHostService) ScaleService(ctx context.Context, name string, replicas uint64, labels container.ContainerLabels) error {
	containers, _ := m.ListAllContainers(labels)
	if !slices.ContainsFunc(containers, func(c container.Container) bool {
		return c.Labels[container.SwarmServiceNameLabel] == name
	}) {
		return ErrServiceNotFound
	}

	tried := make(map[container.ClientService]bool)
	for _, client := range append(m.manager.LocalClientServices(), m.manager.List()...) {
		if tried[client] {
			continue
		}
		tried[client] = true

		scaler, ok := client.(container.ServiceScaler)
		if !ok {
			continue
		}
		if err := scaler.ScaleService(ctx, name, replicas); !errors.Is(err, container.ErrNotSwarmManager) {
			return err
		}
	}
	return container.ErrNotSwarmManager
}

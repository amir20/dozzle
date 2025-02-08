package docker_support

import (
	"context"
	"fmt"
	"time"

	"github.com/amir20/dozzle/internal/container"
	container_support "github.com/amir20/dozzle/internal/support/container"
	"github.com/rs/zerolog/log"
)

type HostUnavailableError struct {
	Host container.Host
	Err  error
}

func (h *HostUnavailableError) Error() string {
	return fmt.Sprintf("host %s unavailable: %v", h.Host.ID, h.Err)
}

type ClientManager interface {
	Find(id string) (container_support.ClientService, bool)
	List() []container_support.ClientService
	RetryAndList() ([]container_support.ClientService, []error)
	Subscribe(ctx context.Context, channel chan<- container.Host)
	Hosts(ctx context.Context) []container.Host
	LocalClients() []container.Client
}

type MultiHostService struct {
	manager ClientManager
	timeout time.Duration
}

func NewMultiHostService(manager ClientManager, timeout time.Duration) *MultiHostService {
	m := &MultiHostService{
		manager: manager,
		timeout: timeout,
	}

	return m
}

func (m *MultiHostService) FindContainer(host string, id string, labels container.ContainerLabels) (*container_support.ContainerService, error) {
	client, ok := m.manager.Find(host)
	if !ok {
		return nil, fmt.Errorf("host %s not found", host)
	}
	ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
	defer cancel()
	container, err := client.FindContainer(ctx, id, labels)
	if err != nil {
		return nil, err
	}

	return container_support.NewContainerService(client, container), nil
}

func (m *MultiHostService) ListContainersForHost(host string, labels container.ContainerLabels) ([]container.Container, error) {
	client, ok := m.manager.Find(host)
	if !ok {
		return nil, fmt.Errorf("host %s not found", host)
	}
	ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
	defer cancel()

	return client.ListContainers(ctx, labels)
}

func (m *MultiHostService) ListAllContainers(labels container.ContainerLabels) ([]container.Container, []error) {
	containers := make([]container.Container, 0)
	clients, errors := m.manager.RetryAndList()

	for _, client := range clients {
		ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
		defer cancel()
		list, err := client.ListContainers(ctx, labels)
		if err != nil {
			host, _ := client.Host(ctx)
			log.Debug().Err(err).Str("host", host.Name).Msg("error listing containers")
			host.Available = false
			errors = append(errors, &HostUnavailableError{Host: host, Err: err})
			continue
		}

		containers = append(containers, list...)
	}

	return containers, errors
}

func (m *MultiHostService) ListAllContainersFiltered(userLabels container.ContainerLabels, filter container_support.ContainerFilter) ([]container.Container, []error) {
	containers, err := m.ListAllContainers(userLabels)
	filtered := make([]container.Container, 0, len(containers))
	for _, container := range containers {
		if filter(&container) {
			filtered = append(filtered, container)
		}
	}
	return filtered, err
}

func (m *MultiHostService) SubscribeEventsAndStats(ctx context.Context, events chan<- container.ContainerEvent, stats chan<- container.ContainerStat) {
	for _, client := range m.manager.List() {
		client.SubscribeEvents(ctx, events)
		client.SubscribeStats(ctx, stats)
	}
}

func (m *MultiHostService) SubscribeContainersStarted(ctx context.Context, containers chan<- container.Container, filter container_support.ContainerFilter) {
	newContainers := make(chan container.Container)
	for _, client := range m.manager.List() {
		client.SubscribeContainersStarted(ctx, newContainers)
	}
	go func() {
		<-ctx.Done()
		close(newContainers)
	}()

	go func() {
		for container := range newContainers {
			if filter(&container) {
				select {
				case containers <- container:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
}

func (m *MultiHostService) Hosts() []container.Host {
	ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
	defer cancel()
	return m.manager.Hosts(ctx)
}

func (m *MultiHostService) LocalHost() (container.Host, error) {
	for _, host := range m.Hosts() {
		if host.Type == "local" {
			return host, nil
		}
	}
	return container.Host{}, fmt.Errorf("local host not found")
}

func (m *MultiHostService) SubscribeAvailableHosts(ctx context.Context, hosts chan<- container.Host) {
	m.manager.Subscribe(ctx, hosts)
}

func (m *MultiHostService) LocalClients() []container.Client {
	return m.manager.LocalClients()
}

func (m *MultiHostService) TotalClients() int {
	return len(m.manager.List())
}

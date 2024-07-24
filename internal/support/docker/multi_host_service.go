package docker_support

import (
	"context"
	"fmt"

	"github.com/amir20/dozzle/internal/docker"
	log "github.com/sirupsen/logrus"
)

type ContainerFilter = func(*docker.Container) bool

type HostUnavailableError struct {
	Host docker.Host
	Err  error
}

func (h *HostUnavailableError) Error() string {
	return fmt.Sprintf("host %s unavailable: %v", h.Host.ID, h.Err)
}

type ClientManager interface {
	Find(id string) (ClientService, bool)
	List() []ClientService
	RetryAndList() ([]ClientService, []error)
	FailedAgents() []string
	Subscribe(ctx context.Context, channel chan<- docker.Host)
}

type MultiHostService struct {
	manager   ClientManager
	SwarmMode bool
}

func NewMultiHostService(manager ClientManager) *MultiHostService {
	m := &MultiHostService{
		manager: manager,
	}

	log.Debugf("created multi host service manager %s", manager)

	return m
}

func (m *MultiHostService) FindContainer(host string, id string) (*containerService, error) {
	client, ok := m.manager.Find(host)
	if !ok {
		return nil, fmt.Errorf("host %s not found", host)
	}

	container, err := client.FindContainer(id)
	if err != nil {
		return nil, err
	}

	return &containerService{
		clientService: client,
		Container:     container,
	}, nil
}

func (m *MultiHostService) ListContainersForHost(host string) ([]docker.Container, error) {
	client, ok := m.manager.Find(host)
	if !ok {
		return nil, fmt.Errorf("host %s not found", host)
	}

	return client.ListContainers()
}

func (m *MultiHostService) ListAllContainers() ([]docker.Container, []error) {
	containers := make([]docker.Container, 0)
	clients, errors := m.manager.RetryAndList()

	for _, client := range clients {
		list, err := client.ListContainers()
		if err != nil {
			log.Debugf("error listing containers for host %s: %v", client.Host().ID, err)
			host := client.Host()
			host.Available = false
			errors = append(errors, &HostUnavailableError{Host: host, Err: err})
			continue
		}

		containers = append(containers, list...)
	}

	return containers, errors
}

func (m *MultiHostService) ListAllContainersFiltered(filter ContainerFilter) ([]docker.Container, []error) {
	containers, err := m.ListAllContainers()
	filtered := make([]docker.Container, 0, len(containers))
	for _, container := range containers {
		if filter(&container) {
			filtered = append(filtered, container)
		}
	}
	return filtered, err
}

func (m *MultiHostService) SubscribeEventsAndStats(ctx context.Context, events chan<- docker.ContainerEvent, stats chan<- docker.ContainerStat) {
	for _, client := range m.manager.List() {
		client.SubscribeEvents(ctx, events)
		client.SubscribeStats(ctx, stats)
	}
}

func (m *MultiHostService) SubscribeContainersStarted(ctx context.Context, containers chan<- docker.Container, filter ContainerFilter) {
	newContainers := make(chan docker.Container)
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

func (m *MultiHostService) TotalClients() int {
	return len(m.manager.List())
}

func (m *MultiHostService) Hosts() []docker.Host {
	clients := m.manager.List()
	hosts := make([]docker.Host, 0, len(clients))
	for _, client := range clients {
		host := client.Host()
		host.Available = true
		hosts = append(hosts, host)
	}

	for _, endpoint := range m.manager.FailedAgents() {
		hosts = append(hosts, docker.Host{
			ID:        endpoint,
			Name:      endpoint,
			Endpoint:  endpoint,
			Available: false,
			Type:      "agent",
		})
	}

	return hosts
}

func (m *MultiHostService) LocalHost() (docker.Host, error) {
	for _, host := range m.Hosts() {
		if host.Type == "local" {
			return host, nil
		}
	}
	return docker.Host{}, fmt.Errorf("local host not found")
}

func (m *MultiHostService) SubscribeAvailableHosts(ctx context.Context, hosts chan<- docker.Host) {
	m.manager.Subscribe(ctx, hosts)
}

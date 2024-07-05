package docker_support

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"

	"github.com/amir20/dozzle/internal/agent"
	"github.com/amir20/dozzle/internal/docker"
	log "github.com/sirupsen/logrus"

	"github.com/cenkalti/backoff/v4"
)

type ContainerFilter = func(*docker.Container) bool

type HostUnavailableError struct {
	Host docker.Host
	Err  error
}

func (h *HostUnavailableError) Error() string {
	return fmt.Sprintf("host %s unavailable: %v", h.Host.ID, h.Err)
}

type MultiHostService struct {
	clients   map[string]ClientService
	SwarmMode bool
}

func NewMultiHostService(clients []ClientService) *MultiHostService {
	m := &MultiHostService{
		clients: make(map[string]ClientService),
	}

	for _, client := range clients {
		if _, ok := m.clients[client.Host().ID]; ok {
			log.Warnf("duplicate host %s found, skipping", client.Host())
			continue
		}
		m.clients[client.Host().ID] = client
	}

	return m
}

func NewSwarmService(client docker.Client, certificates tls.Certificate) *MultiHostService {
	m := &MultiHostService{
		clients:   make(map[string]ClientService),
		SwarmMode: true,
	}

	localClient := NewDockerClientService(client)
	m.clients[localClient.Host().ID] = localClient

	discover := func() {
		ips, err := net.LookupIP("tasks.dozzle")
		if err != nil {
			log.Fatalf("error looking up swarm services: %v", err)
		}

		found := 0
		replaced := 0
		for _, ip := range ips {
			client, err := agent.NewClient(ip.String()+":7007", certificates)
			if err != nil {
				log.Warnf("error creating client for %s: %v", ip, err)
				continue
			}

			if client.Host().ID == localClient.Host().ID {
				continue
			}

			service := NewAgentService(client)
			if existing, ok := m.clients[service.Host().ID]; !ok {
				log.Debugf("adding swarm service %s", service.Host().ID)
				m.clients[service.Host().ID] = service
				found++
			} else if existing.Host().Endpoint != service.Host().Endpoint {
				log.Debugf("swarm service %s already exists with different endpoint %s and old value %s", service.Host().ID, service.Host().Endpoint, existing.Host().Endpoint)
				delete(m.clients, existing.Host().ID)
				m.clients[service.Host().ID] = service
				replaced++
			}
		}

		if found > 0 {
			log.Infof("found %d new dozzle replicas", found)
		}
		if replaced > 0 {
			log.Infof("replaced %d dozzle replicas", replaced)
		}
	}

	go func() {
		ticker := backoff.NewTicker(backoff.NewExponentialBackOff(
			backoff.WithMaxElapsedTime(0)),
		)
		for range ticker.C {
			log.Tracef("discovering swarm services")
			discover()
		}
	}()

	return m
}

func (m *MultiHostService) FindContainer(host string, id string) (*containerService, error) {
	client, ok := m.clients[host]
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
	client, ok := m.clients[host]
	if !ok {
		return nil, fmt.Errorf("host %s not found", host)
	}

	return client.ListContainers()
}

func (m *MultiHostService) ListAllContainers() ([]docker.Container, []error) {
	containers := make([]docker.Container, 0)
	var errors []error

	for _, client := range m.clients {
		list, err := client.ListContainers()
		if err != nil {
			log.Debugf("error listing containers for host %s: %v", client.Host().ID, err)
			errors = append(errors, &HostUnavailableError{Host: client.Host(), Err: err})
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
	for _, client := range m.clients {
		client.SubscribeEvents(ctx, events)
		client.SubscribeStats(ctx, stats)
	}
}

func (m *MultiHostService) SubscribeContainersStarted(ctx context.Context, containers chan<- docker.Container, filter ContainerFilter) {
	newContainers := make(chan docker.Container)
	for _, client := range m.clients {
		client.SubscribeContainersStarted(ctx, newContainers)
	}

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
	return len(m.clients)
}

func (m *MultiHostService) Hosts() []docker.Host {
	hosts := make([]docker.Host, 0, len(m.clients))
	for _, client := range m.clients {
		hosts = append(hosts, client.Host())
	}

	return hosts
}

func (m *MultiHostService) LocalHost() (docker.Host, error) {
	host := docker.Host{}

	for _, host := range m.Hosts() {
		if host.Endpoint == "local" {

			return host, nil
		}
	}

	return host, fmt.Errorf("local host not found")
}

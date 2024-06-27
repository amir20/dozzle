package docker_support

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"time"

	"github.com/amir20/dozzle/internal/agent"
	"github.com/amir20/dozzle/internal/docker"
	log "github.com/sirupsen/logrus"
)

// host unavailability error
type HostUnavailableError struct {
	Host docker.Host
	Err  error
}

func (h *HostUnavailableError) Error() string {
	return fmt.Sprintf("host %s unavailable: %v", h.Host.ID, h.Err)
}

type MultiHostService interface {
	FindContainer(host string, id string) (ContainerService, error)
	ListAllContainers() ([]docker.Container, []error)
	ListContainersForHost(host string) ([]docker.Container, error)
	SubscribeEventsAndStats(ctx context.Context, events chan<- docker.ContainerEvent, stats chan<- docker.ContainerStat)
	UnsubscribeEventsAndStats(ctx context.Context)
	TotalClients() int
	Hosts() []docker.Host
}

type multiHostService struct {
	clients map[string]ClientService
}

func NewMultiHostService(clients []ClientService) MultiHostService {
	m := &multiHostService{
		clients: make(map[string]ClientService),
	}

	for _, client := range clients {
		if _, ok := m.clients[client.Host().ID]; ok {
			log.Warnf("duplicate host %s found, skipping", client.Host().ID)
			continue
		}
		m.clients[client.Host().ID] = client
	}

	return m
}

func NewSwarmService(localClient ClientService, certificates tls.Certificate) MultiHostService {
	m := &multiHostService{
		clients: make(map[string]ClientService),
	}

	m.clients[localClient.Host().ID] = localClient

	go func() {
		log.Debugf("waiting for swarm services to be available")
		time.Sleep(4 * time.Second)
		ips, err := net.LookupIP("tasks.dozzle")
		if err != nil {
			log.Fatalf("error looking up swarm services: %v", err)
		}

		log.Debugf("found %d swarm services", len(ips))

		for _, ip := range ips {
			client, err := agent.NewClient(ip.String()+":7007", certificates)
			log.Debugf("connected to %s", ip)

			if err != nil {
				log.Warnf("error creating client for %s: %v", ip, err)
				continue
			}
			service := NewAgentService(client)
			if service.Host().ID != localClient.Host().ID {
				m.clients[service.Host().ID] = service
			}
		}
	}()

	return m
}

func (m *multiHostService) FindContainer(host string, id string) (ContainerService, error) {
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
		container:     container,
	}, nil
}

func (m *multiHostService) ListContainersForHost(host string) ([]docker.Container, error) {
	client, ok := m.clients[host]
	if !ok {
		return nil, fmt.Errorf("host %s not found", host)
	}

	return client.ListContainers()
}

func (m *multiHostService) ListAllContainers() ([]docker.Container, []error) {
	var containers []docker.Container
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

func (m *multiHostService) SubscribeEventsAndStats(ctx context.Context, events chan<- docker.ContainerEvent, stats chan<- docker.ContainerStat) {
	for _, client := range m.clients {
		client.SubscribeEvents(ctx, events)
		client.SubscribeStats(ctx, stats)
	}
}

func (m *multiHostService) UnsubscribeEventsAndStats(ctx context.Context) {
	for _, client := range m.clients {
		client.UnsubscribeEvents(ctx)
		client.UnsubscribeStats(ctx)
	}
}

func (m *multiHostService) TotalClients() int {
	return len(m.clients)
}

func (m *multiHostService) Hosts() []docker.Host {
	hosts := make([]docker.Host, 0, len(m.clients))
	for _, client := range m.clients {
		hosts = append(hosts, client.Host())
	}

	return hosts
}

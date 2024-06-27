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

func NewSwarmService(client docker.Client, certificates tls.Certificate) MultiHostService {
	m := &multiHostService{
		clients: make(map[string]ClientService),
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
		time.Sleep(5 * time.Second)
		discover()

		for {
			time.Sleep(30 * time.Second)
			log.Tracef("discovering swarm services")
			discover()
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

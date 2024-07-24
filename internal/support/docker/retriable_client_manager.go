package docker_support

import (
	"context"
	"crypto/tls"
	"fmt"
	"sync"

	"github.com/amir20/dozzle/internal/agent"
	"github.com/amir20/dozzle/internal/docker"
	"github.com/puzpuzpuz/xsync/v3"

	log "github.com/sirupsen/logrus"
)

type RetriableClientManager struct {
	clients      map[string]ClientService
	failedAgents []string
	certs        tls.Certificate
	mu           sync.RWMutex
	subscribers  *xsync.MapOf[context.Context, chan<- docker.Host]
}

func NewRetriableClientManager(clients []ClientService, agents []string, certs tls.Certificate) *RetriableClientManager {
	log.Debugf("creating retriable client manager with %d clients and %d agents", len(clients), len(agents))

	clientMap := make(map[string]ClientService)
	for _, client := range clients {
		if _, ok := clientMap[client.Host().ID]; ok {
			log.Warnf("duplicate client found for host %s", client.Host().ID)
		} else {
			clientMap[client.Host().ID] = client
		}
	}

	failed := make([]string, 0)
	for _, endpoint := range agents {
		if agent, err := agent.NewClient(endpoint, certs); err == nil {
			if _, ok := clientMap[agent.Host().ID]; ok {
				log.Warnf("duplicate client found for host %s", agent.Host().ID)
			} else {
				clientMap[agent.Host().ID] = NewAgentService(agent)
			}
		} else {
			log.Warnf("error creating agent client for %s: %v", endpoint, err)
			failed = append(failed, endpoint)
		}
	}

	return &RetriableClientManager{
		clients:      clientMap,
		failedAgents: failed,
		certs:        certs,
		subscribers:  xsync.NewMapOf[context.Context, chan<- docker.Host](),
	}
}

func (m *RetriableClientManager) Subscribe(ctx context.Context, channel chan<- docker.Host) {
	m.subscribers.Store(ctx, channel)

	go func() {
		<-ctx.Done()
		m.subscribers.Delete(ctx)
	}()
}

func (m *RetriableClientManager) RetryAndList() ([]ClientService, []error) {
	m.mu.Lock()
	errors := make([]error, 0)
	if len(m.failedAgents) > 0 {
		newFailed := make([]string, 0)
		for _, endpoint := range m.failedAgents {
			if agent, err := agent.NewClient(endpoint, m.certs); err == nil {
				m.clients[agent.Host().ID] = NewAgentService(agent)

				m.subscribers.Range(func(ctx context.Context, channel chan<- docker.Host) bool {
					host := agent.Host()
					host.Available = true

					// We don't want to block the subscribers in event.go
					go func() {
						select {
						case channel <- host:
						case <-ctx.Done():
						}
					}()

					return true
				})

			} else {
				log.Warnf("error creating agent client for %s: %v", endpoint, err)
				errors = append(errors, err)
				newFailed = append(newFailed, endpoint)
			}
		}
		m.failedAgents = newFailed
	}

	m.mu.Unlock()

	return m.List(), errors
}

func (m *RetriableClientManager) List() []ClientService {
	m.mu.RLock()
	defer m.mu.RUnlock()

	clients := make([]ClientService, 0, len(m.clients))
	for _, client := range m.clients {
		clients = append(clients, client)
	}
	return clients
}

func (m *RetriableClientManager) Find(id string) (ClientService, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	client, ok := m.clients[id]
	return client, ok
}

func (m *RetriableClientManager) String() string {
	return fmt.Sprintf("RetriableClientManager{clients: %d, failedAgents: %d}", len(m.clients), len(m.failedAgents))
}

func (m *RetriableClientManager) Hosts() []docker.Host {
	clients := m.List()

	hosts := make([]docker.Host, 0, len(clients))
	for _, client := range clients {
		host := client.Host()
		host.Available = true
		hosts = append(hosts, host)
	}

	for _, endpoint := range m.failedAgents {
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

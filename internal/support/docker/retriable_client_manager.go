package docker_support

import (
	"context"
	"crypto/tls"
	"fmt"
	"sync"

	"github.com/amir20/dozzle/internal/agent"
	"github.com/amir20/dozzle/internal/docker"
	"github.com/puzpuzpuz/xsync/v3"
	lop "github.com/samber/lo/parallel"

	"github.com/rs/zerolog/log"
)

type RetriableClientManager struct {
	clients      map[string]ClientService
	failedAgents []string
	certs        tls.Certificate
	mu           sync.RWMutex
	subscribers  *xsync.MapOf[context.Context, chan<- docker.Host]
}

func NewRetriableClientManager(agents []string, certs tls.Certificate, clients ...ClientService) *RetriableClientManager {
	clientMap := make(map[string]ClientService)
	for _, client := range clients {
		host, err := client.Host()
		if err != nil {
			log.Warn().Err(err).Str("host", host.Name).Msg("error fetching host info for client")
			continue
		}

		if _, ok := clientMap[host.ID]; ok {
			log.Warn().Str("host", host.Name).Msg("duplicate client found")
		} else {
			clientMap[host.ID] = client
		}
	}

	failed := make([]string, 0)
	for _, endpoint := range agents {
		agent, err := agent.NewClient(endpoint, certs)
		if err != nil {
			log.Warn().Err(err).Str("endpoint", endpoint).Msg("error creating agent client")
			failed = append(failed, endpoint)
			continue
		}

		host, err := agent.Host()
		if err != nil {
			log.Warn().Err(err).Str("endpoint", endpoint).Msg("error fetching host info for agent")
			failed = append(failed, endpoint)
			continue
		}

		if _, ok := clientMap[host.ID]; ok {
			log.Warn().Str("host", host.Name).Str("id", host.ID).Msg("duplicate host with same ID found")
		} else {
			clientMap[host.ID] = NewAgentService(agent)
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
			agent, err := agent.NewClient(endpoint, m.certs)
			if err != nil {
				log.Warn().Err(err).Str("endpoint", endpoint).Msg("error creating agent client")
				errors = append(errors, err)
				newFailed = append(newFailed, endpoint)
				continue
			}

			host, err := agent.Host()
			if err != nil {
				log.Warn().Err(err).Str("endpoint", endpoint).Msg("error fetching host info for agent")
				errors = append(errors, err)
				newFailed = append(newFailed, endpoint)
				continue
			}

			m.clients[host.ID] = NewAgentService(agent)
			m.subscribers.Range(func(ctx context.Context, channel chan<- docker.Host) bool {
				host.Available = true

				// We don't want to block the subscribers in event.go
				go func(host docker.Host) {
					select {
					case channel <- host:
					case <-ctx.Done():
					}
				}(host)

				return true
			})
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

	hosts := lop.Map(clients, func(client ClientService, _ int) docker.Host {
		host, err := client.Host()
		if err != nil {
			host.Available = false
		} else {
			host.Available = true
		}

		return host
	})

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

package docker_support

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/amir20/dozzle/internal/agent"
	"github.com/amir20/dozzle/internal/container"
	container_support "github.com/amir20/dozzle/internal/support/container"

	"github.com/puzpuzpuz/xsync/v4"
	"github.com/samber/lo"
	lop "github.com/samber/lo/parallel"

	"github.com/rs/zerolog/log"
)

type RetriableClientManager struct {
	clients      map[string]container_support.ClientService
	failedAgents []string
	certs        tls.Certificate
	mu           sync.RWMutex
	subscribers  *xsync.Map[context.Context, chan<- container.Host]
	timeout      time.Duration
}

func NewRetriableClientManager(agents []string, timeout time.Duration, certs tls.Certificate, clients ...container_support.ClientService) *RetriableClientManager {
	type entry struct {
		host    container.Host
		service container_support.ClientService
		failed  string // endpoint, set only for failed agents
		ok      bool
	}

	results := make([]entry, len(clients)+len(agents))
	var wg sync.WaitGroup

	for i, c := range clients {
		wg.Go(func() {
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			host, err := c.Host(ctx)
			if err != nil {
				log.Warn().Err(err).Msg("error fetching host info for client")
				return
			}
			results[i] = entry{host: host, service: c, ok: true}
		})
	}

	for i, endpoint := range agents {
		idx := len(clients) + i
		wg.Go(func() {
			a, err := agent.NewClient(endpoint, certs)
			if err != nil {
				log.Warn().Err(err).Str("endpoint", endpoint).Msg("error creating agent client")
				results[idx] = entry{failed: endpoint}
				return
			}
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			host, err := a.Host(ctx)
			if err != nil {
				log.Warn().Err(err).Str("endpoint", endpoint).Msg("error fetching host info for agent")
				results[idx] = entry{failed: endpoint}
				return
			}
			results[idx] = entry{host: host, service: container_support.NewAgentService(a), ok: true}
		})
	}

	wg.Wait()

	clientMap := make(map[string]container_support.ClientService)
	failedList := make([]string, 0)
	for _, r := range results {
		if r.failed != "" {
			failedList = append(failedList, r.failed)
			continue
		}
		if !r.ok {
			continue
		}
		if _, exists := clientMap[r.host.ID]; exists {
			log.Warn().Str("name", r.host.Name).Str("id", r.host.ID).Msg("An agent with an existing ID was found. Removing the duplicate host. For more details, see http://localhost:5173/guide/agent#agent-not-showing-up.")
			continue
		}
		clientMap[r.host.ID] = r.service
	}

	return &RetriableClientManager{
		clients:      clientMap,
		failedAgents: failedList,
		certs:        certs,
		subscribers:  xsync.NewMap[context.Context, chan<- container.Host](),
		timeout:      timeout,
	}
}

func (m *RetriableClientManager) Subscribe(ctx context.Context, channel chan<- container.Host) {
	m.subscribers.Store(ctx, channel)

	go func() {
		<-ctx.Done()
		m.subscribers.Delete(ctx)
	}()
}

func (m *RetriableClientManager) RetryAndList() ([]container_support.ClientService, []error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.failedAgents) == 0 {
		return lo.Values(m.clients), nil
	}

	type retryResult struct {
		endpoint string
		host     container.Host
		service  container_support.ClientService
		err      error
	}

	results := make([]retryResult, len(m.failedAgents))
	var wg sync.WaitGroup
	for i, endpoint := range m.failedAgents {
		wg.Go(func() {
			a, err := agent.NewClient(endpoint, m.certs)
			if err != nil {
				log.Warn().Err(err).Str("endpoint", endpoint).Msg("error creating agent client")
				results[i] = retryResult{endpoint: endpoint, err: err}
				return
			}
			ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
			defer cancel()
			h, err := a.Host(ctx)
			if err != nil {
				log.Warn().Err(err).Str("endpoint", endpoint).Msg("error fetching host info for agent")
				results[i] = retryResult{endpoint: endpoint, err: err}
				return
			}
			results[i] = retryResult{
				endpoint: endpoint,
				host:     h,
				service:  container_support.NewAgentService(a),
			}
		})
	}
	wg.Wait()

	var errs []error
	newFailed := make([]string, 0)
	for _, r := range results {
		if r.err != nil {
			errs = append(errs, r.err)
			newFailed = append(newFailed, r.endpoint)
			continue
		}
		if _, ok := m.clients[r.host.ID]; ok {
			log.Warn().Str("name", r.host.Name).Str("id", r.host.ID).Msg("An agent with an existing ID was found. Removing the duplicate host. For more details, see http://localhost:5173/guide/agent#agent-not-showing-up.")
			continue
		}
		m.clients[r.host.ID] = r.service
		host := r.host
		host.Available = true
		m.subscribers.Range(func(ctx context.Context, channel chan<- container.Host) bool {
			// We don't want to block the subscribers in event.go
			go func() {
				select {
				case channel <- host:
				case <-ctx.Done():
				}
			}()
			return true
		})
	}
	m.failedAgents = newFailed

	return lo.Values(m.clients), errs
}

func (m *RetriableClientManager) List() []container_support.ClientService {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return lo.Values(m.clients)
}

func (m *RetriableClientManager) Find(id string) (container_support.ClientService, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	client, ok := m.clients[id]
	return client, ok
}

func (m *RetriableClientManager) String() string {
	return fmt.Sprintf("RetriableClientManager{clients: %d, failedAgents: %d}", len(m.clients), len(m.failedAgents))
}

func (m *RetriableClientManager) Hosts(ctx context.Context) []container.Host {
	clients := m.List()

	hosts := lop.Map(clients, func(client container_support.ClientService, _ int) container.Host {
		host, err := client.Host(ctx)
		if err != nil {
			log.Warn().Err(err).Str("host", host.Name).Msg("error fetching host info for client")
			host.Available = false
		} else {
			host.Available = true
		}

		return host
	})

	for _, endpoint := range m.failedAgents {
		parts := strings.SplitN(endpoint, "|", 3)
		addr := parts[0]
		name := addr
		group := ""
		if len(parts) >= 2 && parts[1] != "" {
			name = parts[1]
		}
		if len(parts) == 3 {
			group = parts[2]
		}
		hosts = append(hosts, container.Host{
			ID:        endpoint,
			Name:      name,
			Endpoint:  addr,
			Available: false,
			Type:      "agent",
			Group:     group,
		})
	}

	return hosts
}

func (m *RetriableClientManager) LocalClients() []container.Client {
	services := m.List()

	clients := make([]container.Client, 0)

	for _, service := range services {
		if clientService, ok := service.(*DockerClientService); ok {
			clients = append(clients, clientService.client)
		}
	}

	return clients
}

func (m *RetriableClientManager) LocalClientServices() []container_support.ClientService {
	services := m.List()

	result := make([]container_support.ClientService, 0)

	for _, service := range services {
		if _, ok := service.(*DockerClientService); ok {
			result = append(result, service)
		}
	}

	return result
}

package docker_support

import (
	"context"
	"crypto/tls"
	"fmt"
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
	clientMap := make(map[string]container_support.ClientService)
	var mapMu sync.Mutex
	var wg sync.WaitGroup

	for _, c := range clients {
		wg.Add(1)
		go func(c container_support.ClientService) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			host, err := c.Host(ctx)
			if err != nil {
				log.Warn().Err(err).Str("host", host.Name).Msg("error fetching host info for client")
				return
			}
			mapMu.Lock()
			defer mapMu.Unlock()
			if _, ok := clientMap[host.ID]; ok {
				log.Warn().Str("name", host.Name).Str("id", host.ID).Msg("An agent with an existing ID was found. Removing the duplicate host. For more details, see http://localhost:5173/guide/agent#agent-not-showing-up.")
			} else {
				clientMap[host.ID] = c
			}
		}(c)
	}

	type agentEntry struct {
		hostID   string
		hostName string
		service  container_support.ClientService
	}

	succeeded := make(chan agentEntry, len(agents))
	failedCh := make(chan string, len(agents))

	for _, endpoint := range agents {
		wg.Add(1)
		go func(ep string) {
			defer wg.Done()
			a, err := agent.NewClient(ep, certs)
			if err != nil {
				log.Warn().Err(err).Str("endpoint", ep).Msg("error creating agent client")
				failedCh <- ep
				return
			}
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			host, err := a.Host(ctx)
			if err != nil {
				log.Warn().Err(err).Str("endpoint", ep).Msg("error fetching host info for agent")
				failedCh <- ep
				return
			}
			succeeded <- agentEntry{hostID: host.ID, hostName: host.Name, service: container_support.NewAgentService(a)}
		}(endpoint)
	}

	wg.Wait()
	close(succeeded)
	close(failedCh)

	failedList := make([]string, 0)
	for ep := range failedCh {
		failedList = append(failedList, ep)
	}
	for entry := range succeeded {
		if _, ok := clientMap[entry.hostID]; ok {
			log.Warn().Str("name", entry.hostName).Str("id", entry.hostID).Msg("An agent with an existing ID was found. Removing the duplicate host. For more details, see http://localhost:5173/guide/agent#agent-not-showing-up.")
		} else {
			clientMap[entry.hostID] = entry.service
		}
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
	if len(m.failedAgents) == 0 {
		m.mu.Unlock()
		return m.List(), nil
	}
	endpoints := make([]string, len(m.failedAgents))
	copy(endpoints, m.failedAgents)
	m.mu.Unlock()

	type retryResult struct {
		endpoint string
		hostID   string
		hostName string
		service  container_support.ClientService
		host     container.Host
		err      error
	}

	results := make([]retryResult, len(endpoints))
	var wg sync.WaitGroup
	for i, ep := range endpoints {
		wg.Add(1)
		go func(i int, ep string) {
			defer wg.Done()
			a, err := agent.NewClient(ep, m.certs)
			if err != nil {
				log.Warn().Err(err).Str("endpoint", ep).Msg("error creating agent client")
				results[i] = retryResult{endpoint: ep, err: err}
				return
			}
			ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
			defer cancel()
			h, err := a.Host(ctx)
			if err != nil {
				log.Warn().Err(err).Str("endpoint", ep).Msg("error fetching host info for agent")
				results[i] = retryResult{endpoint: ep, err: err}
				return
			}
			results[i] = retryResult{
				endpoint: ep,
				hostID:   h.ID,
				hostName: h.Name,
				service:  container_support.NewAgentService(a),
				host:     h,
			}
		}(i, ep)
	}
	wg.Wait()

	var errs []error
	var newFailed []string

	m.mu.Lock()
	for _, r := range results {
		if r.err != nil {
			errs = append(errs, r.err)
			newFailed = append(newFailed, r.endpoint)
			continue
		}
		if _, ok := m.clients[r.hostID]; ok {
			log.Warn().Str("name", r.hostName).Str("id", r.hostID).Msg("An agent with an existing ID was found. Removing the duplicate host. For more details, see http://localhost:5173/guide/agent#agent-not-showing-up.")
		} else {
			m.clients[r.hostID] = r.service
		}
		h := r.host
		h.Available = true
		m.subscribers.Range(func(ctx context.Context, channel chan<- container.Host) bool {
			// We don't want to block the subscribers in event.go
			go func(host container.Host) {
				select {
				case channel <- host:
				case <-ctx.Done():
				}
			}(h)
			return true
		})
	}
	retriedSet := make(map[string]bool, len(endpoints))
	for _, ep := range endpoints {
		retriedSet[ep] = true
	}
	remaining := make([]string, 0)
	for _, ep := range m.failedAgents {
		if !retriedSet[ep] {
			remaining = append(remaining, ep)
		}
	}
	m.failedAgents = append(remaining, newFailed...)
	m.mu.Unlock()

	return m.List(), errs
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
		hosts = append(hosts, container.Host{
			ID:        endpoint,
			Name:      endpoint,
			Endpoint:  endpoint,
			Available: false,
			Type:      "agent",
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

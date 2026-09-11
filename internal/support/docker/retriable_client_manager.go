package docker_support

import (
	"context"
	"crypto/tls"
	"fmt"
	"slices"
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

// hostSubscriber is one registered listener for "a host became available".
//
// Subscriptions are keyed by this pointer rather than by their context: one
// caller may open several subscriptions under a single context (the cloud
// client's log and stat streamers share one stream lifetime), and keying by
// context would let the second registration silently evict the first.
type hostSubscriber struct {
	ctx     context.Context
	channel chan<- container.Host
}

type RetriableClientManager struct {
	clients      map[string]container_support.ClientService
	failedAgents []string
	certs        tls.Certificate
	mu           sync.RWMutex
	subscribers  *xsync.Map[*hostSubscriber, struct{}]
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
		subscribers:  xsync.NewMap[*hostSubscriber, struct{}](),
		timeout:      timeout,
	}
}

func (m *RetriableClientManager) Subscribe(ctx context.Context, channel chan<- container.Host) {
	sub := &hostSubscriber{ctx: ctx, channel: channel}
	m.subscribers.Store(sub, struct{}{})

	go func() {
		<-ctx.Done()
		m.subscribers.Delete(sub)
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
		m.publish(host)
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

// rekey moves a client to the id its host now reports, so Find keeps resolving
// after the agent behind it restarted under a new id. It returns the host to
// report, which carries the id it was previously known by so an open tab can
// drop the stale entry instead of showing the same machine twice until reload.
func (m *RetriableClientManager) rekey(oldID string, service container_support.ClientService, host container.Host) container.Host {
	m.mu.Lock()
	if current, ok := m.clients[oldID]; !ok || current != service {
		// A concurrent Hosts() already repaired this one.
		m.mu.Unlock()
		return host
	}
	if existing, taken := m.clients[host.ID]; taken && existing != service {
		m.mu.Unlock()
		log.Warn().Str("name", host.Name).Str("id", host.ID).Str("previousId", oldID).Msg("host reported an id that already belongs to another host, leaving both in place. See https://dozzle.dev/guide/faq#i-am-seeing-duplicate-hosts-error-in-the-logs-how-do-i-fix-it")
		return host
	}
	delete(m.clients, oldID)
	m.clients[host.ID] = service
	m.mu.Unlock()

	log.Info().Str("name", host.Name).Str("id", host.ID).Str("previousId", oldID).Msg("host came back with a new id, updating routing")

	host.ReplacesID = oldID
	m.publish(host)

	return host
}

// publish tells every subscriber about a host, without blocking this caller on
// a slow one. Mirrors what RetryAndList does when an agent first comes back.
func (m *RetriableClientManager) publish(host container.Host) {
	m.subscribers.Range(func(sub *hostSubscriber, _ struct{}) bool {
		go func() {
			select {
			case sub.channel <- host:
			case <-sub.ctx.Done():
			}
		}()
		return true
	})
}

func (m *RetriableClientManager) Hosts(ctx context.Context) []container.Host {
	m.mu.RLock()
	type entry struct {
		id      string
		service container_support.ClientService
	}
	entries := make([]entry, 0, len(m.clients))
	for id, service := range m.clients {
		entries = append(entries, entry{id: id, service: service})
	}
	failedAgents := slices.Clone(m.failedAgents)
	m.mu.RUnlock()

	type result struct {
		entry entry
		host  container.Host
	}

	results := lop.Map(entries, func(e entry, _ int) result {
		host, err := e.service.Host(ctx)
		if err != nil {
			log.Warn().Err(err).Str("host", host.Name).Msg("error fetching host info for client")
			host.Available = false
		} else {
			host.Available = true
		}

		return result{entry: e, host: host}
	})

	hosts := make([]container.Host, 0, len(results)+len(failedAgents))
	for _, r := range results {
		// An agent mints its id when its own process starts, so an agent that
		// restarted since we last looked answers under a different id than the
		// one this map is keyed by. Nothing else notices: RetryAndList only ever
		// revisits endpoints that never connected. Left alone, the id we hand the
		// UI here is one Find has never heard of, and every lookup for that host
		// fails with "host not found" until the hub itself is restarted.
		if r.host.Available && r.host.ID != "" && r.host.ID != r.entry.id {
			r.host = m.rekey(r.entry.id, r.entry.service, r.host)
		}
		hosts = append(hosts, r.host)
	}

	for _, endpoint := range failedAgents {
		addr, name, group, err := agent.ParseEndpoint(endpoint)
		if err != nil {
			log.Warn().Err(err).Str("endpoint", endpoint).Msg("skipping malformed agent endpoint")
			continue
		}
		if name == "" {
			name = addr
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

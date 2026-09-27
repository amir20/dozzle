package hostservice

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"slices"
	"sync"
	"time"

	"github.com/amir20/dozzle/internal/analytics"
	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/container/agent"
	"github.com/amir20/dozzle/internal/container/docker"

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
	clients      map[string]container.ClientService
	failedAgents []string
	// agents maps each connected agent's endpoint to the host id it is filed
	// under in clients, so one can be removed by the endpoint it was added with.
	agents map[string]connectedAgent
	// agentCerts overrides certs for the agents that were given the hub's
	// private pair. Every other agent keeps the pair the hub was started with.
	agentCerts  map[string]tls.Certificate
	certs       tls.Certificate
	mu          sync.RWMutex
	subscribers *xsync.Map[*hostSubscriber, struct{}]
	timeout     time.Duration
	dial        agentDialer

	// wasAvailable remembers each agent's last answer, so a disconnect is counted
	// once when it happens rather than on every look while it stays down.
	wasAvailable *xsync.Map[string, bool]
}

type connectedAgent struct {
	id     string
	closer io.Closer
}

// agentDialer opens a client for an agent endpoint without contacting it yet.
// A seam so tests can add agents without a TLS server.
type agentDialer func(endpoint string, certs tls.Certificate) (container.ClientService, io.Closer, error)

func dialAgent(endpoint string, certs tls.Certificate) (container.ClientService, io.Closer, error) {
	a, err := agent.NewClient(endpoint, certs)
	if err != nil {
		return nil, nil, err
	}
	return agent.NewService(a), a, nil
}

var (
	ErrAgentExists   = errors.New("agent is already added")
	ErrAgentNotFound = errors.New("agent not found")
	ErrDuplicateHost = errors.New("another connected host already has this agent's id")
)

func NewRetriableClientManager(agents []string, timeout time.Duration, certs tls.Certificate, clients ...container.ClientService) *RetriableClientManager {
	return newRetriableClientManager(agents, nil, timeout, certs, dialAgent, clients...)
}

// NewRetriableClientManagerWithAgentCerts is NewRetriableClientManager where
// some agents authenticate with a pair of their own instead of certs.
func NewRetriableClientManagerWithAgentCerts(agents []string, agentCerts map[string]tls.Certificate, timeout time.Duration, certs tls.Certificate, clients ...container.ClientService) *RetriableClientManager {
	return newRetriableClientManager(agents, agentCerts, timeout, certs, dialAgent, clients...)
}

func newRetriableClientManager(agents []string, agentCerts map[string]tls.Certificate, timeout time.Duration, certs tls.Certificate, dial agentDialer, clients ...container.ClientService) *RetriableClientManager {
	if agentCerts == nil {
		agentCerts = map[string]tls.Certificate{}
	}
	certFor := func(endpoint string) tls.Certificate {
		if c, ok := agentCerts[endpoint]; ok {
			return c
		}
		return certs
	}
	type entry struct {
		host     container.Host
		service  container.ClientService
		closer   io.Closer
		endpoint string // set only for agents
		failed   string // endpoint, set only for failed agents
		ok       bool
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
			service, closer, err := dial(endpoint, certFor(endpoint))
			if err != nil {
				log.Warn().Err(err).Str("endpoint", endpoint).Msg("error creating agent client")
				results[idx] = entry{failed: endpoint}
				return
			}
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			host, err := service.Host(ctx)
			if err != nil {
				log.Warn().Err(err).Str("endpoint", endpoint).Msg("error fetching host info for agent")
				results[idx] = entry{failed: endpoint}
				return
			}
			results[idx] = entry{host: host, service: service, closer: closer, endpoint: endpoint, ok: true}
		})
	}

	wg.Wait()

	clientMap := make(map[string]container.ClientService)
	agentMap := make(map[string]connectedAgent)
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
			log.Warn().Str("name", r.host.Name).Str("id", r.host.ID).Msg("An agent with an existing ID was found. Removing the duplicate host. For more details, see https://dozzle.dev/guide/faq#i-am-seeing-duplicate-hosts-error-in-the-logs-how-do-i-fix-it")
			continue
		}
		clientMap[r.host.ID] = r.service
		if r.endpoint != "" {
			agentMap[r.endpoint] = connectedAgent{id: r.host.ID, closer: r.closer}
		}
	}

	return &RetriableClientManager{
		clients:      clientMap,
		failedAgents: failedList,
		agents:       agentMap,
		agentCerts:   agentCerts,
		certs:        certs,
		subscribers:  xsync.NewMap[*hostSubscriber, struct{}](),
		timeout:      timeout,
		dial:         dial,
		wasAvailable: xsync.NewMap[string, bool](),
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

func (m *RetriableClientManager) RetryAndList() ([]container.ClientService, []error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.failedAgents) == 0 {
		return lo.Values(m.clients), nil
	}

	type retryResult struct {
		endpoint string
		host     container.Host
		service  container.ClientService
		closer   io.Closer
		err      error
	}

	results := make([]retryResult, len(m.failedAgents))
	var wg sync.WaitGroup
	for i, endpoint := range m.failedAgents {
		wg.Go(func() {
			service, closer, err := m.dial(endpoint, m.certFor(endpoint))
			if err != nil {
				log.Warn().Err(err).Str("endpoint", endpoint).Msg("error creating agent client")
				results[i] = retryResult{endpoint: endpoint, err: err}
				return
			}
			ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
			defer cancel()
			h, err := service.Host(ctx)
			if err != nil {
				closeQuietly(closer)
				log.Warn().Err(err).Str("endpoint", endpoint).Msg("error fetching host info for agent")
				results[i] = retryResult{endpoint: endpoint, err: err}
				return
			}
			results[i] = retryResult{
				endpoint: endpoint,
				host:     h,
				service:  service,
				closer:   closer,
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
			closeQuietly(r.closer)
			log.Warn().Str("name", r.host.Name).Str("id", r.host.ID).Msg("An agent with an existing ID was found. Removing the duplicate host. For more details, see https://dozzle.dev/guide/faq#i-am-seeing-duplicate-hosts-error-in-the-logs-how-do-i-fix-it")
			continue
		}
		m.clients[r.host.ID] = r.service
		m.agents[r.endpoint] = connectedAgent{id: r.host.ID, closer: r.closer}
		host := r.host
		host.Available = true
		m.publish(host)
	}
	m.failedAgents = newFailed

	return lo.Values(m.clients), errs
}

// AddAgent connects to an agent and, only once it answers, starts serving it
// the same way an agent that came back on RetryAndList is served. Nothing is
// kept when it fails, so a typo never lingers as an unavailable host. cert is
// the pair to present to this agent, nil for the one the hub started with.
func (m *RetriableClientManager) AddAgent(ctx context.Context, endpoint string, cert *tls.Certificate) (container.Host, error) {
	if _, _, _, err := agent.ParseEndpoint(endpoint); err != nil {
		return container.Host{}, err
	}

	m.mu.RLock()
	_, connected := m.agents[endpoint]
	failed := slices.Contains(m.failedAgents, endpoint)
	m.mu.RUnlock()
	if connected || failed {
		return container.Host{}, ErrAgentExists
	}

	pair := m.certs
	if cert != nil {
		pair = *cert
	}
	service, closer, err := m.dial(endpoint, pair)
	if err != nil {
		return container.Host{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()
	host, err := service.Host(ctx)
	if err != nil {
		closeQuietly(closer)
		return container.Host{}, err
	}

	m.mu.Lock()
	// Checked again: the dial above ran without the lock.
	if _, ok := m.agents[endpoint]; ok || slices.Contains(m.failedAgents, endpoint) {
		m.mu.Unlock()
		closeQuietly(closer)
		return container.Host{}, ErrAgentExists
	}
	if _, ok := m.clients[host.ID]; ok {
		m.mu.Unlock()
		closeQuietly(closer)
		return container.Host{}, ErrDuplicateHost
	}
	m.clients[host.ID] = service
	m.agents[endpoint] = connectedAgent{id: host.ID, closer: closer}
	if cert != nil {
		m.agentCerts[endpoint] = *cert
	}
	m.mu.Unlock()

	log.Info().Str("endpoint", endpoint).Str("name", host.Name).Str("id", host.ID).Msg("agent added")
	host.Available = true
	m.publish(host)
	return host, nil
}

// RemoveAgent stops serving an agent added by endpoint, connected or not.
// Open streams to it end when its connection closes.
func (m *RetriableClientManager) RemoveAgent(endpoint string) error {
	m.mu.Lock()
	delete(m.agentCerts, endpoint)
	if i := slices.Index(m.failedAgents, endpoint); i >= 0 {
		m.failedAgents = slices.Delete(m.failedAgents, i, i+1)
		m.mu.Unlock()
		// Never connected, so the UI lists it under its endpoint.
		m.publish(container.Host{ID: endpoint, Endpoint: endpoint, Type: "agent", Removed: true})
		return nil
	}
	a, ok := m.agents[endpoint]
	if !ok {
		m.mu.Unlock()
		return ErrAgentNotFound
	}
	delete(m.agents, endpoint)
	delete(m.clients, a.id)
	m.mu.Unlock()

	closeQuietly(a.closer)
	m.publish(container.Host{ID: a.id, Endpoint: endpoint, Type: "agent", Removed: true})
	log.Info().Str("endpoint", endpoint).Str("id", a.id).Msg("agent removed")
	return nil
}

// certFor is the pair to present to endpoint. Callers hold m.mu.
func (m *RetriableClientManager) certFor(endpoint string) tls.Certificate {
	if c, ok := m.agentCerts[endpoint]; ok {
		return c
	}
	return m.certs
}

// AgentHostID returns the id of the host an endpoint is connected as, or ""
// when it is not connected.
func (m *RetriableClientManager) AgentHostID(endpoint string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.agents[endpoint].id
}

func closeQuietly(c io.Closer) {
	if c == nil {
		return
	}
	if err := c.Close(); err != nil {
		log.Debug().Err(err).Msg("error closing agent client")
	}
}

func (m *RetriableClientManager) List() []container.ClientService {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return lo.Values(m.clients)
}

func (m *RetriableClientManager) Find(id string) (container.ClientService, bool) {
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
func (m *RetriableClientManager) rekey(oldID string, service container.ClientService, host container.Host) container.Host {
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
	for endpoint, a := range m.agents {
		if a.id == oldID {
			m.agents[endpoint] = connectedAgent{id: host.ID, closer: a.closer}
		}
	}
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
		service container.ClientService
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
		if r.host.Type == "agent" {
			if prev, ok := m.wasAvailable.Load(r.entry.id); ok && prev && !r.host.Available {
				analytics.Count("agent.disconnect")
			}
			m.wasAvailable.Store(r.entry.id, r.host.Available)
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
		if clientService, ok := service.(*docker.Service); ok {
			clients = append(clients, clientService.Client())
		}
	}

	return clients
}

func (m *RetriableClientManager) LocalClientServices() []container.ClientService {
	services := m.List()

	result := make([]container.ClientService, 0)

	for _, service := range services {
		if _, ok := service.(*docker.Service); ok {
			result = append(result, service)
		}
	}

	return result
}

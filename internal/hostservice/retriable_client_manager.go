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
//
// Each subscriber has its own queue drained by one goroutine, so publish never
// blocks on a slow subscriber and updates still arrive in the order they were
// published (an agent added then removed must not read as removed then added).
type hostSubscriber struct {
	ctx     context.Context
	channel chan<- container.Host

	mu      sync.Mutex
	pending []container.Host
	wake    chan struct{}
}

func (s *hostSubscriber) enqueue(host container.Host) {
	s.mu.Lock()
	// Only a host's latest state matters, so a subscriber that stops reading
	// holds at most one update per host instead of every one since.
	s.pending = slices.DeleteFunc(s.pending, func(h container.Host) bool { return h.ID == host.ID })
	s.pending = append(s.pending, host)
	s.mu.Unlock()
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

func (s *hostSubscriber) drain() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-s.wake:
		}
		s.mu.Lock()
		batch := s.pending
		s.pending = nil
		s.mu.Unlock()
		for _, host := range batch {
			select {
			case s.channel <- host:
			case <-s.ctx.Done():
				return
			}
		}
	}
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
	retryMu     sync.Mutex
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

// NewRetriableClientManager dials agents with certs, except those in agentCerts
// (may be nil), which authenticate with a pair of their own.
func NewRetriableClientManager(agents []string, agentCerts map[string]tls.Certificate, timeout time.Duration, certs tls.Certificate, clients ...container.ClientService) *RetriableClientManager {
	return newRetriableClientManager(agents, agentCerts, timeout, certs, dialAgent, clients...)
}

func newRetriableClientManager(agents []string, agentCerts map[string]tls.Certificate, timeout time.Duration, certs tls.Certificate, dial agentDialer, clients ...container.ClientService) *RetriableClientManager {
	if agentCerts == nil {
		agentCerts = map[string]tls.Certificate{}
	}
	m := &RetriableClientManager{
		clients:      make(map[string]container.ClientService),
		failedAgents: make([]string, 0),
		agents:       make(map[string]connectedAgent),
		agentCerts:   agentCerts,
		certs:        certs,
		subscribers:  xsync.NewMap[*hostSubscriber, struct{}](),
		timeout:      timeout,
		dial:         dial,
		wasAvailable: xsync.NewMap[string, bool](),
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
			host, service, closer, err := m.connect(context.Background(), endpoint, m.certFor(endpoint))
			if err != nil {
				log.Warn().Err(err).Str("endpoint", endpoint).Msg("error connecting to agent")
				results[idx] = entry{failed: endpoint}
				return
			}
			results[idx] = entry{host: host, service: service, closer: closer, endpoint: endpoint, ok: true}
		})
	}

	wg.Wait()

	for _, r := range results {
		if r.failed != "" {
			m.failedAgents = append(m.failedAgents, r.failed)
			continue
		}
		if !r.ok {
			continue
		}
		if _, exists := m.clients[r.host.ID]; exists {
			closeQuietly(r.closer)
			log.Warn().Str("name", r.host.Name).Str("id", r.host.ID).Msg("An agent with an existing ID was found. Removing the duplicate host. For more details, see https://dozzle.dev/guide/faq#i-am-seeing-duplicate-hosts-error-in-the-logs-how-do-i-fix-it")
			continue
		}
		m.clients[r.host.ID] = r.service
		if r.endpoint != "" {
			m.agents[r.endpoint] = connectedAgent{id: r.host.ID, closer: r.closer}
		}
	}

	return m
}

// connect dials an agent and asks it for its host, closing the client again
// when it does not answer within the timeout.
func (m *RetriableClientManager) connect(ctx context.Context, endpoint string, cert tls.Certificate) (container.Host, container.ClientService, io.Closer, error) {
	service, closer, err := m.dial(endpoint, cert)
	if err != nil {
		return container.Host{}, nil, nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()
	host, err := service.Host(ctx)
	if err != nil {
		closeQuietly(closer)
		return container.Host{}, nil, nil, err
	}
	return host, service, closer, nil
}

func (m *RetriableClientManager) Subscribe(ctx context.Context, channel chan<- container.Host) {
	sub := &hostSubscriber{ctx: ctx, channel: channel, wake: make(chan struct{}, 1)}
	m.subscribers.Store(sub, struct{}{})

	go func() {
		sub.drain()
		m.subscribers.Delete(sub)
	}()
}

func (m *RetriableClientManager) RetryAndList() ([]container.ClientService, []error) {
	// Retries run one at a time, but the dials below happen without m.mu, so a
	// down agent never holds up List, Find or AddAgent for the dial timeout.
	m.retryMu.Lock()
	defer m.retryMu.Unlock()

	m.mu.RLock()
	type attempt struct {
		endpoint string
		cert     tls.Certificate
	}
	attempts := make([]attempt, len(m.failedAgents))
	for i, endpoint := range m.failedAgents {
		attempts[i] = attempt{endpoint: endpoint, cert: m.certFor(endpoint)}
	}
	m.mu.RUnlock()

	if len(attempts) == 0 {
		return m.List(), nil
	}

	type retryResult struct {
		endpoint string
		host     container.Host
		service  container.ClientService
		closer   io.Closer
		err      error
	}

	results := make([]retryResult, len(attempts))
	var wg sync.WaitGroup
	for i, a := range attempts {
		wg.Go(func() {
			h, service, closer, err := m.connect(context.Background(), a.endpoint, a.cert)
			if err != nil {
				log.Warn().Err(err).Str("endpoint", a.endpoint).Msg("error connecting to agent")
			}
			results[i] = retryResult{endpoint: a.endpoint, host: h, service: service, closer: closer, err: err}
		})
	}
	wg.Wait()

	var errs []error
	var published []container.Host
	m.mu.Lock()
	for _, r := range results {
		if r.err != nil {
			errs = append(errs, r.err)
			continue
		}
		i := slices.Index(m.failedAgents, r.endpoint)
		if i < 0 {
			// Removed while it was being dialed.
			closeQuietly(r.closer)
			continue
		}
		// Connected or a duplicate, it is no longer retried.
		m.failedAgents = slices.Delete(m.failedAgents, i, i+1)
		if _, ok := m.clients[r.host.ID]; ok {
			closeQuietly(r.closer)
			log.Warn().Str("name", r.host.Name).Str("id", r.host.ID).Msg("An agent with an existing ID was found. Removing the duplicate host. For more details, see https://dozzle.dev/guide/faq#i-am-seeing-duplicate-hosts-error-in-the-logs-how-do-i-fix-it")
			continue
		}
		m.clients[r.host.ID] = r.service
		m.agents[r.endpoint] = connectedAgent{id: r.host.ID, closer: r.closer}
		host := r.host
		host.Available = true
		published = append(published, host)
	}
	clients := lo.Values(m.clients)
	m.mu.Unlock()

	for _, host := range published {
		m.publish(host)
	}
	return clients, errs
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
	host, service, closer, err := m.connect(ctx, endpoint, pair)
	if err != nil {
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
		m.publish(container.Host{ID: endpoint, Endpoint: agentAddr(endpoint), Type: "agent", Removed: true})
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
	m.wasAvailable.Delete(a.id)

	closeQuietly(a.closer)
	m.publish(container.Host{ID: a.id, Endpoint: agentAddr(endpoint), Type: "agent", Removed: true})
	log.Info().Str("endpoint", endpoint).Str("id", a.id).Msg("agent removed")
	return nil
}

// agentAddr is the address part of an "address|name|group" endpoint, the same
// Endpoint Hosts reports for it.
func agentAddr(endpoint string) string {
	if addr, _, _, err := agent.ParseEndpoint(endpoint); err == nil {
		return addr
	}
	return endpoint
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
	// The availability history follows the host to its new id, so the next
	// outage still counts and the old id does not linger.
	if prev, ok := m.wasAvailable.LoadAndDelete(oldID); ok {
		m.wasAvailable.Store(host.ID, prev)
	}

	log.Info().Str("name", host.Name).Str("id", host.ID).Str("previousId", oldID).Msg("host came back with a new id, updating routing")

	host.ReplacesID = oldID
	m.publish(host)

	return host
}

// publish tells every subscriber about a host, without blocking this caller on
// a slow one, and in the order it was called.
func (m *RetriableClientManager) publish(host container.Host) {
	m.subscribers.Range(func(sub *hostSubscriber, _ struct{}) bool {
		sub.enqueue(host)
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
		// Removed while Host() was in flight: reporting it would bring a removed
		// host back, and its availability record with it. A concurrent Hosts may
		// already have re-keyed it, so its new id counts as still served.
		if !m.serves(r.entry.id, r.entry.service) && !m.serves(r.host.ID, r.entry.service) {
			continue
		}
		// An agent mints its id when its own process starts, so an agent that
		// restarted since we last looked answers under a different id than the
		// one this map is keyed by. Nothing else notices: RetryAndList only ever
		// revisits endpoints that never connected. Left alone, the id we hand the
		// UI here is one Find has never heard of, and every lookup for that host
		// fails with "host not found" until the hub itself is restarted.
		key := r.entry.id
		if r.host.Available && r.host.ID != "" && r.host.ID != r.entry.id {
			r.host = m.rekey(r.entry.id, r.entry.service, r.host)
			if _, ok := m.Find(r.host.ID); ok {
				key = r.host.ID
			}
		}
		if r.host.Type == "agent" {
			// LoadAndStore, not Load then Store: Hosts runs concurrently, and two callers that
			// both saw it up would count one outage twice.
			if prev, ok := m.wasAvailable.LoadAndStore(key, r.host.Available); ok && prev && !r.host.Available {
				analytics.Count("agent.disconnect")
			}
			// RemoveAgent may have dropped this record between the check above
			// and the store, which would leave it behind for good.
			if !m.serves(key, r.entry.service) {
				m.wasAvailable.Delete(key)
			}
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

// serves reports whether id still routes to service.
func (m *RetriableClientManager) serves(id string, service container.ClientService) bool {
	current, ok := m.Find(id)
	return ok && current == service
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

package hostservice

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"maps"
	"net"
	"os"
	"slices"
	"sync"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/container/docker"

	"github.com/puzpuzpuz/xsync/v4"

	"github.com/rs/zerolog/log"
)

type SwarmClientManager struct {
	clients map[string]container.ClientService
	// peers maps the host id of every other node's agent to the endpoint it was
	// found at, so one whose task left tasks.<service> can be dropped.
	peers        map[string]swarmPeer
	lookup       func(host string) ([]net.IP, error)
	dial         agentDialer
	certs        tls.Certificate
	mu           sync.RWMutex
	subscribers  *xsync.Map[*hostSubscriber, struct{}]
	localClient  container.Client
	localIPs     []string
	name         string
	timeout      time.Duration
	agentManager *RetriableClientManager
}

type swarmPeer struct {
	endpoint string
	closer   io.Closer
}

func localIPs() []string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return []string{}
	}

	ips := make([]string, 0)
	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ips = append(ips, ipnet.IP.String())
			}
		}
	}
	return ips
}

func NewSwarmClientManager(localClient *docker.Client, certs tls.Certificate, timeout time.Duration, agentManager *RetriableClientManager, labels container.ContainerLabels) *SwarmClientManager {
	clientMap := make(map[string]container.ClientService)
	localService := docker.NewService(localClient, labels)
	clientMap[localClient.Host().ID] = localService

	id, ok := os.LookupEnv("HOSTNAME")
	if !ok {
		log.Fatal().Msg("HOSTNAME environment variable not set when looking for swarm service name")
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	c, err := localClient.FindContainer(ctx, id)
	if err != nil {
		log.Fatal().Err(err).Msg("error finding own container when looking for swarm service name")
	}

	serviceName := c.Labels["com.docker.swarm.service.name"]

	log.Debug().Str("service", serviceName).Msg("found swarm service name")

	return &SwarmClientManager{
		localClient:  localClient,
		clients:      clientMap,
		peers:        make(map[string]swarmPeer),
		lookup:       net.LookupIP,
		dial:         dialAgent,
		certs:        certs,
		subscribers:  xsync.NewMap[*hostSubscriber, struct{}](),
		localIPs:     localIPs(),
		name:         serviceName,
		timeout:      timeout,
		agentManager: agentManager,
	}
}

func (m *SwarmClientManager) Subscribe(ctx context.Context, channel chan<- container.Host) {
	sub := &hostSubscriber{ctx: ctx, channel: channel}
	m.subscribers.Store(sub, struct{}{})
	m.agentManager.Subscribe(ctx, channel)

	go func() {
		<-ctx.Done()
		m.subscribers.Delete(sub)
	}()
}

func (m *SwarmClientManager) RetryAndList() ([]container.ClientService, []error) {
	ips, err := m.lookup(fmt.Sprintf("tasks.%s", m.name))

	errors := make([]error, 0)
	if err != nil {
		log.Fatal().Err(err).Msg("error looking up swarm service tasks")
		errors = append(errors, err)
		return m.List(), errors
	}

	live := make(map[string]bool, len(ips))
	for _, ip := range ips {
		live[ip.String()+":7007"] = true
	}

	log.Debug().Strs(fmt.Sprintf("tasks.%s", m.name), slices.Collect(maps.Keys(live))).Strs("localIPs", m.localIPs).Msg("found swarm service tasks")

	m.mu.Lock()
	// A node that left the swarm takes its task out of tasks.<service>, and nothing
	// else ever tells us. Kept, it is shown as unavailable forever, and every
	// Hosts() waits the full timeout on it. No removal is published: the events
	// stream keeps a host's visible set across one, so a node that left and came
	// back would show up in an open tab with none of its containers.
	known := make(map[string]bool, len(m.peers))
	for id, peer := range m.peers {
		if live[peer.endpoint] {
			known[peer.endpoint] = true
			continue
		}
		delete(m.peers, id)
		delete(m.clients, id)
		closeQuietly(peer.closer)
		log.Info().Str("endpoint", peer.endpoint).Str("id", id).Msg("removed swarm agent that left the service")
	}
	m.mu.Unlock()

	var endpoints []string
	for _, ip := range ips {
		if slices.Contains(m.localIPs, ip.String()) {
			log.Debug().Stringer("ip", ip).Msg("skipping local IP")
			continue
		}
		if known[ip.String()+":7007"] {
			log.Debug().Stringer("ip", ip).Msg("skipping existing client")
			continue
		}
		endpoints = append(endpoints, ip.String()+":7007")
	}

	// Dialed in parallel and without m.mu: this runs on every container listing, and
	// a task that does not answer would otherwise hold List, Find and Hosts for the
	// timeout, once per such task.
	type dialed struct {
		host    container.Host
		service container.ClientService
		closer  io.Closer
		err     error
	}
	results := make([]dialed, len(endpoints))
	var wg sync.WaitGroup
	for i, endpoint := range endpoints {
		wg.Go(func() {
			service, closer, err := m.dial(endpoint, m.certs)
			if err != nil {
				log.Warn().Err(err).Str("endpoint", endpoint).Msg("error creating agent client")
				results[i] = dialed{err: err}
				return
			}
			ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
			host, err := service.Host(ctx)
			cancel()
			if err != nil {
				log.Warn().Err(err).Str("endpoint", endpoint).Msg("error getting host from agent client")
				closeQuietly(closer)
				results[i] = dialed{err: err}
				return
			}
			results[i] = dialed{host: host, service: service, closer: closer}
		})
	}
	wg.Wait()

	var added []container.Host
	m.mu.Lock()
	for i, r := range results {
		if r.err != nil {
			errors = append(errors, r.err)
			continue
		}

		if r.host.ID == m.localClient.Host().ID {
			log.Debug().Str("endpoint", endpoints[i]).Msg("skipping local client")
			closeQuietly(r.closer)
			continue
		}

		// A second live task for a node already served (a start-first update, two
		// replicas on one node, or a concurrent call that got there first).
		// Replacing the serving client would leak it and republish the node on
		// every call, so the duplicate goes instead.
		if _, ok := m.peers[r.host.ID]; ok {
			log.Debug().Str("endpoint", endpoints[i]).Str("id", r.host.ID).Msg("skipping second task for a node already served")
			closeQuietly(r.closer)
			continue
		}

		m.clients[r.host.ID] = r.service
		m.peers[r.host.ID] = swarmPeer{endpoint: endpoints[i], closer: r.closer}
		log.Info().Str("endpoint", endpoints[i]).Str("id", r.host.ID).Str("name", r.host.Name).Msg("added new swarm agent")
		added = append(added, r.host)
	}
	m.mu.Unlock()

	for _, host := range added {
		host.Available = true
		host.Type = "swarm"
		m.subscribers.Range(func(sub *hostSubscriber, _ struct{}) bool {
			// We don't want to block the subscribers in event.go
			go func() {
				select {
				case sub.channel <- host:
				case <-sub.ctx.Done():
				}
			}()
			return true
		})
	}

	return m.List(), errors
}

func (m *SwarmClientManager) List() []container.ClientService {
	m.mu.RLock()
	defer m.mu.RUnlock()

	agents := m.agentManager.List()
	clients := slices.AppendSeq(make([]container.ClientService, 0, len(m.clients)), maps.Values(m.clients))

	return append(agents, clients...)
}

func (m *SwarmClientManager) Find(id string) (container.ClientService, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	client, ok := m.clients[id]

	if !ok {
		client, ok = m.agentManager.Find(id)
	}

	return client, ok
}

func (m *SwarmClientManager) Hosts(ctx context.Context) []container.Host {
	m.mu.RLock()
	clients := slices.AppendSeq(make([]container.ClientService, 0, len(m.clients)), maps.Values(m.clients))
	m.mu.RUnlock()

	swarmNodes := make([]container.Host, len(clients))
	var agents []container.Host
	var wg sync.WaitGroup
	// Alongside the nodes, not after them: they share ctx, and one node that
	// never answers would otherwise leave the agents an expired deadline.
	wg.Go(func() {
		agents = m.agentManager.Hosts(ctx)
	})
	for i, client := range clients {
		wg.Go(func() {
			host, err := client.Host(ctx)
			if err != nil {
				log.Warn().Err(err).Str("id", host.ID).Msg("error getting host from client")
				host.Available = false
			} else {
				host.Available = true
			}
			host.Type = "swarm"

			swarmNodes[i] = host
		})
	}
	wg.Wait()

	return append(agents, swarmNodes...)
}

func (m *SwarmClientManager) String() string {
	return fmt.Sprintf("SwarmClientManager{clients: %d}", len(m.clients))
}

func (m *SwarmClientManager) LocalClients() []container.Client {
	return []container.Client{m.localClient}
}

func (m *SwarmClientManager) LocalClientServices() []container.ClientService {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]container.ClientService, 0)
	for _, service := range m.clients {
		if _, ok := service.(*docker.Service); ok {
			result = append(result, service)
		}
	}

	return result
}

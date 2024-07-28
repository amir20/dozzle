package docker_support

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"sync"

	"github.com/amir20/dozzle/internal/agent"
	"github.com/amir20/dozzle/internal/docker"
	"github.com/puzpuzpuz/xsync/v3"
	"github.com/samber/lo"
	lop "github.com/samber/lo/parallel"

	log "github.com/sirupsen/logrus"
)

type SwarmClientManager struct {
	clients     map[string]ClientService
	certs       tls.Certificate
	mu          sync.RWMutex
	subscribers *xsync.MapOf[context.Context, chan<- docker.Host]
	localClient docker.Client
	localIPs    []string
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

func NewSwarmClientManager(localClient docker.Client, certs tls.Certificate) *SwarmClientManager {
	clientMap := make(map[string]ClientService)
	localService := NewDockerClientService(localClient)
	clientMap[localClient.Host().ID] = localService

	return &SwarmClientManager{
		localClient: localClient,
		clients:     clientMap,
		certs:       certs,
		subscribers: xsync.NewMapOf[context.Context, chan<- docker.Host](),
		localIPs:    localIPs(),
	}
}

func (m *SwarmClientManager) Subscribe(ctx context.Context, channel chan<- docker.Host) {
	m.subscribers.Store(ctx, channel)

	go func() {
		<-ctx.Done()
		m.subscribers.Delete(ctx)
	}()
}

func (m *SwarmClientManager) RetryAndList() ([]ClientService, []error) {
	m.mu.Lock()
	errors := make([]error, 0)

	ips, err := net.LookupIP("tasks.dozzle")

	if err != nil {
		log.Fatalf("error looking up swarm services: %v", err)
		errors = append(errors, err)
		return m.List(), errors
	}

	clients := lo.Values(m.clients)
	endpoints := lo.KeyBy(clients, func(client ClientService) string {
		host, _ := client.Host()
		return host.Endpoint
	})

	log.Debugf("tasks.dozzle = %v, localIP = %v, clients.endpoints = %v", ips, m.localIPs, lo.Keys(endpoints))

	for _, ip := range ips {
		if lo.Contains(m.localIPs, ip.String()) {
			log.Debugf("skipping local ip %s", ip.String())
			continue
		}

		if _, ok := endpoints[ip.String()+":7007"]; ok {
			log.Debugf("skipping existing client for %s", ip.String())
			continue
		}

		agent, err := agent.NewClient(ip.String()+":7007", m.certs)
		if err != nil {
			log.Warnf("error creating client for %s: %v", ip, err)
			errors = append(errors, err)
			continue
		}

		host, err := agent.Host()
		if err != nil {
			log.Warnf("error getting host data for agent %s: %v", ip, err)
			errors = append(errors, err)
			if err := agent.Close(); err != nil {
				log.Warnf("error closing local client: %v", err)
			}
			continue
		}

		if host.ID == m.localClient.Host().ID {
			log.Debugf("skipping local client with ID %s", host.ID)
			if err := agent.Close(); err != nil {
				log.Warnf("error closing local client: %v", err)
			}
			continue
		}

		client := NewAgentService(agent)
		m.clients[host.ID] = client
		log.Infof("added client for %s", host.ID)

		m.subscribers.Range(func(ctx context.Context, channel chan<- docker.Host) bool {
			host.Available = true
			host.Type = "swarm"

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

	m.mu.Unlock()

	return m.List(), errors
}

func (m *SwarmClientManager) List() []ClientService {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return lo.Values(m.clients)
}

func (m *SwarmClientManager) Find(id string) (ClientService, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	client, ok := m.clients[id]
	return client, ok
}

func (m *SwarmClientManager) Hosts() []docker.Host {
	clients := m.List()

	return lop.Map(clients, func(client ClientService, _ int) docker.Host {
		host, err := client.Host()
		if err != nil {
			host.Available = false
		} else {
			host.Available = true
		}
		host.Type = "swarm"

		return host
	})

}

func (m *SwarmClientManager) String() string {
	return fmt.Sprintf("SwarmClientManager{clients: %d}", len(m.clients))
}

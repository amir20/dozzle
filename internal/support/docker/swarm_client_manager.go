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

	log "github.com/sirupsen/logrus"
)

type SwarmClientManager struct {
	clients     map[string]ClientService
	certs       tls.Certificate
	mu          sync.RWMutex
	subscribers *xsync.MapOf[context.Context, chan<- docker.Host]
	localIP     string
	localClient docker.Client
}

func localIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}

func NewSwarmClientManager(localClient docker.Client, certs tls.Certificate) *SwarmClientManager {
	log.Debugf("creating swarm client manager with local client")

	clientMap := make(map[string]ClientService)
	localService := NewDockerClientService(localClient)
	clientMap[localClient.Host().ID] = localService

	return &SwarmClientManager{
		localClient: localClient,
		clients:     clientMap,
		certs:       certs,
		subscribers: xsync.NewMapOf[context.Context, chan<- docker.Host](),
		localIP:     localIP(),
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
		return client.Host().Endpoint
	})

	for _, ip := range ips {
		if ip.String() == m.localIP {
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

		if agent.Host().ID == m.localClient.Host().ID {
			log.Debugf("skipping local client")
			if err := agent.Close(); err != nil {
				log.Warnf("error closing local client: %v", err)
			}
			continue
		}

		client := NewAgentService(agent)
		m.clients[agent.Host().ID] = client
		log.Infof("added client for %s", agent.Host().ID)
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

func (m *SwarmClientManager) FailedAgents() []string {
	return nil
}

func (m *SwarmClientManager) String() string {
	return fmt.Sprintf("SwarmClientManager{clients: %d}", len(m.clients))
}

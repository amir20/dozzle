package support_k8s

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/amir20/dozzle/internal/agent"
	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/k8s"
	container_support "github.com/amir20/dozzle/internal/support/container"

	"github.com/puzpuzpuz/xsync/v3"
	"github.com/samber/lo"
	lop "github.com/samber/lo/parallel"

	"github.com/rs/zerolog/log"
)

type K8sClusterManager struct {
	clients     map[string]container_support.ClientService
	certs       tls.Certificate
	mu          sync.RWMutex
	subscribers *xsync.MapOf[context.Context, chan<- container.Host]
	localClient container.Client
	localIPs    []string
	timeout     time.Duration
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

func NewK8sClusterManager(localClient *k8s.K8sClient, certs tls.Certificate, timeout time.Duration, filter container.ContainerFilter) *K8sClusterManager {
	clientMap := make(map[string]container_support.ClientService)
	localService := NewK8sClientService(localClient, filter)
	clientMap[localClient.Host().ID] = localService

	return &K8sClusterManager{
		localClient: localClient,
		clients:     clientMap,
		certs:       certs,
		subscribers: xsync.NewMapOf[context.Context, chan<- container.Host](),
		localIPs:    localIPs(),
		timeout:     timeout,
	}
}

func (m *K8sClusterManager) Subscribe(ctx context.Context, channel chan<- container.Host) {
	m.subscribers.Store(ctx, channel)

	go func() {
		<-ctx.Done()
		m.subscribers.Delete(ctx)
	}()
}

func (m *K8sClusterManager) RetryAndList() ([]container_support.ClientService, []error) {
	m.mu.Lock()

	ips, err := net.LookupIP(fmt.Sprintf("%s.default.svc.cluster.local", "dozzle-headless"))

	errors := make([]error, 0)
	if err != nil {
		log.Fatal().Err(err).Msg("error looking up swarm service tasks")
		errors = append(errors, err)
		m.mu.Unlock()
		return m.List(), errors
	}

	clients := lo.Values(m.clients)
	endpoints := lo.KeyBy(clients, func(client container_support.ClientService) string {
		ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
		defer cancel()
		host, _ := client.Host(ctx)
		return host.Endpoint
	})

	ipStrings := lo.Map(ips, func(ip net.IP, _ int) string {
		return ip.String()
	})

	log.Debug().Strs(fmt.Sprintf("%s.default.svc.cluster.local", "dozzle-headless"), ipStrings).Strs("localIPs", m.localIPs).Strs("clients.endpoints", lo.Keys(endpoints)).Msg("found swarm service tasks")

	for _, ip := range ips {
		if lo.Contains(m.localIPs, ip.String()) {
			log.Debug().Stringer("ip", ip).Msg("skipping local IP")
			continue
		}

		if _, ok := endpoints[ip.String()+":7007"]; ok {
			log.Debug().Stringer("ip", ip).Msg("skipping existing client")
			continue
		}

		agent, err := agent.NewClient(ip.String()+":7007", m.certs)
		if err != nil {
			log.Warn().Err(err).Stringer("ip", ip).Msg("error creating agent client")
			errors = append(errors, err)
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
		defer cancel()
		host, err := agent.Host(ctx)
		if err != nil {
			log.Warn().Err(err).Stringer("ip", ip).Msg("error getting host from agent client")
			errors = append(errors, err)
			if err := agent.Close(); err != nil {
				log.Warn().Err(err).Stringer("ip", ip).Msg("error closing agent client")
			}
			continue
		}

		if host.ID == m.localClient.Host().ID {
			log.Debug().Stringer("ip", ip).Msg("skipping local client")
			if err := agent.Close(); err != nil {
				log.Warn().Err(err).Stringer("ip", ip).Msg("error closing agent client")
			}
			continue
		}

		client := container_support.NewAgentService(agent)
		m.clients[host.ID] = client
		log.Info().Stringer("ip", ip).Str("id", host.ID).Str("name", host.Name).Msg("added new swarm agent")

		m.subscribers.Range(func(ctx context.Context, channel chan<- container.Host) bool {
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

func (m *K8sClusterManager) List() []container_support.ClientService {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return lo.Values(m.clients)
}

func (m *K8sClusterManager) Find(id string) (container_support.ClientService, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	client, ok := m.clients[id]

	return client, ok
}

func (m *K8sClusterManager) Hosts(ctx context.Context) []container.Host {
	m.mu.RLock()
	clients := lo.Values(m.clients)
	m.mu.RUnlock()

	clusterNodes := lop.Map(clients, func(client container_support.ClientService, _ int) container.Host {
		host, err := client.Host(ctx)
		if err != nil {
			log.Warn().Err(err).Str("id", host.ID).Msg("error getting host from client")
			host.Available = false
		} else {
			host.Available = true
		}
		host.Type = "swarm"

		return host
	})

	return clusterNodes
}

func (m *K8sClusterManager) String() string {
	return fmt.Sprintf("K8sClusterManager{clients: %d}", len(m.clients))
}

func (m *K8sClusterManager) LocalClients() []container.Client {
	return []container.Client{m.localClient}
}

package docker_support

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"sync"

	"github.com/amir20/dozzle/internal/agent"
	"github.com/amir20/dozzle/internal/docker"
	"github.com/puzpuzpuz/xsync/v3"
	"github.com/samber/lo"
	lop "github.com/samber/lo/parallel"

	"github.com/rs/zerolog/log"
)

type SwarmClientManager struct {
	clients     map[string]ClientService
	certs       tls.Certificate
	mu          sync.RWMutex
	subscribers *xsync.MapOf[context.Context, chan<- docker.Host]
	localClient docker.Client
	localIPs    []string
	name        string
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

	id, ok := os.LookupEnv("HOSTNAME")
	if !ok {
		log.Fatal().Msg("HOSTNAME environment variable not set when looking for swarm service name")
	}

	container, err := localClient.FindContainer(id)
	if err != nil {
		log.Fatal().Err(err).Msg("error finding own container when looking for swarm service name")
	}

	serviceName := container.Labels["com.docker.swarm.service.name"]

	log.Debug().Str("service", serviceName).Msg("found swarm service name")

	return &SwarmClientManager{
		localClient: localClient,
		clients:     clientMap,
		certs:       certs,
		subscribers: xsync.NewMapOf[context.Context, chan<- docker.Host](),
		localIPs:    localIPs(),
		name:        serviceName,
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

	ips, err := net.LookupIP(fmt.Sprintf("tasks.%s", m.name))

	errors := make([]error, 0)
	if err != nil {
		log.Fatal().Err(err).Msg("error looking up swarm service tasks")
		errors = append(errors, err)
		m.mu.Unlock()
		return m.List(), errors
	}

	clients := lo.Values(m.clients)
	endpoints := lo.KeyBy(clients, func(client ClientService) string {
		host, _ := client.Host()
		return host.Endpoint
	})
	// TODO - fix this
	//log.Debugf("tasks.dozzle = %v, localIP = %v, clients.endpoints = %v", ips, m.localIPs, lo.Keys(endpoints))
	// log.Debug().Stringers("ips", ips).Msg("found swarm service tasks")

	for _, ip := range ips {
		if lo.Contains(m.localIPs, ip.String()) {
			log.Debug().Str("ip", ip.String()).Msg("skipping local IP")
			continue
		}

		if _, ok := endpoints[ip.String()+":7007"]; ok {
			log.Debug().Stringer("ip", ip).Msg("skipping existing client")
			continue
		}

		agent, err := agent.NewClient(ip.String()+":7007", m.certs)
		if err != nil {
			log.Warn().Err(err).Str("ip", ip.String()).Msg("error creating agent client")
			errors = append(errors, err)
			continue
		}

		host, err := agent.Host()
		if err != nil {
			log.Warn().Err(err).Str("ip", ip.String()).Msg("error getting host from agent client")
			errors = append(errors, err)
			if err := agent.Close(); err != nil {
				log.Warn().Err(err).Str("ip", ip.String()).Msg("error closing agent client")
			}
			continue
		}

		if host.ID == m.localClient.Host().ID {
			log.Warn().Str("ip", ip.String()).Msg("skipping local client")
			if err := agent.Close(); err != nil {
				log.Warn().Err(err).Str("ip", ip.String()).Msg("error closing agent client")
			}
			continue
		}

		client := NewAgentService(agent)
		m.clients[host.ID] = client
		log.Info().Str("ip", ip.String()).Str("id", host.ID).Str("name", host.Name).Msg("added new client")

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

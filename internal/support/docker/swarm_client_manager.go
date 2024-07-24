package docker_support

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"sync"

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
}

func NewSwarmClientManager(localClient docker.Client, certs tls.Certificate) *SwarmClientManager {
	log.Debugf("creating swarm client manager with local client")

	clientMap := make(map[string]ClientService)
	localService := NewDockerClientService(localClient)
	clientMap[localClient.Host().ID] = localService

	return &SwarmClientManager{
		clients:     clientMap,
		certs:       certs,
		subscribers: xsync.NewMapOf[context.Context, chan<- docker.Host](),
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

	lo.GroupBy[T any, U comparable, Slice ~[]T](collection Slice, iteratee func(item T) U)



	m.mu.Unlock()

	return m.List(), errors
}

func (m *SwarmClientManager) List() []ClientService {
	m.mu.RLock()
	defer m.mu.RUnlock()

	clients := make([]ClientService, 0, len(m.clients))
	for _, client := range m.clients {
		clients = append(clients, client)
	}
	return clients
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

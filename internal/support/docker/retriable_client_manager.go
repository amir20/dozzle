package docker_support

import (
	"crypto/tls"
	"fmt"

	"github.com/amir20/dozzle/internal/agent"
	log "github.com/sirupsen/logrus"
)

type RetriableClientManager struct {
	clients      map[string]ClientService
	failedAgents []string
	certs        tls.Certificate
}

func NewRetriableClientManager(clients []ClientService, agents []string, certs tls.Certificate) *RetriableClientManager {
	log.Debugf("creating retriable client manager with %d clients and %d agents", len(clients), len(agents))

	clientMap := make(map[string]ClientService)
	for _, client := range clients {
		if _, ok := clientMap[client.Host().ID]; ok {
			log.Warnf("duplicate client found for host %s", client.Host().ID)
		} else {
			clientMap[client.Host().ID] = client
		}
	}

	failed := make([]string, 0)
	for _, endpoint := range agents {
		if agent, err := agent.NewClient(endpoint, certs); err == nil {
			if _, ok := clientMap[agent.Host().ID]; ok {
				log.Warnf("duplicate client found for host %s", agent.Host().ID)
			} else {
				clientMap[agent.Host().ID] = NewAgentService(agent)
			}
		} else {
			log.Warnf("error creating agent client for %s: %v", endpoint, err)
			failed = append(failed, endpoint)
		}
	}

	return &RetriableClientManager{
		clients:      clientMap,
		failedAgents: failed,
		certs:        certs,
	}
}

func (m *RetriableClientManager) RetryAndList() ([]ClientService, []error) {
	errors := make([]error, 0)
	if len(m.failedAgents) > 0 {
		newFailed := make([]string, 0)
		for _, endpoint := range m.failedAgents {
			if agent, err := agent.NewClient(endpoint, m.certs); err == nil {
				m.clients[agent.Host().ID] = NewAgentService(agent)
			} else {
				log.Warnf("error creating agent client for %s: %v", endpoint, err)
				errors = append(errors, err)
				newFailed = append(newFailed, endpoint)
			}
		}
		m.failedAgents = newFailed
	}

	return m.List(), errors
}

func (m *RetriableClientManager) List() []ClientService {
	clients := make([]ClientService, 0, len(m.clients))
	for _, client := range m.clients {
		clients = append(clients, client)
	}
	return clients
}

func (m *RetriableClientManager) Find(id string) (ClientService, bool) {
	client, ok := m.clients[id]
	return client, ok
}

func (m *RetriableClientManager) String() string {
	return fmt.Sprintf("RetriableClientManager{clients: %d, failedAgents: %d}", len(m.clients), len(m.failedAgents))
}

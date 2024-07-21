package docker_support

import (
	"crypto/tls"

	"github.com/amir20/dozzle/internal/agent"
	log "github.com/sirupsen/logrus"
)

type RetritableClientManager struct {
	clients      map[string]ClientService
	failedAgents []string
	certs        tls.Certificate
}

func NewRetritableClientManager(clients []ClientService, agents []string, certs tls.Certificate) *RetritableClientManager {
	clientMap := make(map[string]ClientService)
	for _, client := range clients {
		clientMap[client.Host().ID] = client
	}

	failed := make([]string, 0)
	for _, endpoint := range agents {
		if agent, err := agent.NewClient(endpoint, certs); err == nil {
			clientMap[agent.Host().ID] = NewAgentService(agent)
		} else {
			log.Warnf("error creating agent client for %s: %v", endpoint, err)
			failed = append(failed, endpoint)
		}
	}

	return &RetritableClientManager{
		clients:      clientMap,
		failedAgents: failed,
		certs:        certs,
	}
}

func (m *RetritableClientManager) RetryAndList() ([]ClientService, []error) {
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

func (m *RetritableClientManager) List() []ClientService {
	clients := make([]ClientService, len(m.clients))

	for _, client := range m.clients {
		clients = append(clients, client)
	}

	return clients
}

func (m *RetritableClientManager) Find(id string) (ClientService, bool) {
	client, ok := m.clients[id]
	return client, ok
}

package docker_support

import (
	"fmt"
)

type MultiHostService interface {
	FindContainer(host string, id string) (ContainerService, error)
}

type multiHostService struct {
	clients map[string]ClientService
}

func NewMultiHostService(clients map[string]ClientService) *multiHostService {
	return &multiHostService{
		clients: clients,
	}
}

func (m *multiHostService) FindContainer(host string, id string) (ContainerService, error) {
	client, ok := m.clients[host]
	if !ok {
		return nil, fmt.Errorf("host %s not found", host)
	}

	container, err := client.FindContainer(id)
	if err != nil {
		return nil, err
	}

	return &containerService{
		clientService: client,
		container:     container,
	}, nil
}

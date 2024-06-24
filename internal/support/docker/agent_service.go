package docker_support

import (
	"context"
	"io"
	"time"

	"github.com/amir20/dozzle/internal/agent"
	"github.com/amir20/dozzle/internal/docker"
)

type agentService struct {
	client *agent.Client
}

func NewAgentService(client *agent.Client) ClientService {
	return &agentService{client: client}
}

func (a *agentService) FindContainer(id string) (docker.Container, error) {
	return a.client.FindContainer(id)
}

func (a *agentService) RawLogReader(ctx context.Context, container docker.Container, from time.Time, to time.Time, stdTypes docker.StdType) (io.ReadCloser, error) {
	return nil, nil
}

func (a *agentService) StreamLogsBetweenDates(ctx context.Context, container docker.Container, from time.Time, to time.Time, stdTypes docker.StdType) (<-chan *docker.LogEvent, error) {
	events := make(chan *docker.LogEvent)
	go a.client.StreamContainerLogs(ctx, container.ID, from, to, stdTypes, events)
	return events, nil
}

func (a *agentService) StreamLogs(ctx context.Context, container docker.Container, from time.Time, stdTypes docker.StdType, events chan<- *docker.LogEvent) error {
	return a.client.StreamContainerLogs(ctx, container.ID, from, time.Now().Add(24*time.Hour), stdTypes, events)
}

func (a *agentService) ListContainers() ([]docker.Container, error) {
	return a.client.ListContainers()
}

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
	return &agentService{
		client: client,
	}
}

func (a *agentService) FindContainer(id string) (docker.Container, error) {
	return a.client.FindContainer(id)
}

func (a *agentService) RawLogs(ctx context.Context, container docker.Container, from time.Time, to time.Time, stdTypes docker.StdType) (io.ReadCloser, error) {
	return a.client.StreamRawBytes(ctx, container.ID, from, to, stdTypes)
}

func (a *agentService) LogsBetweenDates(ctx context.Context, container docker.Container, from time.Time, to time.Time, stdTypes docker.StdType) (<-chan *docker.LogEvent, error) {
	return a.client.LogsBetweenDates(ctx, container.ID, from, to, stdTypes)
}

func (a *agentService) StreamLogs(ctx context.Context, container docker.Container, from time.Time, stdTypes docker.StdType, events chan<- *docker.LogEvent) error {
	return a.client.StreamContainerLogs(ctx, container.ID, from, stdTypes, events)
}

func (a *agentService) ListContainers() ([]docker.Container, error) {
	return a.client.ListContainers()
}

func (a *agentService) Host() docker.Host {
	return a.client.Host()
}

func (a *agentService) SubscribeStats(ctx context.Context, stats chan<- docker.ContainerStat) {
	go a.client.StreamStats(ctx, stats)
}

func (a *agentService) SubscribeEvents(ctx context.Context, events chan<- docker.ContainerEvent) {
	go a.client.StreamEvents(ctx, events)
}

func (d *agentService) SubscribeContainersStarted(ctx context.Context, containers chan<- docker.Container) {
	go d.client.StreamNewContainers(ctx, containers)
}

func (a *agentService) ContainerAction(container docker.Container, action docker.ContainerAction) error {
	panic("not implemented")
}

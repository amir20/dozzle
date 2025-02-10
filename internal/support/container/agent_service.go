package container_support

import (
	"context"
	"io"

	"time"

	"github.com/amir20/dozzle/internal/agent"
	"github.com/amir20/dozzle/internal/container"
	"github.com/rs/zerolog/log"
)

type agentService struct {
	client *agent.Client
	host   container.Host
}

func NewAgentService(client *agent.Client) ClientService {
	return &agentService{
		client: client,
	}
}

func (a *agentService) FindContainer(ctx context.Context, id string, labels container.ContainerLabels) (container.Container, error) {
	return a.client.FindContainer(ctx, id)
}

func (a *agentService) RawLogs(ctx context.Context, container container.Container, from time.Time, to time.Time, stdTypes container.StdType) (io.ReadCloser, error) {
	return a.client.StreamRawBytes(ctx, container.ID, from, to, stdTypes)
}

func (a *agentService) LogsBetweenDates(ctx context.Context, container container.Container, from time.Time, to time.Time, stdTypes container.StdType) (<-chan *container.LogEvent, error) {
	return a.client.LogsBetweenDates(ctx, container.ID, from, to, stdTypes)
}

func (a *agentService) StreamLogs(ctx context.Context, container container.Container, from time.Time, stdTypes container.StdType, events chan<- *container.LogEvent) error {
	return a.client.StreamContainerLogs(ctx, container.ID, from, stdTypes, events)
}

func (a *agentService) ListContainers(ctx context.Context, labels container.ContainerLabels) ([]container.Container, error) {
	log.Debug().Interface("labels", labels).Msg("Listing containers from agent")
	return a.client.ListContainers(ctx, labels)
}

func (a *agentService) Host(ctx context.Context) (container.Host, error) {
	host, err := a.client.Host(ctx)
	if err != nil {
		host := a.host
		host.Available = false
		return host, err
	}

	a.host = host
	return a.host, err
}

func (a *agentService) SubscribeStats(ctx context.Context, stats chan<- container.ContainerStat) {
	go a.client.StreamStats(ctx, stats)
}

func (a *agentService) SubscribeEvents(ctx context.Context, events chan<- container.ContainerEvent) {
	go a.client.StreamEvents(ctx, events)
}

func (d *agentService) SubscribeContainersStarted(ctx context.Context, containers chan<- container.Container) {
	go d.client.StreamNewContainers(ctx, containers)
}

func (a *agentService) ContainerAction(ctx context.Context, container container.Container, action container.ContainerAction) error {
	return a.client.ContainerAction(ctx, container.ID, action)
}

package docker_support

import (
	"context"
	"io"
	"time"

	"github.com/amir20/dozzle/internal/agent"
	"github.com/amir20/dozzle/internal/docker"
	"github.com/puzpuzpuz/xsync/v3"
)

type agentService struct {
	client            *agent.Client
	statsSubscribers  *xsync.MapOf[context.Context, context.CancelFunc]
	eventsSubscribers *xsync.MapOf[context.Context, context.CancelFunc]
}

func NewAgentService(client *agent.Client) ClientService {
	return &agentService{
		client:            client,
		statsSubscribers:  xsync.NewMapOf[context.Context, context.CancelFunc](),
		eventsSubscribers: xsync.NewMapOf[context.Context, context.CancelFunc](),
	}
}

func (a *agentService) FindContainer(id string) (docker.Container, error) {
	return a.client.FindContainer(id)
}

func (a *agentService) RawLogs(ctx context.Context, container docker.Container, from time.Time, to time.Time, stdTypes docker.StdType) (io.ReadCloser, error) {
	return a.client.StreamRawBytes(ctx, container.ID, from, to, stdTypes)
}

func (a *agentService) StreamLogsBetweenDates(ctx context.Context, container docker.Container, from time.Time, to time.Time, stdTypes docker.StdType) (<-chan *docker.LogEvent, error) {
	events := make(chan *docker.LogEvent)
	go a.client.StreamContainerLogs(ctx, container.ID, from, to, stdTypes, events)
	return events, nil
}

func (a *agentService) StreamLogs(ctx context.Context, container docker.Container, from time.Time, stdTypes docker.StdType, events chan<- *docker.LogEvent) error {
	return a.client.StreamContainerLogs(ctx, container.ID, from, time.Now().Add(48*time.Hour), stdTypes, events)
}

func (a *agentService) ListContainers() ([]docker.Container, error) {
	return a.client.ListContainers()
}

func (a *agentService) Host() docker.Host {
	return a.client.Host()
}

func (a *agentService) SubscribeStats(ctx context.Context, stats chan<- docker.ContainerStat) {
	context, cancel := context.WithCancel(ctx)
	go a.client.StreamStats(context, stats)
	a.statsSubscribers.Store(ctx, cancel)
}

func (a *agentService) UnsubscribeStats(ctx context.Context) {
	if cancel, ok := a.statsSubscribers.LoadAndDelete(ctx); ok {
		cancel()
	}
}

func (a *agentService) SubscribeEvents(ctx context.Context, events chan<- docker.ContainerEvent) {
	context, cancel := context.WithCancel(ctx)
	go a.client.StreamEvents(context, events)
	a.eventsSubscribers.Store(ctx, cancel)
}

func (a *agentService) UnsubscribeEvents(ctx context.Context) {
	if cancel, ok := a.eventsSubscribers.LoadAndDelete(ctx); ok {
		cancel()
	}
}

func (d *agentService) StreamContainersStarted(ctx context.Context, containers chan<- docker.Container) {
	d.client.StreamNewContainers(ctx, containers)
}

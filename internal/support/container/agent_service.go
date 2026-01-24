package container_support

import (
	"context"
	"io"
	"sync/atomic"

	"time"

	"github.com/amir20/dozzle/internal/agent"
	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/types"
	"github.com/rs/zerolog/log"
)

type agentService struct {
	client  *agent.Client
	host    atomic.Pointer[container.Host]
	healthy atomic.Bool
}

func NewAgentService(client *agent.Client) ClientService {
	svc := &agentService{client: client}
	svc.healthy.Store(true)
	return svc
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
		if cached := a.host.Load(); cached != nil {
			h := *cached
			h.Available = false
			return h, err
		}
		return container.Host{Available: false}, err
	}

	a.host.Store(&host)
	return host, nil
}

func (a *agentService) SubscribeStats(ctx context.Context, stats chan<- container.ContainerStat) {
	go func() {
		if err := a.client.StreamStats(ctx, stats); err != nil {
			a.healthy.Store(false)
		}
	}()
}

func (a *agentService) SubscribeEvents(ctx context.Context, events chan<- container.ContainerEvent) {
	go func() {
		if err := a.client.StreamEvents(ctx, events); err != nil {
			a.healthy.Store(false)
		}
	}()
}

func (a *agentService) SubscribeContainersStarted(ctx context.Context, containers chan<- container.Container) {
	go func() {
		if err := a.client.StreamNewContainers(ctx, containers); err != nil {
			a.healthy.Store(false)
		}
	}()
}

func (a *agentService) Healthy() bool {
	return a.healthy.Load()
}

func (a *agentService) ContainerAction(ctx context.Context, container container.Container, action container.ContainerAction) error {
	return a.client.ContainerAction(ctx, container.ID, action)
}

func (a *agentService) Attach(ctx context.Context, c container.Container, events container.ExecEventReader, stdout io.Writer) error {
	panic("not implemented")
}

func (a *agentService) Exec(ctx context.Context, c container.Container, cmd []string, events container.ExecEventReader, stdout io.Writer) error {
	return a.client.Exec(ctx, c.ID, cmd, events, stdout)
}

func (a *agentService) UpdateNotificationConfig(ctx context.Context, subscriptions []types.SubscriptionConfig, dispatchers []types.DispatcherConfig) error {
	return a.client.UpdateNotificationConfig(ctx, subscriptions, dispatchers)
}

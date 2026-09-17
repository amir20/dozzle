package agent

import (
	"context"
	"io"
	"sync"
	"sync/atomic"

	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/imagecheck"
	"github.com/amir20/dozzle/types"
	"github.com/rs/zerolog/log"
)

type service struct {
	client *Client
	host   atomic.Pointer[container.Host]
}

func NewService(client *Client) ClientService {
	return &service{
		client: client,
	}
}

func (a *service) FindContainer(ctx context.Context, id string, labels container.ContainerLabels) (container.Container, error) {
	return a.client.FindContainer(ctx, id, labels)
}

func (a *service) RawLogs(ctx context.Context, container container.Container, from time.Time, to time.Time, stdTypes container.StdType) (io.ReadCloser, error) {
	return a.client.StreamRawBytes(ctx, container.ID, from, to, stdTypes)
}

func (a *service) LogsBetweenDates(ctx context.Context, container container.Container, from time.Time, to time.Time, stdTypes container.StdType) (<-chan *container.LogEvent, error) {
	return a.client.LogsBetweenDates(ctx, container.ID, from, to, stdTypes)
}

func (a *service) StreamLogs(ctx context.Context, container container.Container, from time.Time, stdTypes container.StdType, events chan<- *container.LogEvent) error {
	return a.client.StreamContainerLogs(ctx, container.ID, from, stdTypes, events)
}

func (a *service) ListContainers(ctx context.Context, labels container.ContainerLabels) ([]container.Container, error) {
	log.Debug().Interface("labels", labels).Msg("Listing containers from agent")
	return a.client.ListContainers(ctx, labels)
}

func (a *service) Host(ctx context.Context) (container.Host, error) {
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

func (a *service) SubscribeStats(ctx context.Context, stats chan<- container.ContainerStat) {
	go a.client.StreamStats(ctx, stats)
}

func (a *service) SubscribeEvents(ctx context.Context, events chan<- container.ContainerEvent) {
	go a.client.StreamEvents(ctx, events)
}

func (d *service) SubscribeContainersStarted(ctx context.Context, containers chan<- container.Container) {
	go d.client.StreamNewContainers(ctx, containers)
}

func (a *service) ContainerAction(ctx context.Context, container container.Container, action container.ContainerAction) error {
	return a.client.ContainerAction(ctx, container.ID, action)
}

func (a *service) UpdateContainer(ctx context.Context, c container.Container, progressCh chan<- container.UpdateProgress) (bool, error) {
	return a.client.UpdateContainer(ctx, c.ID, progressCh)
}

func (a *service) CheckImageUpdate(ctx context.Context, c container.Container, force bool) (imagecheck.Result, error) {
	return a.client.CheckImageUpdate(ctx, c.ID, force)
}

func (a *service) Attach(ctx context.Context, c container.Container, events container.ExecEventReader, stdout io.Writer) error {
	cancelCtx, cancel := context.WithCancel(ctx)
	session, err := a.client.ContainerAttach(cancelCtx, c.ID)
	if err != nil {
		cancel()
		return err
	}

	var wg sync.WaitGroup

	wg.Go(func() {
		defer session.Writer.Close()
		defer cancel()

	loop:
		for {
			event, err := events.ReadEvent()
			if err != nil {
				if err != io.EOF {
					log.Error().Err(err).Msg("error reading event")
				}
				break
			}

			switch event.Type {
			case "userinput":
				if _, err := session.Writer.Write([]byte(event.Data)); err != nil {
					log.Error().Err(err).Msg("error writing to container")
					break loop
				}
			case "resize":
				if err := session.Resize(event.Width, event.Height); err != nil {
					log.Error().Err(err).Msg("error resizing terminal")
				}
			}
		}
	})

	wg.Go(func() {
		defer cancel()
		if _, err := io.Copy(stdout, session.Reader); err != nil {
			log.Error().Err(err).Msg("error copying stdout")
		}
	})

	wg.Wait()
	return nil
}

func (a *service) Exec(ctx context.Context, c container.Container, cmd []string, events container.ExecEventReader, stdout io.Writer) error {
	return a.client.Exec(ctx, c.ID, cmd, events, stdout)
}

func (a *service) UpdateNotificationConfig(ctx context.Context, subscriptions []types.SubscriptionConfig, dispatchers []types.DispatcherConfig) error {
	return a.client.UpdateNotificationConfig(ctx, subscriptions, dispatchers)
}

func (a *service) UpdateCloudConfig(ctx context.Context, cloudConfig *types.CloudConfig) error {
	return a.client.UpdateCloudConfig(ctx, cloudConfig)
}

func (a *service) GetNotificationStats(ctx context.Context) ([]types.SubscriptionStats, error) {
	return a.client.GetNotificationStats(ctx)
}

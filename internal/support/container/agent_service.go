package container_support

import (
	"context"
	"encoding/json"
	"io"
	"sync"

	"time"

	"github.com/amir20/dozzle/internal/agent"
	"github.com/amir20/dozzle/internal/container"
	"github.com/docker/docker/pkg/stdcopy"
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

func (a *agentService) Attach(ctx context.Context, container container.Container, stdin io.Reader, stdout io.Writer) error {
	panic("not implemented")
}

func (a *agentService) Exec(ctx context.Context, c container.Container, cmd []string, stdin io.Reader, stdout io.Writer) error {
	cancelCtx, cancel := context.WithCancel(ctx)
	session, err := a.client.ContainerExec(cancelCtx, c.ID, cmd)

	if err != nil {
		cancel()
		return err
	}

	var wg sync.WaitGroup

	wg.Go(func() {
		decoder := json.NewDecoder(stdin)
		for {
			var event container.ExecEvent
			if err := decoder.Decode(&event); err != nil {
				if err != io.EOF {
					log.Error().Err(err).Msg("error decoding event from ws using agent")
				}
				break
			}

			switch event.Type {
			case "userinput":
				if _, err := session.Writer.Write([]byte(event.Data)); err != nil {
					log.Error().Err(err).Msg("error writing to container using agent")
					break
				}
			case "resize":
				if err := session.Resize(event.Width, event.Height); err != nil {
					log.Error().Err(err).Msg("error resizing terminal using agent")
				}
			}
		}
		cancel()
		session.Writer.Close()
	})

	wg.Go(func() {
		if _, err := stdcopy.StdCopy(stdout, stdout, session.Reader); err != nil {
			log.Error().Err(err).Msg("error while writing to ws using agent")
		}
		cancel()
	})

	wg.Wait()

	return nil
}

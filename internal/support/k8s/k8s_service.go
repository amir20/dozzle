package k8s_support

import (
	"context"
	"encoding/json"
	"io"
	"sync"

	"github.com/rs/zerolog/log"

	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/k8s"
)

type K8sClientService struct {
	client *k8s.K8sClient
	store  *container.ContainerStore
}

func NewK8sClientService(client *k8s.K8sClient, labels container.ContainerLabels) *K8sClientService {
	statsCollector, err := k8s.NewK8sStatsCollector(client, labels)
	if err != nil {
		log.Fatal().Err(err).Msg("Could not create k8s stats collector")
	}
	return &K8sClientService{
		client: client,
		store:  container.NewContainerStore(context.Background(), client, statsCollector, labels),
	}
}

func (k *K8sClientService) FindContainer(ctx context.Context, id string, labels container.ContainerLabels) (container.Container, error) {
	return k.store.FindContainer(id, labels)
}

func (k *K8sClientService) ListContainers(ctx context.Context, labels container.ContainerLabels) ([]container.Container, error) {
	return k.store.ListContainers(labels)
}

func (k *K8sClientService) Host(ctx context.Context) (container.Host, error) {
	return k.client.Host(), nil
}

func (k *K8sClientService) ContainerAction(ctx context.Context, container container.Container, action container.ContainerAction) error {
	return k.client.ContainerActions(ctx, action, container.ID)
}

func (k *K8sClientService) LogsBetweenDates(ctx context.Context, c container.Container, from time.Time, to time.Time, stdTypes container.StdType) (<-chan *container.LogEvent, error) {
	reader, err := k.client.ContainerLogsBetweenDates(ctx, c.ID, from, to, stdTypes)
	if err != nil {
		return nil, err
	}

	k8sReader := k8s.NewLogReader(reader)
	g := container.NewEventGenerator(ctx, k8sReader, c)
	return g.Events, nil
}

func (k *K8sClientService) RawLogs(ctx context.Context, container container.Container, from time.Time, to time.Time, stdTypes container.StdType) (io.ReadCloser, error) {
	return k.client.ContainerLogsBetweenDates(ctx, container.ID, from, to, stdTypes)
}

func (k *K8sClientService) StreamLogs(ctx context.Context, c container.Container, from time.Time, stdTypes container.StdType, events chan<- *container.LogEvent) error {
	reader, err := k.client.ContainerLogs(ctx, c.ID, from, stdTypes)
	if err != nil {
		return err
	}

	k8sReader := k8s.NewLogReader(reader)
	g := container.NewEventGenerator(ctx, k8sReader, c)
	for event := range g.Events {
		events <- event
	}

	select {
	case e := <-g.Errors:
		return e
	default:
		return nil
	}
}

func (k *K8sClientService) SubscribeStats(ctx context.Context, stats chan<- container.ContainerStat) {
	k.store.SubscribeStats(ctx, stats)
}

func (k *K8sClientService) SubscribeEvents(ctx context.Context, events chan<- container.ContainerEvent) {
	k.store.SubscribeEvents(ctx, events)
}

func (k *K8sClientService) SubscribeContainersStarted(ctx context.Context, containers chan<- container.Container) {
	k.store.SubscribeNewContainers(ctx, containers)
}

func (k *K8sClientService) Attach(ctx context.Context, c container.Container, stdin io.Reader, stdout io.Writer) error {
	cancelCtx, cancel := context.WithCancel(ctx)
	session, err := k.client.ContainerAttach(cancelCtx, c.ID)
	if err != nil {
		cancel()
		return err
	}

	var wg sync.WaitGroup

	wg.Go(func() {
		defer session.Writer.Close()
		defer cancel()

		decoder := json.NewDecoder(stdin)
		for {
			var event container.ExecEvent
			if err := decoder.Decode(&event); err != nil {
				if err != io.EOF {
					log.Error().Err(err).Msg("error decoding event")
				}
				break
			}

			switch event.Type {
			case "userinput":
				if _, err := session.Writer.Write([]byte(event.Data)); err != nil {
					log.Error().Err(err).Msg("error writing to container")
					break
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

func (k *K8sClientService) Exec(ctx context.Context, c container.Container, cmd []string, stdin io.Reader, stdout io.Writer) error {
	cancelCtx, cancel := context.WithCancel(ctx)
	session, err := k.client.ContainerExec(cancelCtx, c.ID, cmd)
	if err != nil {
		cancel()
		return err
	}

	var wg sync.WaitGroup

	wg.Go(func() {
		defer session.Writer.Close()
		defer cancel()

		decoder := json.NewDecoder(stdin)
		for {
			var event container.ExecEvent
			if err := decoder.Decode(&event); err != nil {
				if err != io.EOF {
					log.Error().Err(err).Msg("error decoding event")
				}
				break
			}

			switch event.Type {
			case "userinput":
				if _, err := session.Writer.Write([]byte(event.Data)); err != nil {
					log.Error().Err(err).Msg("error writing to container")
					break
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

package k8s

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/rs/zerolog/log"

	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/container/logparse"
	"github.com/amir20/dozzle/internal/imagecheck"
)

type Service struct {
	client *Client
	store  *container.ContainerStore
}

func NewService(client *Client, labels container.ContainerLabels) *Service {
	statsCollector, err := NewStatsCollector(client, labels)
	if err != nil {
		log.Fatal().Err(err).Msg("Could not create k8s stats collector")
	}
	return &Service{
		client: client,
		store:  container.NewContainerStore(context.Background(), client, statsCollector, labels),
	}
}

// Client returns the underlying k8s client.
func (k *Service) Client() *Client {
	return k.client
}

func (k *Service) FindContainer(ctx context.Context, id string, labels container.ContainerLabels) (container.Container, error) {
	return k.store.FindContainer(ctx, id, labels)
}

func (k *Service) ListContainers(ctx context.Context, labels container.ContainerLabels) ([]container.Container, error) {
	return k.store.ListContainers(ctx, labels)
}

func (k *Service) Host(ctx context.Context) (container.Host, error) {
	return k.client.Host(), nil
}

func (k *Service) ContainerAction(ctx context.Context, container container.Container, action container.ContainerAction) error {
	return k.client.ContainerActions(ctx, action, container.ID)
}

func (k *Service) LogsBetweenDates(ctx context.Context, c container.Container, from time.Time, to time.Time, stdTypes container.StdType) (<-chan *container.LogEvent, error) {
	reader, err := k.client.ContainerLogsBetweenDates(ctx, c.ID, from, to, stdTypes)
	if err != nil {
		return nil, err
	}

	k8sReader := NewLogReader(reader)
	g := logparse.NewEventGenerator(ctx, k8sReader, c)
	return g.Events, nil
}

func (k *Service) RawLogs(ctx context.Context, container container.Container, from time.Time, to time.Time, stdTypes container.StdType) (io.ReadCloser, error) {
	return k.client.ContainerLogsBetweenDates(ctx, container.ID, from, to, stdTypes)
}

func (k *Service) StreamLogs(ctx context.Context, c container.Container, from time.Time, stdTypes container.StdType, events chan<- *container.LogEvent) error {
	reader, err := k.client.ContainerLogs(ctx, c.ID, from, stdTypes)
	if err != nil {
		return err
	}

	k8sReader := NewLogReader(reader)
	g := logparse.NewEventGenerator(ctx, k8sReader, c)
	for event := range g.Events {
		select {
		case events <- event:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	select {
	case e := <-g.Errors:
		return e
	default:
		return nil
	}
}

func (k *Service) SubscribeStats(ctx context.Context, stats chan<- container.ContainerStat) {
	k.store.SubscribeStats(ctx, stats)
}

func (k *Service) SubscribeEvents(ctx context.Context, events chan<- container.ContainerEvent) {
	k.store.SubscribeEvents(ctx, events)
}

func (k *Service) SubscribeContainersStarted(ctx context.Context, containers chan<- container.Container) {
	k.store.SubscribeNewContainers(ctx, containers)
}

// CheckImageUpdate is not supported in Kubernetes mode, where image rollout is
// the cluster's responsibility rather than Dozzle's.
func (k *Service) CheckImageUpdate(ctx context.Context, c container.Container, force bool) (imagecheck.Result, error) {
	return imagecheck.Result{
		Image:     c.Image,
		Status:    imagecheck.StatusSkipped,
		Reason:    "image update checks are not supported in Kubernetes mode",
		CheckedAt: time.Now(),
	}, nil
}

func (k *Service) UpdateContainer(ctx context.Context, c container.Container, progressCh chan<- container.UpdateProgress) (bool, error) {
	defer close(progressCh)
	return false, fmt.Errorf("update container is not supported in Kubernetes mode")
}

func (k *Service) Attach(ctx context.Context, c container.Container, events container.ExecEventReader, stdout io.Writer) error {
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

func (k *Service) Exec(ctx context.Context, c container.Container, cmd []string, events container.ExecEventReader, stdout io.Writer) error {
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

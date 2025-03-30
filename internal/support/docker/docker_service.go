package docker_support

import (
	"context"
	"io"
	"sync"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/docker"

	"github.com/docker/docker/pkg/stdcopy"
	"github.com/rs/zerolog/log"
)

type DockerClientService struct {
	client container.Client
	store  *container.ContainerStore
}

func NewDockerClientService(client container.Client, labels container.ContainerLabels) *DockerClientService {
	statsCollector := docker.NewDockerStatsCollector(client, labels)
	return &DockerClientService{
		client: client,
		store:  container.NewContainerStore(context.Background(), client, statsCollector, labels),
	}
}

func (d *DockerClientService) RawLogs(ctx context.Context, container container.Container, from time.Time, to time.Time, stdTypes container.StdType) (io.ReadCloser, error) {
	reader, err := d.client.ContainerLogsBetweenDates(ctx, container.ID, from, to, stdTypes)
	if err != nil {
		return nil, err
	}

	in, out := io.Pipe()

	go func() {
		if container.Tty {
			if _, err := io.Copy(out, reader); err != nil {
				log.Error().Err(err).Msgf("error copying logs for container %s", container.ID)
			}
		} else {
			if _, err := stdcopy.StdCopy(out, out, reader); err != nil {
				log.Error().Err(err).Msgf("error copying logs for container %s", container.ID)
			}
		}

		out.Close()
	}()

	return in, nil

}

func (d *DockerClientService) LogsBetweenDates(ctx context.Context, c container.Container, from time.Time, to time.Time, stdTypes container.StdType) (<-chan *container.LogEvent, error) {
	reader, err := d.client.ContainerLogsBetweenDates(ctx, c.ID, from, to, stdTypes)
	if err != nil {
		return nil, err
	}

	dockerReader := docker.NewLogReader(reader, c.Tty)
	g := container.NewEventGenerator(ctx, dockerReader, c)
	return g.Events, nil
}

func (d *DockerClientService) StreamLogs(ctx context.Context, c container.Container, from time.Time, stdTypes container.StdType, events chan<- *container.LogEvent) error {
	reader, err := d.client.ContainerLogs(ctx, c.ID, from, stdTypes)
	if err != nil {
		return err
	}

	dockerReader := docker.NewLogReader(reader, c.Tty)
	g := container.NewEventGenerator(ctx, dockerReader, c)
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

func (d *DockerClientService) FindContainer(ctx context.Context, id string, labels container.ContainerLabels) (container.Container, error) {
	return d.store.FindContainer(id, labels)
}

func (d *DockerClientService) ContainerAction(ctx context.Context, container container.Container, action container.ContainerAction) error {
	return d.client.ContainerActions(ctx, action, container.ID)
}

func (d *DockerClientService) ListContainers(ctx context.Context, labels container.ContainerLabels) ([]container.Container, error) {
	return d.store.ListContainers(labels)
}

func (d *DockerClientService) Host(ctx context.Context) (container.Host, error) {
	return d.client.Host(), nil
}

func (d *DockerClientService) SubscribeStats(ctx context.Context, stats chan<- container.ContainerStat) {
	d.store.SubscribeStats(ctx, stats)
}

func (d *DockerClientService) SubscribeEvents(ctx context.Context, events chan<- container.ContainerEvent) {
	d.store.SubscribeEvents(ctx, events)
}

func (d *DockerClientService) SubscribeContainersStarted(ctx context.Context, containers chan<- container.Container) {
	d.store.SubscribeNewContainers(ctx, containers)
}

func (d *DockerClientService) Attach(ctx context.Context, container container.Container, stdin io.Reader, stdout io.Writer) error {
	cancelCtx, cancel := context.WithCancel(ctx)
	containerWriter, containerReader, err := d.client.ContainerAttach(cancelCtx, container.ID)
	if err != nil {
		cancel()
		return err
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		if _, err := io.Copy(containerWriter, stdin); err != nil {
			log.Error().Err(err).Msg("error while reading from ws")
		}
		cancel()
		containerWriter.Close()
	}()

	go func() {
		defer wg.Done()
		if container.Tty {
			if _, err := io.Copy(stdout, containerReader); err != nil {
				log.Error().Err(err).Msg("error while writing to ws")
			}
		} else {
			if _, err := stdcopy.StdCopy(stdout, stdout, containerReader); err != nil {
				log.Error().Err(err).Msg("error while writing to ws")
			}
		}
		cancel()
	}()

	wg.Wait()

	return nil
}

func (d *DockerClientService) Exec(ctx context.Context, container container.Container, cmd []string, stdin io.Reader, stdout io.Writer) error {
	cancelCtx, cancel := context.WithCancel(ctx)
	containerWriter, containerReader, err := d.client.ContainerExec(cancelCtx, container.ID, cmd)
	if err != nil {
		cancel()
		return err
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		if _, err := io.Copy(containerWriter, stdin); err != nil {
			log.Error().Err(err).Msg("error while reading from ws")
		}
		cancel()
		containerWriter.Close()
	}()

	go func() {
		defer wg.Done()
		if _, err := stdcopy.StdCopy(stdout, stdout, containerReader); err != nil {
			log.Error().Err(err).Msg("error while writing to ws")
		}
		cancel()
	}()

	wg.Wait()

	return nil
}

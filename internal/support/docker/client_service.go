package docker_support

import (
	"context"
	"io"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/docker"
	"github.com/amir20/dozzle/internal/support"
)

type dockerClientService struct {
	client *docker.DockerClient
	store  *container.ContainerStore
}

func NewDockerClientService(client *docker.DockerClient, filter container.ContainerFilter) support.ClientService {
	return &dockerClientService{
		client: client,
		store:  container.NewContainerStore(context.Background(), client, filter),
	}
}

func (d *dockerClientService) RawLogs(ctx context.Context, container container.Container, from time.Time, to time.Time, stdTypes container.StdType) (io.ReadCloser, error) {
	return d.client.ContainerLogsBetweenDates(ctx, container.ID, from, to, stdTypes)
}

func (d *dockerClientService) LogsBetweenDates(ctx context.Context, c container.Container, from time.Time, to time.Time, stdTypes container.StdType) (<-chan *container.LogEvent, error) {
	reader, err := d.client.ContainerLogsBetweenDates(ctx, c.ID, from, to, stdTypes)
	if err != nil {
		return nil, err
	}

	dockerReader := docker.NewLogReader(reader, c.Tty)
	g := container.NewEventGenerator(ctx, dockerReader, c)
	return g.Events, nil
}

func (d *dockerClientService) StreamLogs(ctx context.Context, c container.Container, from time.Time, stdTypes container.StdType, events chan<- *container.LogEvent) error {
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

func (d *dockerClientService) FindContainer(ctx context.Context, id string, filter container.ContainerFilter) (container.Container, error) {
	return d.store.FindContainer(id, filter)
}

func (d *dockerClientService) ContainerAction(ctx context.Context, container container.Container, action container.ContainerAction) error {
	return d.client.ContainerActions(ctx, action, container.ID)
}

func (d *dockerClientService) ListContainers(ctx context.Context, filter container.ContainerFilter) ([]container.Container, error) {
	return d.store.ListContainers(filter)
}

func (d *dockerClientService) Host(ctx context.Context) (container.Host, error) {
	return d.client.Host(), nil
}

func (d *dockerClientService) SubscribeStats(ctx context.Context, stats chan<- container.ContainerStat) {
	d.store.SubscribeStats(ctx, stats)
}

func (d *dockerClientService) SubscribeEvents(ctx context.Context, events chan<- container.ContainerEvent) {
	d.store.SubscribeEvents(ctx, events)
}

func (d *dockerClientService) SubscribeContainersStarted(ctx context.Context, containers chan<- container.Container) {
	d.store.SubscribeNewContainers(ctx, containers)
}

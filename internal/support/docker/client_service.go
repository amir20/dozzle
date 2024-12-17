package docker_support

import (
	"context"
	"io"
	"time"

	"github.com/amir20/dozzle/internal/docker"
)

type ClientService interface {
	FindContainer(ctx context.Context, id string, filter docker.ContainerFilter) (docker.Container, error)
	ListContainers(ctx context.Context, filter docker.ContainerFilter) ([]docker.Container, error)
	Host(ctx context.Context) (docker.Host, error)
	ContainerAction(ctx context.Context, container docker.Container, action docker.ContainerAction) error
	LogsBetweenDates(ctx context.Context, container docker.Container, from time.Time, to time.Time, stdTypes docker.StdType) (<-chan *docker.LogEvent, error)
	RawLogs(ctx context.Context, container docker.Container, from time.Time, to time.Time, stdTypes docker.StdType) (io.ReadCloser, error)

	// Subscriptions
	SubscribeStats(ctx context.Context, stats chan<- docker.ContainerStat)
	SubscribeEvents(ctx context.Context, events chan<- docker.ContainerEvent)
	SubscribeContainersStarted(ctx context.Context, containers chan<- docker.Container)

	// Blocking streaming functions that should be used in a goroutine
	StreamLogs(ctx context.Context, container docker.Container, from time.Time, stdTypes docker.StdType, events chan<- *docker.LogEvent) error
}

type dockerClientService struct {
	client docker.Client
	store  *docker.ContainerStore
}

func NewDockerClientService(client docker.Client, filter docker.ContainerFilter) ClientService {
	return &dockerClientService{
		client: client,
		store:  docker.NewContainerStore(context.Background(), client, filter),
	}
}

func (d *dockerClientService) RawLogs(ctx context.Context, container docker.Container, from time.Time, to time.Time, stdTypes docker.StdType) (io.ReadCloser, error) {
	return d.client.ContainerLogsBetweenDates(ctx, container.ID, from, to, stdTypes)
}

func (d *dockerClientService) LogsBetweenDates(ctx context.Context, container docker.Container, from time.Time, to time.Time, stdTypes docker.StdType) (<-chan *docker.LogEvent, error) {
	reader, err := d.client.ContainerLogsBetweenDates(ctx, container.ID, from, to, stdTypes)
	if err != nil {
		return nil, err
	}

	g := docker.NewEventGenerator(ctx, reader, container)
	return g.Events, nil
}

func (d *dockerClientService) StreamLogs(ctx context.Context, container docker.Container, from time.Time, stdTypes docker.StdType, events chan<- *docker.LogEvent) error {
	reader, err := d.client.ContainerLogs(ctx, container.ID, from, stdTypes)
	if err != nil {
		return err
	}

	g := docker.NewEventGenerator(ctx, reader, container)
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

func (d *dockerClientService) FindContainer(ctx context.Context, id string, filter docker.ContainerFilter) (docker.Container, error) {
	return d.store.FindContainer(id, filter)
}

func (d *dockerClientService) ContainerAction(ctx context.Context, container docker.Container, action docker.ContainerAction) error {
	return d.client.ContainerActions(ctx, action, container.ID)
}

func (d *dockerClientService) ListContainers(ctx context.Context, filter docker.ContainerFilter) ([]docker.Container, error) {
	return d.store.ListContainers(filter)
}

func (d *dockerClientService) Host(ctx context.Context) (docker.Host, error) {
	return d.client.Host(), nil
}

func (d *dockerClientService) SubscribeStats(ctx context.Context, stats chan<- docker.ContainerStat) {
	d.store.SubscribeStats(ctx, stats)
}

func (d *dockerClientService) SubscribeEvents(ctx context.Context, events chan<- docker.ContainerEvent) {
	d.store.SubscribeEvents(ctx, events)
}

func (d *dockerClientService) SubscribeContainersStarted(ctx context.Context, containers chan<- docker.Container) {
	d.store.SubscribeNewContainers(ctx, containers)
}

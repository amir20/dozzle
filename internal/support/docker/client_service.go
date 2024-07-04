package docker_support

import (
	"context"
	"io"
	"time"

	"github.com/amir20/dozzle/internal/docker"
)

type ClientService interface {
	FindContainer(id string) (docker.Container, error)
	ListContainers() ([]docker.Container, error)
	Host() docker.Host
	ContainerAction(container docker.Container, action docker.ContainerAction) error
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

func NewDockerClientService(client docker.Client) ClientService {
	return &dockerClientService{
		client: client,
		store:  docker.NewContainerStore(context.Background(), client),
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

	g := docker.NewEventGenerator(reader, container)
	return g.Events, nil
}

func (d *dockerClientService) StreamLogs(ctx context.Context, container docker.Container, from time.Time, stdTypes docker.StdType, events chan<- *docker.LogEvent) error {
	reader, err := d.client.ContainerLogs(ctx, container.ID, from, stdTypes)
	if err != nil {
		return err
	}

	g := docker.NewEventGenerator(reader, container)
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

func (d *dockerClientService) FindContainer(id string) (docker.Container, error) {
	container, err := d.store.FindContainer(id)
	if err != nil {
		if err == docker.ErrContainerNotFound {
			return d.client.FindContainer(id)
		} else {
			return docker.Container{}, err
		}
	}

	return container, nil
}

func (d *dockerClientService) ContainerAction(container docker.Container, action docker.ContainerAction) error {
	return d.client.ContainerActions(action, container.ID)
}

func (d *dockerClientService) ListContainers() ([]docker.Container, error) {
	return d.store.ListContainers()
}

func (d *dockerClientService) Host() docker.Host {
	return d.client.Host()
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

package docker_support

import (
	"context"
	"io"
	"time"

	"github.com/amir20/dozzle/internal/docker"
)

type ClientService interface {
	RawLogReader(ctx context.Context, container docker.Container, from time.Time, to time.Time, stdTypes docker.StdType) (io.ReadCloser, error)
	StreamLogsBetweenDates(ctx context.Context, container docker.Container, from time.Time, to time.Time, stdTypes docker.StdType) (<-chan *docker.LogEvent, error)
	StreamLogs(ctx context.Context, container docker.Container, from time.Time, stdTypes docker.StdType, events chan<- *docker.LogEvent) error
	FindContainer(id string) (docker.Container, error)
}

type dockerClientService struct {
	client docker.Client
}

func NewDockerClientService(client docker.Client) ClientService {
	return &dockerClientService{
		client: client,
	}
}

func (d *dockerClientService) RawLogReader(ctx context.Context, container docker.Container, from time.Time, to time.Time, stdTypes docker.StdType) (io.ReadCloser, error) {
	return d.client.ContainerLogsBetweenDates(ctx, container.ID, from, to, stdTypes)
}

func (d *dockerClientService) StreamLogsBetweenDates(ctx context.Context, container docker.Container, from time.Time, to time.Time, stdTypes docker.StdType) (<-chan *docker.LogEvent, error) {
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
	for {
		select {
		case event := <-g.Events:
			events <- event
		case <-ctx.Done():
			return nil
		case e := <-g.Errors:
			return e
		}
	}
}

func (d *dockerClientService) FindContainer(id string) (docker.Container, error) {
	return d.client.FindContainer(id)
}

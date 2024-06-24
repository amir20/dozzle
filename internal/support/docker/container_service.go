package docker_support

import (
	"context"
	"io"
	"time"

	"github.com/amir20/dozzle/internal/docker"
)

type ContainerService interface {
	RawLogs(ctx context.Context, from time.Time, to time.Time, stdTypes docker.StdType) (io.ReadCloser, error)
	StreamLogsBetweenDates(ctx context.Context, from time.Time, to time.Time, stdTypes docker.StdType) (<-chan *docker.LogEvent, error)
	StreamLogs(ctx context.Context, from time.Time, stdTypes docker.StdType, events chan<- *docker.LogEvent) error
	Container() docker.Container
}

type containerService struct {
	clientService ClientService
	container     docker.Container
}

func (c *containerService) RawLogs(ctx context.Context, from time.Time, to time.Time, stdTypes docker.StdType) (io.ReadCloser, error) {
	return c.clientService.RawLogs(ctx, c.container, from, to, stdTypes)
}

func (c *containerService) StreamLogsBetweenDates(ctx context.Context, from time.Time, to time.Time, stdTypes docker.StdType) (<-chan *docker.LogEvent, error) {
	return c.clientService.StreamLogsBetweenDates(ctx, c.container, from, to, stdTypes)
}

func (c *containerService) StreamLogs(ctx context.Context, from time.Time, stdTypes docker.StdType, events chan<- *docker.LogEvent) error {
	return c.clientService.StreamLogs(ctx, c.container, from, stdTypes, events)
}

func (c *containerService) Container() docker.Container {
	return c.container
}

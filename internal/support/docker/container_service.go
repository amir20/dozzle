package docker_support

import (
	"context"
	"io"
	"time"

	"github.com/amir20/dozzle/internal/docker"
)

type ContainerService interface {
	RawLogReader(ctx context.Context, from time.Time, to time.Time, stdTypes docker.StdType) (io.ReadCloser, error)
	StreamLogsBetweenDates(ctx context.Context, from time.Time, to time.Time, stdTypes docker.StdType) (<-chan *docker.LogEvent, error)
	StreamLogs(ctx context.Context, from time.Time, stdTypes docker.StdType, events chan<- *docker.LogEvent) error
}

type containerService struct {
	clientService ClientService
	container     docker.Container
}

func (c *containerService) RawLogReader(ctx context.Context, from time.Time, to time.Time, stdTypes docker.StdType) (io.ReadCloser, error) {
	return c.clientService.RawLogReader(ctx, c.container, from, to, stdTypes)
}

func (c *containerService) StreamLogsBetweenDates(ctx context.Context, from time.Time, to time.Time, stdTypes docker.StdType) (<-chan *docker.LogEvent, error) {
	return c.clientService.StreamLogsBetweenDates(ctx, c.container, from, to, stdTypes)
}

func (c *containerService) StreamLogs(ctx context.Context, from time.Time, stdTypes docker.StdType, events chan<- *docker.LogEvent) error {
	return c.clientService.StreamLogs(ctx, c.container, from, stdTypes, events)
}

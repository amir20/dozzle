package docker_support

import (
	"context"
	"io"
	"time"

	"github.com/amir20/dozzle/internal/docker"
)

type containerService struct {
	clientService ClientService
	Container     docker.Container
}

func (c *containerService) RawLogs(ctx context.Context, from time.Time, to time.Time, stdTypes docker.StdType) (io.ReadCloser, error) {
	return c.clientService.RawLogs(ctx, c.Container, from, to, stdTypes)
}

func (c *containerService) LogsBetweenDates(ctx context.Context, from time.Time, to time.Time, stdTypes docker.StdType) (<-chan *docker.LogEvent, error) {
	return c.clientService.LogsBetweenDates(ctx, c.Container, from, to, stdTypes)
}

func (c *containerService) StreamLogs(ctx context.Context, from time.Time, stdTypes docker.StdType, events chan<- *docker.LogEvent) error {
	return c.clientService.StreamLogs(ctx, c.Container, from, stdTypes, events)
}

func (c *containerService) Action(action docker.ContainerAction) error {
	return c.clientService.ContainerAction(c.Container, action)
}

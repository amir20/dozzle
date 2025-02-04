package docker_support

import (
	"context"
	"io"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/support"
)

type containerService struct {
	clientService support.ClientService
	Container     container.Container
}

func (c *containerService) RawLogs(ctx context.Context, from time.Time, to time.Time, stdTypes container.StdType) (io.ReadCloser, error) {
	return c.clientService.RawLogs(ctx, c.Container, from, to, stdTypes)
}

func (c *containerService) LogsBetweenDates(ctx context.Context, from time.Time, to time.Time, stdTypes container.StdType) (<-chan *container.LogEvent, error) {
	return c.clientService.LogsBetweenDates(ctx, c.Container, from, to, stdTypes)
}

func (c *containerService) StreamLogs(ctx context.Context, from time.Time, stdTypes container.StdType, events chan<- *container.LogEvent) error {
	return c.clientService.StreamLogs(ctx, c.Container, from, stdTypes, events)
}

func (c *containerService) Action(ctx context.Context, action container.ContainerAction) error {
	return c.clientService.ContainerAction(ctx, c.Container, action)
}

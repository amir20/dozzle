package container_support

import (
	"context"
	"io"
	"time"

	"github.com/amir20/dozzle/internal/container"
)

type ContainerService struct {
	clientService ClientService
	Container     container.Container
}

func NewContainerService(clientService ClientService, container container.Container) *ContainerService {
	return &ContainerService{
		clientService: clientService,
		Container:     container,
	}
}

func (c *ContainerService) RawLogs(ctx context.Context, from time.Time, to time.Time, stdTypes container.StdType) (io.ReadCloser, error) {
	return c.clientService.RawLogs(ctx, c.Container, from, to, stdTypes)
}

func (c *ContainerService) LogsBetweenDates(ctx context.Context, from time.Time, to time.Time, stdTypes container.StdType) (<-chan *container.LogEvent, error) {
	return c.clientService.LogsBetweenDates(ctx, c.Container, from, to, stdTypes)
}

func (c *ContainerService) StreamLogs(ctx context.Context, from time.Time, stdTypes container.StdType, events chan<- *container.LogEvent) error {
	return c.clientService.StreamLogs(ctx, c.Container, from, stdTypes, events)
}

func (c *ContainerService) Action(ctx context.Context, action container.ContainerAction) error {
	return c.clientService.ContainerAction(ctx, c.Container, action)
}

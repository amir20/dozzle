package container

import (
	"context"
	"io"
	"time"

	"github.com/amir20/dozzle/internal/imagecheck"
)

type ContainerService struct {
	clientService ClientService
	Container     Container
}

func NewContainerService(clientService ClientService, container Container) *ContainerService {
	return &ContainerService{
		clientService: clientService,
		Container:     container,
	}
}

func (c *ContainerService) RawLogs(ctx context.Context, from time.Time, to time.Time, stdTypes StdType) (io.ReadCloser, error) {
	return c.clientService.RawLogs(ctx, c.Container, from, to, stdTypes)
}

func (c *ContainerService) LogsBetweenDates(ctx context.Context, from time.Time, to time.Time, stdTypes StdType) (<-chan *LogEvent, error) {
	return c.clientService.LogsBetweenDates(ctx, c.Container, from, to, stdTypes)
}

func (c *ContainerService) StreamLogs(ctx context.Context, from time.Time, stdTypes StdType, events chan<- *LogEvent) error {
	return c.clientService.StreamLogs(ctx, c.Container, from, stdTypes, events)
}

func (c *ContainerService) Action(ctx context.Context, action ContainerAction) error {
	return c.clientService.ContainerAction(ctx, c.Container, action)
}

func (c *ContainerService) Update(ctx context.Context, progressCh chan<- UpdateProgress) (bool, error) {
	return c.clientService.UpdateContainer(ctx, c.Container, progressCh)
}

func (c *ContainerService) CheckImageUpdate(ctx context.Context, force bool) (imagecheck.Result, error) {
	return c.clientService.CheckImageUpdate(ctx, c.Container, force)
}

func (c *ContainerService) Attach(ctx context.Context, events ExecEventReader, stdout io.Writer) error {
	return c.clientService.Attach(ctx, c.Container, events, stdout)
}

func (c *ContainerService) Exec(ctx context.Context, cmd []string, events ExecEventReader, stdout io.Writer) error {
	return c.clientService.Exec(ctx, c.Container, cmd, events, stdout)
}

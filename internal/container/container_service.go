package container

import (
	"context"
	"errors"
	"io"
	"slices"
	"time"

	"github.com/amir20/dozzle/internal/imagecheck"
	"github.com/amir20/dozzle/internal/updatepolicy"
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

func (c *ContainerService) LogHistogram(ctx context.Context, from time.Time, to time.Time, width time.Duration) (LogHistogram, error) {
	return c.clientService.LogHistogram(ctx, c.Container, from, to, width)
}

func (c *ContainerService) StreamLogs(ctx context.Context, from time.Time, stdTypes StdType, events chan<- *LogEvent) error {
	return c.clientService.StreamLogs(ctx, c.Container, from, stdTypes, events)
}

// HealthLog reads the container's recent healthcheck runs live from the
// engine. A host that cannot read them returns errors.ErrUnsupported.
func (c *ContainerService) HealthLog(ctx context.Context) ([]HealthProbe, error) {
	reader, ok := c.clientService.(HealthLogReader)
	if !ok {
		return nil, errors.ErrUnsupported
	}
	return reader.HealthLog(ctx, c.Container)
}

// ErrNotAttached is returned for a network or volume the container does not use.
// Inspect is reached through a container the user can see, so it only ever
// reaches that container's own resources.
var ErrNotAttached = errors.New("the container does not use this network or volume")

// InspectNetwork reads one of the networks the container is attached to. A host
// that cannot inspect networks returns errors.ErrUnsupported.
func (c *ContainerService) InspectNetwork(ctx context.Context, name string) (Network, error) {
	if !slices.ContainsFunc(c.Container.Networks, func(n ContainerNetwork) bool { return n.Name == name }) {
		return Network{}, ErrNotAttached
	}
	inspector, ok := c.clientService.(HostResourceInspector)
	if !ok {
		return Network{}, errors.ErrUnsupported
	}
	return inspector.InspectNetwork(ctx, name)
}

// InspectVolume reads one of the volumes the container mounts. A host that
// cannot inspect volumes returns errors.ErrUnsupported.
func (c *ContainerService) InspectVolume(ctx context.Context, name string) (Volume, error) {
	if name == "" || !slices.ContainsFunc(c.Container.Mounts, func(m Mount) bool { return m.Name == name }) {
		return Volume{}, ErrNotAttached
	}
	inspector, ok := c.clientService.(HostResourceInspector)
	if !ok {
		return Volume{}, errors.ErrUnsupported
	}
	return inspector.InspectVolume(ctx, name)
}

func (c *ContainerService) Action(ctx context.Context, action ContainerAction) error {
	return c.clientService.ContainerAction(ctx, c.Container, action)
}

// Update pulls the container's image and, when it changed, swaps the
// container for one on it. source is one of the UpdateSource* values. An
// update that ran, committed or undone, is added to Updates. A container
// labelled dev.dozzle.update=check is refused, whoever asks.
func (c *ContainerService) Update(ctx context.Context, source string, progressCh chan<- UpdateProgress) (bool, error) {
	if updatepolicy.CheckOnly(c.Container.Labels) {
		close(progressCh)
		return false, ErrCheckOnly
	}
	var updated bool
	var err error
	c.recorded(ctx, source, progressCh, func(ch chan<- UpdateProgress) {
		updated, err = c.clientService.UpdateContainer(ctx, c.Container, ch)
	})
	return updated, err
}

// Rollback swaps the container back to the image it ran before its last
// update, and adds the rollback to Updates once it is done.
func (c *ContainerService) Rollback(ctx context.Context, opts RollbackOptions, progressCh chan<- UpdateProgress) error {
	var err error
	c.recorded(ctx, UpdateSourceRollback, progressCh, func(ch chan<- UpdateProgress) {
		err = c.clientService.RollbackContainer(ctx, c.Container, opts, ch)
	})
	return err
}

// recorded runs fn with a progress channel of its own, forwards everything to
// progressCh, and records the result the last progress carries. fn's client
// closes the channel it is given; progressCh is closed here once fn is done.
func (c *ContainerService) recorded(ctx context.Context, source string, progressCh chan<- UpdateProgress, fn func(chan<- UpdateProgress)) {
	inner := make(chan UpdateProgress)
	var result *UpdateResult
	forwarded := make(chan struct{})
	go func() {
		defer close(forwarded)
		defer close(progressCh)
		for p := range inner {
			if p.Result != nil {
				result = p.Result
			}
			// The caller stops reading when its request ends; the rest is
			// drained so fn never blocks.
			select {
			case progressCh <- p:
			case <-ctx.Done():
			}
		}
	}()
	fn(inner)
	<-forwarded
	if result != nil {
		Updates.Add(newUpdateRecord(c.Container, source, *result, time.Now()))
	}
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

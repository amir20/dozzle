package container

import (
	"context"
	"io"
	"time"

	"github.com/amir20/dozzle/internal/imagecheck"
)

type ContainerFilter = func(*Container) bool

type ClientService interface {
	FindContainer(ctx context.Context, id string, labels ContainerLabels) (Container, error)
	ListContainers(ctx context.Context, filter ContainerLabels) ([]Container, error)
	Host(ctx context.Context) (Host, error)
	ContainerAction(ctx context.Context, container Container, action ContainerAction) error
	UpdateContainer(ctx context.Context, container Container, progressCh chan<- UpdateProgress) (bool, error)
	CheckImageUpdate(ctx context.Context, container Container, force bool) (imagecheck.Result, error)
	LogsBetweenDates(ctx context.Context, container Container, from time.Time, to time.Time, stdTypes StdType) (<-chan *LogEvent, error)
	RawLogs(context.Context, Container, time.Time, time.Time, StdType) (io.ReadCloser, error)

	// Subscriptions
	SubscribeStats(context.Context, chan<- ContainerStat)
	SubscribeEvents(context.Context, chan<- ContainerEvent)
	SubscribeContainersStarted(context.Context, chan<- Container)

	// Blocking streaming functions that should be used in a goroutine
	StreamLogs(context.Context, Container, time.Time, StdType, chan<- *LogEvent) error

	// Terminal
	Attach(context.Context, Container, ExecEventReader, io.Writer) error
	Exec(context.Context, Container, []string, ExecEventReader, io.Writer) error
}

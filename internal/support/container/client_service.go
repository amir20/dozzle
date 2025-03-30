package container_support

import (
	"context"
	"io"
	"time"

	"github.com/amir20/dozzle/internal/container"
)

type ContainerFilter = func(*container.Container) bool

type ClientService interface {
	FindContainer(ctx context.Context, id string, labels container.ContainerLabels) (container.Container, error)
	ListContainers(ctx context.Context, filter container.ContainerLabels) ([]container.Container, error)
	Host(ctx context.Context) (container.Host, error)
	ContainerAction(ctx context.Context, container container.Container, action container.ContainerAction) error
	LogsBetweenDates(ctx context.Context, container container.Container, from time.Time, to time.Time, stdTypes container.StdType) (<-chan *container.LogEvent, error)
	RawLogs(context.Context, container.Container, time.Time, time.Time, container.StdType) (io.ReadCloser, error)

	// Subscriptions
	SubscribeStats(context.Context, chan<- container.ContainerStat)
	SubscribeEvents(context.Context, chan<- container.ContainerEvent)
	SubscribeContainersStarted(context.Context, chan<- container.Container)

	// Blocking streaming functions that should be used in a goroutine
	StreamLogs(context.Context, container.Container, time.Time, container.StdType, chan<- *container.LogEvent) error

	// Terminal
	Attach(context.Context, container.Container, io.Reader, io.Writer) error
	Exec(context.Context, container.Container, []string, io.Reader, io.Writer) error
}

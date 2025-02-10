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
	RawLogs(ctx context.Context, container container.Container, from time.Time, to time.Time, stdTypes container.StdType) (io.ReadCloser, error)

	// Subscriptions
	SubscribeStats(ctx context.Context, stats chan<- container.ContainerStat)
	SubscribeEvents(ctx context.Context, events chan<- container.ContainerEvent)
	SubscribeContainersStarted(ctx context.Context, containers chan<- container.Container)

	// Blocking streaming functions that should be used in a goroutine
	StreamLogs(ctx context.Context, container container.Container, from time.Time, stdTypes container.StdType, events chan<- *container.LogEvent) error
}

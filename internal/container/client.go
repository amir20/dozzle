package container

import (
	"context"
	"io"
	"time"

	"github.com/docker/docker/api/types"
)

type StdType int

const (
	UNKNOWN StdType = 1 << iota
	STDOUT
	STDERR
)
const STDALL = STDOUT | STDERR

func (s StdType) String() string {
	switch s {
	case STDOUT:
		return "stdout"
	case STDERR:
		return "stderr"
	case STDALL:
		return "all"
	default:
		return "unknown"
	}
}

type Client interface {
	ListContainers(context.Context, ContainerFilter) ([]Container, error)
	FindContainer(context.Context, string) (Container, error)
	ContainerLogs(context.Context, string, time.Time, StdType) (io.ReadCloser, error)
	ContainerEvents(context.Context, chan<- ContainerEvent) error
	ContainerLogsBetweenDates(context.Context, string, time.Time, time.Time, StdType) (io.ReadCloser, error)
	ContainerStats(context.Context, string, chan<- ContainerStat) error
	Ping(context.Context) (types.Ping, error)
	Host() Host
	ContainerActions(ctx context.Context, action ContainerAction, containerID string) error
	IsSwarmMode() bool
}

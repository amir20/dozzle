package container

import (
	"context"
	"io"
	"time"
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
	ListContainers(context.Context, ContainerLabels) ([]Container, error)
	FindContainer(context.Context, string) (Container, error)
	ContainerLogs(context.Context, string, time.Time, StdType) (io.ReadCloser, error)
	ContainerEvents(context.Context, chan<- ContainerEvent) error
	ContainerLogsBetweenDates(context.Context, string, time.Time, time.Time, StdType) (io.ReadCloser, error)
	ContainerStats(context.Context, string, chan<- ContainerStat) error
	Ping(context.Context) error
	Host() Host
	ContainerActions(ctx context.Context, action ContainerAction, containerID string) error
	ContainerAttach(ctx context.Context, id string) (io.WriteCloser, io.Reader, error)
	ContainerExec(ctx context.Context, id string, cmd []string) (io.WriteCloser, io.Reader, error)
}

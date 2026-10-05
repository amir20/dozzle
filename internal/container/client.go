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

type ExecSession struct {
	Writer io.WriteCloser
	Reader io.Reader
	Resize func(width uint, height uint) error
}

type ExecEvent struct {
	Type   string `json:"type"`
	Data   string `json:"data,omitempty"`
	Width  uint   `json:"width,omitempty"`
	Height uint   `json:"height,omitempty"`
}

// ExecEventReader provides structured exec events (userinput, resize)
type ExecEventReader interface {
	ReadEvent() (*ExecEvent, error)
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
	ContainerAttach(ctx context.Context, id string) (*ExecSession, error)
	ContainerExec(ctx context.Context, id string, cmd []string) (*ExecSession, error)
}

// SizeReader is implemented by clients that can report how much a container's
// writable layer holds. Docker works it out by walking the layer on every call
// and keeps nothing, so the store asks rarely (see size_monitor.go). k8s has no
// equivalent, and its client does not implement it.
type SizeReader interface {
	// ContainerSizes measures every container in one call.
	ContainerSizes(ctx context.Context) (map[string]int64, error)
	// ContainerSize measures one container.
	ContainerSize(ctx context.Context, id string) (int64, error)
	// DiskUsage measures every volume on the host, and reads what images and build
	// cache nothing uses. There is no way to measure one volume, so this is always
	// a walk of all of them.
	DiskUsage(ctx context.Context) (DiskUsage, error)
}

// DiskUsage is one DiskUsage call's result.
type DiskUsage struct {
	// Volumes are the volumes each container mounts, keyed by container ID.
	Volumes map[string][]VolumeUsage
	// Reclaimable leaves the container fields zero: stopped containers' layers are
	// already in the store, and asking Docker again would walk them all a second time.
	Reclaimable Reclaimable
}

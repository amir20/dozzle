package container

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/amir20/dozzle/internal/agent/pb"
	"github.com/amir20/dozzle/internal/utils"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Container represents an internal representation of docker containers
type Container struct {
	ID          string                           `json:"id"`
	Name        string                           `json:"name"`
	Image       string                           `json:"image"`
	Command     string                           `json:"command"`
	Created     time.Time                        `json:"created"`
	StartedAt   time.Time                        `json:"startedAt"`
	FinishedAt  time.Time                        `json:"finishedAt"`
	State       string                           `json:"state"`
	Health      string                           `json:"health,omitempty"`
	Host        string                           `json:"host,omitempty"`
	Tty         bool                             `json:"-"`
	Labels      map[string]string                `json:"labels,omitempty"`
	Stats       *utils.RingBuffer[ContainerStat] `json:"stats,omitempty"`
	MemoryLimit uint64                           `json:"memoryLimit"`
	CPULimit    float64                          `json:"cpuLimit"`
	Group       string                           `json:"group,omitempty"`
	FullyLoaded bool                             `json:"-,omitempty"`
}

func (container Container) ToProto() pb.Container {
	var pbStats []*pb.ContainerStat
	for _, stat := range container.Stats.Data() {
		pbStats = append(pbStats, &pb.ContainerStat{
			Id:            stat.ID,
			CpuPercent:    stat.CPUPercent,
			MemoryPercent: stat.MemoryPercent,
			MemoryUsage:   stat.MemoryUsage,
		})
	}

	return pb.Container{
		Id:          container.ID,
		Name:        container.Name,
		Image:       container.Image,
		Created:     timestamppb.New(container.Created),
		State:       container.State,
		Health:      container.Health,
		Host:        container.Host,
		Tty:         container.Tty,
		Labels:      container.Labels,
		Group:       container.Group,
		Started:     timestamppb.New(container.StartedAt),
		Finished:    timestamppb.New(container.FinishedAt),
		Stats:       pbStats,
		Command:     container.Command,
		MemoryLimit: container.MemoryLimit,
		CpuLimit:    container.CPULimit,
		FullyLoaded: container.FullyLoaded,
	}
}

func FromProto(c *pb.Container) Container {
	var stats []ContainerStat
	for _, stat := range c.Stats {
		stats = append(stats, ContainerStat{
			ID:            stat.Id,
			CPUPercent:    stat.CpuPercent,
			MemoryPercent: stat.MemoryPercent,
			MemoryUsage:   stat.MemoryUsage,
		})
	}

	return Container{
		ID:          c.Id,
		Name:        c.Name,
		Image:       c.Image,
		Labels:      c.Labels,
		Group:       c.Group,
		Created:     c.Created.AsTime(),
		State:       c.State,
		Health:      c.Health,
		Host:        c.Host,
		Tty:         c.Tty,
		Command:     c.Command,
		StartedAt:   c.Started.AsTime(),
		FinishedAt:  c.Finished.AsTime(),
		Stats:       utils.RingBufferFrom(300, stats),
		MemoryLimit: c.MemoryLimit,
		CPULimit:    c.CpuLimit,
		FullyLoaded: c.FullyLoaded,
	}
}

// ContainerStat represent stats instant for a container
type ContainerStat struct {
	ID            string  `json:"id"`
	CPUPercent    float64 `json:"cpu"`
	MemoryPercent float64 `json:"memory"`
	MemoryUsage   float64 `json:"memoryUsage"`
}

// ContainerEvent represents events that are triggered
type ContainerEvent struct {
	Name            string            `json:"name"`
	Host            string            `json:"host"`
	ActorID         string            `json:"actorId"`
	ActorAttributes map[string]string `json:"actorAttributes,omitempty"`
	Time            time.Time         `json:"time"`
	Container       *Container        `json:"-"`
}

type ContainerLabels map[string][]string

func ParseContainerFilter(commaValues string) (ContainerLabels, error) {
	filter := make(ContainerLabels)
	if commaValues == "" {
		return filter, nil
	}

	for val := range strings.SplitSeq(commaValues, ",") {
		pos := strings.Index(val, "=")
		if pos == -1 {
			return nil, fmt.Errorf("invalid filter: %s", filter)
		}
		key := val[:pos]
		val := val[pos+1:]
		filter[key] = append(filter[key], val)
	}

	return filter, nil
}

func (f ContainerLabels) Exists() bool {
	return len(f) > 0
}

type LogPosition string

const (
	Beginning LogPosition = "start"
	Middle    LogPosition = "middle"
	End       LogPosition = "end"
)

type ContainerAction string

const (
	Start   ContainerAction = "start"
	Stop    ContainerAction = "stop"
	Restart ContainerAction = "restart"
)

func ParseContainerAction(input string) (ContainerAction, error) {
	action := ContainerAction(input)
	switch action {
	case Start, Stop, Restart:
		return action, nil
	default:
		return "", fmt.Errorf("unknown action: %s", input)
	}
}

type LogEvent struct {
	Message     any         `json:"m,omitempty"`
	RawMessage  string      `json:"rm,omitempty"`
	Timestamp   int64       `json:"ts"`
	Id          uint32      `json:"id,omitempty"`
	Level       string      `json:"l,omitempty"`
	Position    LogPosition `json:"p,omitempty"`
	Stream      string      `json:"s,omitempty"`
	ContainerID string      `json:"c,omitempty"`
}

func (l *LogEvent) HasLevel() bool {
	return l.Level != "unknown"
}

func (l *LogEvent) IsCloseToTime(other *LogEvent) bool {
	return math.Abs(float64(l.Timestamp-other.Timestamp)) < 10
}

func (l *LogEvent) MessageId() int64 {
	return l.Timestamp
}

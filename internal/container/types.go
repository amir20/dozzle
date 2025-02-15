package container

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/amir20/dozzle/internal/utils"
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
	Group       string                           `json:"group,omitempty"`
	FullyLoaded bool                             `json:"-,omitempty"`
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

	for _, val := range strings.Split(commaValues, ",") {
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

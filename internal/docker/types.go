package docker

import (
	"math"
)

// Container represents an internal representation of docker containers
type Container struct {
	ID      string   `json:"id"`
	Names   []string `json:"names"`
	Name    string   `json:"name"`
	Image   string   `json:"image"`
	ImageID string   `json:"imageId"`
	Command string   `json:"command"`
	Created int64    `json:"created"`
	State   string   `json:"state"`
	Status  string   `json:"status"`
	Health  string   `json:"health,omitempty"`
	Host    string   `json:"host,omitempty"`
	Tty     bool     `json:"-"`
}

// ContainerStat represent stats instant for a container
type ContainerStat struct {
	ID            string `json:"id"`
	CPUPercent    int64  `json:"cpu"`
	MemoryPercent int64  `json:"memory"`
	MemoryUsage   int64  `json:"memoryUsage"`
}

// ContainerEvent represents events that are triggered
type ContainerEvent struct {
	ActorID string `json:"actorId"`
	Name    string `json:"name"`
	Host    string `json:"host"`
}

type LogPosition string

const (
	START  LogPosition = "start"
	MIDDLE LogPosition = "middle"
	END    LogPosition = "end"
)

type LogEvent struct {
	Message   any         `json:"m,omitempty"`
	Timestamp int64       `json:"ts"`
	Id        uint32      `json:"id,omitempty"`
	Level     string      `json:"l,omitempty"`
	Position  LogPosition `json:"p,omitempty"`
	Stream    string      `json:"s,omitempty"`
}

func (l *LogEvent) HasLevel() bool {
	return l.Level != ""
}

func (l *LogEvent) IsCloseToTime(other *LogEvent) bool {
	return math.Abs(float64(l.Timestamp-other.Timestamp)) < 10
}

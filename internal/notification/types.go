package notification

import (
	"strings"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
)

// Notification represents a notification event that can be filtered and sent
type Notification struct {
	ID        string    `json:"id"`
	Container Container `json:"container"`
	Log       Log       `json:"log"`
	Timestamp time.Time `json:"timestamp"`
}

// Container represents a simplified container structure optimized for expr filtering
type Container struct {
	ID     string            `json:"id"`
	Name   string            `json:"name"`
	Image  string            `json:"image"`
	State  string            `json:"state"`
	Health string            `json:"health"`
	Host   string            `json:"host"`
	Labels map[string]string `json:"labels"`
}

// FromContainerModel converts internal container.Container to notification.Container
func FromContainerModel(c container.Container) Container {
	return Container{
		ID:     c.ID,
		Name:   c.Name,
		Image:  c.Image,
		State:  c.State,
		Health: c.Health,
		Host:   c.Host,
		Labels: c.Labels,
	}
}

// Log represents a log entry with message that can be string or object
type Log struct {
	ID        uint32 `json:"id"`
	Message   any    `json:"message"` // string for simple/grouped logs, map for complex logs
	Timestamp int64  `json:"timestamp"`
	Level     string `json:"level"`
	Stream    string `json:"stream"`
	Type      string `json:"type"`
}

// FromLogEvent converts container.LogEvent to notification.Log
func FromLogEvent(l container.LogEvent) Log {
	message := extractMessage(l)

	return Log{
		ID:        l.Id,
		Message:   message,
		Timestamp: l.Timestamp,
		Level:     l.Level,
		Stream:    l.Stream,
		Type:      string(l.Type),
	}
}

// extractMessage extracts and joins message from LogEvent
// For grouped logs (fragments), joins them into a single string
// For complex logs (JSON/objects), keeps the original map
// For simple logs, returns the string as-is
func extractMessage(l container.LogEvent) any {
	switch v := l.Message.(type) {
	case string:
		return v
	case []container.LogFragment:
		var parts []string
		for _, fragment := range v {
			parts = append(parts, fragment.Message)
		}
		return strings.Join(parts, "")
	default:
		// For complex objects (maps/JSON), keep the original structure
		return v
	}
}

// Subscription represents a subscription to log streams with filtering
type Subscription struct {
	Name                string      `json:"name" yaml:"name"`
	LogExpression       string      `json:"logExpression" yaml:"logExpression"`
	LogProgram          *vm.Program `json:"-" yaml:"-"` // Compiled log filter expression
	ContainerExpression string      `json:"containerExpression" yaml:"containerExpression"`
	ContainerProgram    *vm.Program `json:"-" yaml:"-"` // Compiled container filter expression
}

// MatchesContainer checks if a container matches this subscription's container filter
func (s *Subscription) MatchesContainer(c Container) bool {
	if s.ContainerProgram == nil {
		return false
	}

	result, err := expr.Run(s.ContainerProgram, c)
	if err != nil {
		return false
	}

	match, ok := result.(bool)
	return ok && match
}

// MatchesLog checks if a log matches this subscription's log filter
func (s *Subscription) MatchesLog(l Log) bool {
	if s.LogProgram == nil {
		return false
	}

	result, err := expr.Run(s.LogProgram, l)
	if err != nil {
		return false
	}

	match, ok := result.(bool)
	return ok && match
}

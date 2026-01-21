package notification

import (
	"strings"
	"sync/atomic"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/types"
	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
	"github.com/puzpuzpuz/xsync/v4"
)

// FromContainerModel converts internal container.Container to types.NotificationContainer
func FromContainerModel(c container.Container) types.NotificationContainer {
	return types.NotificationContainer{
		ID:     c.ID,
		Name:   c.Name,
		Image:  c.Image,
		State:  c.State,
		Health: c.Health,
		Host:   c.Host,
		Labels: c.Labels,
	}
}

// FromLogEvent converts container.LogEvent to types.NotificationLog
func FromLogEvent(l container.LogEvent) types.NotificationLog {
	message := extractMessage(l)

	return types.NotificationLog{
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
	ID                  int    `json:"id" yaml:"id"`
	Name                string `json:"name" yaml:"name"`
	Enabled             bool   `json:"enabled" yaml:"enabled"`
	DispatcherID        int    `json:"dispatcherId" yaml:"dispatcherId"`
	LogExpression       string `json:"logExpression" yaml:"logExpression"`
	ContainerExpression string `json:"containerExpression" yaml:"containerExpression"`

	// Compiled log filter expression
	LogProgram       *vm.Program `json:"-" yaml:"-"` // Compiled log filter expression
	ContainerProgram *vm.Program `json:"-" yaml:"-"` // Compiled container filter expression

	// Runtime stats (not persisted)
	TriggerCount          atomic.Int64                 `json:"-" yaml:"-"`
	LastTriggeredAt       atomic.Pointer[time.Time]    `json:"-" yaml:"-"`
	TriggeredContainerIDs *xsync.Map[string, struct{}] `json:"-" yaml:"-"` // unique container IDs that triggered
}

// TriggeredContainersCount returns the number of unique containers that triggered this subscription
func (s *Subscription) TriggeredContainersCount() int {
	if s.TriggeredContainerIDs == nil {
		return 0
	}
	return s.TriggeredContainerIDs.Size()
}

// AddTriggeredContainer adds a container ID to the triggered set
func (s *Subscription) AddTriggeredContainer(id string) {
	if s.TriggeredContainerIDs == nil {
		s.TriggeredContainerIDs = xsync.NewMap[string, struct{}]()
	}
	s.TriggeredContainerIDs.Store(id, struct{}{})
}

// DispatcherConfig represents a dispatcher configuration
type DispatcherConfig struct {
	ID       int    `json:"id" yaml:"id"`
	Name     string `json:"name" yaml:"name"`
	Type     string `json:"type" yaml:"type"` // "webhook", etc.
	URL      string `json:"url,omitempty" yaml:"url,omitempty"`
	Template string `json:"template,omitempty" yaml:"template,omitempty"` // Go template for custom payload format
}

// Config represents the persisted notification configuration
type Config struct {
	Subscriptions []*Subscription    `json:"subscriptions" yaml:"subscriptions"`
	Dispatchers   []DispatcherConfig `json:"dispatchers" yaml:"dispatchers"`
}

// MatchesContainer checks if a container matches this subscription's container filter
func (s *Subscription) MatchesContainer(c types.NotificationContainer) bool {
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
func (s *Subscription) MatchesLog(l types.NotificationLog) bool {
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

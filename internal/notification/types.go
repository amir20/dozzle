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
	"github.com/rs/zerolog/log"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

// FromContainerModel converts internal container.Container to types.NotificationContainer
func FromContainerModel(c container.Container, host container.Host) types.NotificationContainer {
	return types.NotificationContainer{
		ID:       c.ID,
		Name:     c.Name,
		Image:    c.Image,
		State:    c.State,
		Health:   c.Health,
		HostID:   host.ID,
		HostName: host.Name,
		Labels:   c.Labels,
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
// For complex logs (JSON/objects), converts to a regular map for expr compatibility
// For simple logs, returns the string as-is
func extractMessage(l container.LogEvent) any {
	switch v := l.Message.(type) {
	case string:
		return container.StripANSI(v)
	case []container.LogFragment:
		var parts []string
		for _, fragment := range v {
			parts = append(parts, container.StripANSI(fragment.Message))
		}
		return strings.Join(parts, "")
	case *orderedmap.OrderedMap[string, any]:
		// Convert OrderedMap to regular map for expr compatibility
		result := make(map[string]any)
		for pair := v.Oldest(); pair != nil; pair = pair.Next() {
			result[pair.Key] = pair.Value
		}
		return result
	case *orderedmap.OrderedMap[string, string]:
		// Convert OrderedMap[string, string] to regular map for expr compatibility
		result := make(map[string]any)
		for pair := v.Oldest(); pair != nil; pair = pair.Next() {
			result[pair.Key] = pair.Value
		}
		return result
	default:
		// For other complex objects, keep the original structure
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
	MetricExpression    string `json:"metricExpression,omitempty" yaml:"metricExpression,omitempty"`
	Cooldown            int    `json:"cooldown,omitempty" yaml:"cooldown,omitempty"` // seconds between metric notifications, default 300

	// Compiled filter expressions
	LogProgram       *vm.Program `json:"-" yaml:"-"` // Compiled log filter expression
	ContainerProgram *vm.Program `json:"-" yaml:"-"` // Compiled container filter expression
	MetricProgram    *vm.Program `json:"-" yaml:"-"` // Compiled metric filter expression

	// Runtime stats (not persisted)
	TriggerCount          atomic.Int64                 `json:"-" yaml:"-"`
	LastTriggeredAt       atomic.Pointer[time.Time]    `json:"-" yaml:"-"`
	TriggeredContainerIDs *xsync.Map[string, struct{}] `json:"-" yaml:"-"` // unique container IDs that triggered

	// Per-container cooldown tracking for metric alerts (containerID -> last triggered time)
	MetricCooldowns *xsync.Map[string, time.Time] `json:"-" yaml:"-"`
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
	ID        int        `json:"id" yaml:"id"`
	Name      string     `json:"name" yaml:"name"`
	Type      string     `json:"type" yaml:"type"` // "webhook", "cloud"
	URL       string     `json:"url,omitempty" yaml:"url,omitempty"`
	Template  string     `json:"template,omitempty" yaml:"template,omitempty"` // Go template for custom payload format
	APIKey    string     `json:"apiKey,omitempty" yaml:"apiKey,omitempty"`     // API key for cloud dispatcher
	Prefix    string     `json:"prefix,omitempty" yaml:"prefix,omitempty"`     // API key prefix for cloud dispatcher
	ExpiresAt *time.Time `json:"expiresAt,omitempty" yaml:"expiresAt,omitempty"`
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
		log.Warn().Err(err).Str("expression", s.ContainerExpression).Msg("container expression evaluation error")
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
		// Type mismatches are expected when expression doesn't match log type
		// e.g., "message contains X" on JSON logs or "message.field" on string logs
		log.Debug().Err(err).Str("expression", s.LogExpression).Msg("log expression evaluation error")
		return false
	}

	match, ok := result.(bool)
	return ok && match
}

// IsMetricAlert returns true if this subscription is a metric-based alert
func (s *Subscription) IsMetricAlert() bool {
	return s.MetricExpression != "" && s.MetricProgram != nil
}

// MatchesMetric checks if a stat matches this subscription's metric filter
func (s *Subscription) MatchesMetric(stat types.NotificationStat) bool {
	if s.MetricProgram == nil {
		return false
	}

	result, err := expr.Run(s.MetricProgram, stat)
	if err != nil {
		log.Debug().Err(err).Str("expression", s.MetricExpression).Msg("metric expression evaluation error")
		return false
	}

	match, ok := result.(bool)
	return ok && match
}

// GetCooldownSeconds returns the cooldown in seconds, defaulting to 300 (5 min)
func (s *Subscription) GetCooldownSeconds() int {
	if s.Cooldown <= 0 {
		return 300
	}
	return s.Cooldown
}

// IsMetricCooldownActive checks if the cooldown is still active for a given container
func (s *Subscription) IsMetricCooldownActive(containerID string) bool {
	if s.MetricCooldowns == nil {
		return false
	}
	lastTriggered, ok := s.MetricCooldowns.Load(containerID)
	if !ok {
		return false
	}
	return time.Since(lastTriggered) < time.Duration(s.GetCooldownSeconds())*time.Second
}

// SetMetricCooldown records the current time as the last triggered time for a container
func (s *Subscription) SetMetricCooldown(containerID string) {
	if s.MetricCooldowns == nil {
		s.MetricCooldowns = xsync.NewMap[string, time.Time]()
	}
	s.MetricCooldowns.Store(containerID, time.Now())
}

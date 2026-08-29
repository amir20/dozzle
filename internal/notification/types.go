package notification

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/utils"
	"github.com/amir20/dozzle/types"
	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
	"github.com/puzpuzpuz/xsync/v4"
	"github.com/rs/zerolog/log"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

// isDozzleContainer returns true if the container is a Dozzle instance (to avoid feedback loops)
func isDozzleContainer(c container.Container) bool {
	return strings.Contains(c.Image, "amir20/dozzle")
}

// FromContainerMounts converts a container's MountStats map into the slice form
// exposed to metric expressions. Mounts whose free-space could not be measured
// (Available == false — e.g. Windows volumes or permission errors) are skipped
// so that `any(mounts, .usedPercent >= 85)` never fires on unmeasurable mounts.
func FromContainerMounts(c container.Container) []types.NotificationMount {
	if len(c.MountStats) == 0 {
		return nil
	}
	mounts := make([]types.NotificationMount, 0, len(c.MountStats))
	for _, ms := range c.MountStats {
		if !ms.Available || ms.Total == 0 {
			continue
		}
		used := ms.Used
		// Some fs implementations report Used as Total-Free; recompute to be safe.
		if used == 0 && ms.Free <= ms.Total {
			used = ms.Total - ms.Free
		}
		mounts = append(mounts, types.NotificationMount{
			Destination:    ms.Destination,
			TotalBytes:     ms.Total,
			FreeBytes:      ms.Free,
			UsedBytes:      used,
			UsedPercent:    float64(used) / float64(ms.Total) * 100.0,
			AvailableBytes: ms.Free,
		})
	}
	return mounts
}

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
		var sb strings.Builder
		for i, fragment := range v {
			if i > 0 {
				sb.WriteString("\n")
			}
			sb.WriteString(container.StripANSI(fragment.Message))
		}
		return sb.String()
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
	EventExpression     string `json:"eventExpression,omitempty" yaml:"eventExpression,omitempty"`
	Cooldown            int    `json:"cooldown,omitempty" yaml:"cooldown,omitempty"`         // seconds between metric notifications, default 300
	SampleWindow        int    `json:"sampleWindow,omitempty" yaml:"sampleWindow,omitempty"` // seconds of samples to evaluate, default 15

	// Compiled filter expressions
	LogProgram       *vm.Program `json:"-" yaml:"-"` // Compiled log filter expression
	ContainerProgram *vm.Program `json:"-" yaml:"-"` // Compiled container filter expression
	MetricProgram    *vm.Program `json:"-" yaml:"-"` // Compiled metric filter expression
	EventProgram     *vm.Program `json:"-" yaml:"-"` // Compiled event filter expression

	// Runtime stats (not persisted)
	TriggerCount          atomic.Int64                 `json:"-" yaml:"-"`
	LastTriggeredAt       atomic.Pointer[time.Time]    `json:"-" yaml:"-"`
	TriggeredContainerIDs *xsync.Map[string, struct{}] `json:"-" yaml:"-"` // unique container IDs that triggered

	// Per-container cooldown tracking for metric alerts (containerID -> last triggered time)
	MetricCooldowns *xsync.Map[string, time.Time] `json:"-" yaml:"-"`

	// Per-container cooldown tracking for event alerts (containerID -> last triggered time)
	EventCooldowns *xsync.Map[string, time.Time] `json:"-" yaml:"-"`

	// Open log suppression windows, keyed by container ID + log pattern hash.
	// Keyed on the pattern rather than the container so a different error from
	// the same container is never suppressed. See RecordLogMatch.
	LogCooldowns *xsync.Map[string, *logWindow] `json:"-" yaml:"-"`

	// Per-container sample buffers for windowed metric evaluation (containerID -> ring buffer of match results)
	MetricSampleBuffers *xsync.Map[string, *utils.RingBuffer[bool]] `json:"-" yaml:"-"`
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

// CompileExpressions compiles all expression strings into executable programs.
// Returns an error describing which expression failed to compile.
func (s *Subscription) CompileExpressions() error {
	if s.ContainerExpression != "" {
		program, err := expr.Compile(s.ContainerExpression, expr.Env(types.NotificationContainer{}))
		if err != nil {
			return fmt.Errorf("failed to compile container expression: %w", err)
		}
		s.ContainerProgram = program
	}

	if s.LogExpression != "" {
		program, err := expr.Compile(s.LogExpression, expr.Env(types.NotificationLog{}))
		if err != nil {
			return fmt.Errorf("failed to compile log expression: %w", err)
		}
		s.LogProgram = program
	}

	if s.MetricExpression != "" {
		program, err := expr.Compile(s.MetricExpression, expr.Env(types.NotificationStat{}))
		if err != nil {
			return fmt.Errorf("failed to compile metric expression: %w", err)
		}
		s.MetricProgram = program
	}

	if s.EventExpression != "" {
		program, err := expr.Compile(s.EventExpression, expr.Env(types.NotificationEvent{}))
		if err != nil {
			return fmt.Errorf("failed to compile event expression: %w", err)
		}
		s.EventProgram = program
	}

	return nil
}

// DispatcherConfig represents a dispatcher configuration
type DispatcherConfig struct {
	ID       int               `json:"id" yaml:"id"`
	Name     string            `json:"name" yaml:"name"`
	Type     string            `json:"type" yaml:"type"` // "webhook" or "cloud"
	URL      string            `json:"url,omitempty" yaml:"url,omitempty"`
	Template string            `json:"template,omitempty" yaml:"template,omitempty"` // Go template for custom payload format
	Headers  map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`   // Custom HTTP headers
	Prefix   string            `json:"prefix,omitempty" yaml:"-"`                    // Cloud dispatcher API key prefix (not persisted)
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

// IsLogAlert returns true if this subscription is a log-based alert
func (s *Subscription) IsLogAlert() bool {
	return s.LogExpression != "" && s.LogProgram != nil
}

// IsMetricAlert returns true if this subscription is a metric-based alert
func (s *Subscription) IsMetricAlert() bool {
	return s.MetricExpression != "" && s.MetricProgram != nil
}

// IsEventAlert returns true if this subscription is an event-based alert
func (s *Subscription) IsEventAlert() bool {
	return s.EventExpression != "" && s.EventProgram != nil
}

// MatchesEvent checks if a Docker event matches this subscription's event filter
func (s *Subscription) MatchesEvent(event types.NotificationEvent) bool {
	if s.EventProgram == nil {
		return false
	}
	result, err := expr.Run(s.EventProgram, event)
	if err != nil {
		log.Debug().Err(err).Str("expression", s.EventExpression).Msg("event expression evaluation error")
		return false
	}
	match, ok := result.(bool)
	return ok && match
}

// IsEventCooldownActive checks if the cooldown is still active for a given container
func (s *Subscription) IsEventCooldownActive(containerID string) bool {
	if s.Cooldown == 0 {
		return false
	}
	lastTriggered, ok := s.EventCooldowns.Load(containerID)
	if !ok {
		return false
	}
	cooldown := time.Duration(s.Cooldown) * time.Second
	return time.Now().Before(lastTriggered.Add(cooldown))
}

// SetEventCooldown records the current time as the last triggered time for a container
func (s *Subscription) SetEventCooldown(containerID string) {
	s.EventCooldowns.Store(containerID, time.Now())
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

// GetCooldownSeconds returns the cooldown in seconds, clamped to [0, 3600]
func (s *Subscription) GetCooldownSeconds() int {
	if s.Cooldown <= 0 {
		return 0
	}
	if s.Cooldown > 3600 {
		return 3600
	}
	return s.Cooldown
}

// IsMetricCooldownActive checks if the cooldown is still active for a given container
func (s *Subscription) IsMetricCooldownActive(containerID string) bool {
	if s.Cooldown == 0 {
		return false
	}
	lastTriggered, ok := s.MetricCooldowns.Load(containerID)
	if !ok {
		return false
	}
	cooldown := time.Duration(s.GetCooldownSeconds()) * time.Second
	return time.Now().Before(lastTriggered.Add(cooldown))
}

// SetMetricCooldown records the current time as the last triggered time for a container
func (s *Subscription) SetMetricCooldown(containerID string) {
	s.MetricCooldowns.Store(containerID, time.Now())
}

// GetSampleWindowSeconds returns the sample window in seconds, clamped to [1, 300], defaulting to 15
func (s *Subscription) GetSampleWindowSeconds() int {
	if s.SampleWindow <= 0 {
		return 15
	}
	if s.SampleWindow < 1 {
		return 1
	}
	if s.SampleWindow > 300 {
		return 300
	}
	return s.SampleWindow
}

// RecordMetricSample records a metric evaluation result and returns true if the window threshold is met.
// The alert fires when the buffer is full and >=80% of samples matched.
func (s *Subscription) RecordMetricSample(containerID string, matched bool) bool {
	windowSize := s.GetSampleWindowSeconds()

	// For window size of 1, just return the match result directly
	if windowSize <= 1 {
		return matched
	}

	buf, _ := s.MetricSampleBuffers.LoadOrCompute(containerID, func() (*utils.RingBuffer[bool], bool) {
		return utils.NewRingBuffer[bool](windowSize), false
	})

	buf.Push(matched)

	if buf.Len() < windowSize {
		return false
	}

	trueCount := 0
	for _, v := range buf.Data() {
		if v {
			trueCount++
		}
	}

	return float64(trueCount)/float64(buf.Len()) >= 0.8
}

const (
	// maxSuppressedPerWindow forces a rollup out early once this many repeats
	// have piled up, so a genuine firehose is reported promptly instead of
	// waiting out the full cooldown.
	maxSuppressedPerWindow = 10000

	// maxTrackedLogPatterns bounds the suppression map per subscription. Past
	// this we stop tracking and dispatch every match, because dropping a line we
	// cannot account for is worse than sending it.
	maxTrackedLogPatterns = 5000

	// logWindowSweepInterval is how often closed suppression windows are checked
	// for rollups to send. It bounds how late a rollup can be, so keep it well
	// under the shortest cooldown anyone would set.
	logWindowSweepInterval = 5 * time.Second
)

// logWindow is an open suppression window for one (container, log pattern)
// pair. The first line of a pattern always dispatches immediately and opens the
// window; identical lines that follow are counted here rather than dropped, and
// leave as a single rollup notification when the window closes.
type logWindow struct {
	mu        sync.Mutex
	openedAt  time.Time
	count     int64
	container types.NotificationContainer
	log       types.NotificationLog
}

// logRollup is a pending "this repeated N more times" notification, produced
// when a suppression window closes with repeats recorded against it.
type logRollup struct {
	Container types.NotificationContainer
	Log       types.NotificationLog
	Count     int64
}

// logCooldownKey scopes a suppression window to one pattern on one container.
func logCooldownKey(containerID, patternHash string) string {
	return containerID + "|" + patternHash
}

// RecordLogMatch registers a matched log line and reports whether it should be
// dispatched right now.
//
// The first sighting of a pattern always dispatches and opens a cooldown
// window. Identical lines arriving while that window is open are counted for
// the rollup instead, so nothing is lost, only deferred. A line with a
// different pattern has its own window and is never held back by this one.
//
// Returns true (dispatch now) when the subscription has no cooldown configured,
// when the message carries no usable pattern, or when we are already tracking
// more patterns than we are willing to hold.
func (s *Subscription) RecordLogMatch(c types.NotificationContainer, l types.NotificationLog) bool {
	if s.GetCooldownSeconds() == 0 || s.LogCooldowns == nil {
		return true
	}

	patternHash := HashPattern(ExtractPattern(MessageText(l.Message)))
	if patternHash == "" {
		return true
	}

	key := logCooldownKey(c.ID, patternHash)
	if _, open := s.LogCooldowns.Load(key); !open && s.LogCooldowns.Size() >= maxTrackedLogPatterns {
		return true
	}

	dispatch := false
	s.LogCooldowns.Compute(key, func(w *logWindow, loaded bool) (*logWindow, xsync.ComputeOp) {
		if !loaded {
			dispatch = true
			return &logWindow{openedAt: time.Now(), container: c, log: l}, xsync.UpdateOp
		}
		w.mu.Lock()
		w.count++
		// Keep the most recent line so the rollup quotes something current.
		w.log = l
		w.container = c
		w.mu.Unlock()
		return w, xsync.UpdateOp
	})

	return dispatch
}

// CollectLogRollups closes every suppression window that has run out its
// cooldown or hit the suppression cap, and returns the rollups to dispatch.
//
// A window that closed with no repeats is deleted, so the next occurrence of
// that pattern is treated as news again and dispatches immediately. A window
// that closed with repeats is reset rather than deleted, so a pattern that
// keeps firing costs one notification per cooldown for as long as it lasts.
func (s *Subscription) CollectLogRollups() []logRollup {
	if s.LogCooldowns == nil {
		return nil
	}

	cooldown := time.Duration(s.GetCooldownSeconds()) * time.Second
	now := time.Now()

	var rollups []logRollup
	var expired []string

	s.LogCooldowns.Range(func(key string, w *logWindow) bool {
		w.mu.Lock()
		defer w.mu.Unlock()

		if w.count < maxSuppressedPerWindow && now.Sub(w.openedAt) < cooldown {
			return true
		}

		if w.count == 0 {
			expired = append(expired, key)
			return true
		}

		rollups = append(rollups, logRollup{Container: w.container, Log: w.log, Count: w.count})
		w.openedAt = now
		w.count = 0
		return true
	})

	// Delete under Compute rather than outright: a repeat can land between the
	// scan and the delete, and a plain Delete would drop a line we had already
	// told the caller to suppress.
	for _, key := range expired {
		s.LogCooldowns.Compute(key, func(w *logWindow, loaded bool) (*logWindow, xsync.ComputeOp) {
			if !loaded {
				return nil, xsync.CancelOp
			}
			w.mu.Lock()
			defer w.mu.Unlock()
			if w.count > 0 {
				return w, xsync.UpdateOp
			}
			return nil, xsync.DeleteOp
		})
	}

	return rollups
}

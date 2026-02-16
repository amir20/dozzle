package types

import "time"

// Notification represents a notification event that can be filtered and sent
type Notification struct {
	ID           string                `json:"id"`
	Detail       string                `json:"detail"`
	Container    NotificationContainer `json:"container"`
	Log          *NotificationLog      `json:"log"`
	Stat         *NotificationStat     `json:"stat"`
	Subscription SubscriptionConfig    `json:"subscription"`
	Timestamp    time.Time             `json:"timestamp"`
}

// NotificationContainer represents a simplified container structure for notifications
type NotificationContainer struct {
	ID       string            `json:"id" expr:"id"`
	Name     string            `json:"name" expr:"name"`
	Image    string            `json:"image" expr:"image"`
	State    string            `json:"state" expr:"state"`
	Health   string            `json:"health" expr:"health"`
	HostID   string            `json:"hostId" expr:"hostId"`
	HostName string            `json:"hostName" expr:"hostName"`
	Labels   map[string]string `json:"labels" expr:"labels"`
}

// NotificationLog represents a log entry with message that can be string or object
type NotificationLog struct {
	ID        uint32 `json:"id" expr:"id"`
	Message   any    `json:"message" expr:"message"` // string for simple/grouped logs, map for complex logs
	Timestamp int64  `json:"timestamp" expr:"timestamp"`
	Level     string `json:"level" expr:"level"`
	Stream    string `json:"stream" expr:"stream"`
	Type      string `json:"type" expr:"type"`
}

// NotificationStat represents container resource metrics for metric-based alerts
type NotificationStat struct {
	CPUPercent    float64 `json:"cpu" expr:"cpu"`
	MemoryPercent float64 `json:"memory" expr:"memory"`
	MemoryUsage   float64 `json:"memoryUsage" expr:"memoryUsage"`
}

// SubscriptionConfig represents a notification subscription configuration
type SubscriptionConfig struct {
	ID                  int    `json:"id"`
	Name                string `json:"name"`
	Enabled             bool   `json:"-"`
	DispatcherID        int    `json:"-"`
	LogExpression       string `json:"logExpression,omitempty"`
	ContainerExpression string `json:"containerExpression"`
	MetricExpression    string `json:"metricExpression,omitempty"`
	Cooldown            int    `json:"cooldown,omitempty"`
}

// DispatcherConfig represents a notification dispatcher configuration
type DispatcherConfig struct {
	ID        int
	Name      string
	Type      string
	URL       string
	Template  string
	APIKey    string
	Prefix    string
	ExpiresAt *time.Time
}

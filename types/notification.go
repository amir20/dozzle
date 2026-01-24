package types

import "time"

// Notification represents a notification event that can be filtered and sent
type Notification struct {
	ID           string                `json:"id"`
	Container    NotificationContainer `json:"container"`
	Log          NotificationLog       `json:"log"`
	Subscription SubscriptionConfig    `json:"subscription"`
	Timestamp    time.Time             `json:"timestamp"`
}

// NotificationContainer represents a simplified container structure for notifications
type NotificationContainer struct {
	ID     string            `json:"id" expr:"id"`
	Name   string            `json:"name" expr:"name"`
	Image  string            `json:"image" expr:"image"`
	State  string            `json:"state" expr:"state"`
	Health string            `json:"health" expr:"health"`
	Host   string            `json:"host" expr:"host"`
	Labels map[string]string `json:"labels" expr:"labels"`
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

// SubscriptionConfig represents a notification subscription configuration
type SubscriptionConfig struct {
	ID                  int    `json:"id"`
	Name                string `json:"name"`
	Enabled             bool   `json:"-"`
	DispatcherID        int    `json:"-"`
	LogExpression       string `json:"logExpression"`
	ContainerExpression string `json:"containerExpression"`
}

// DispatcherConfig represents a notification dispatcher configuration
type DispatcherConfig struct {
	ID       int
	Name     string
	Type     string
	URL      string
	Template string
}

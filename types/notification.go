package types

import "time"

// Notification represents a notification event that can be filtered and sent
type Notification struct {
	ID        string                `json:"id"`
	Container NotificationContainer `json:"container"`
	Log       NotificationLog       `json:"log"`
	Timestamp time.Time             `json:"timestamp"`
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

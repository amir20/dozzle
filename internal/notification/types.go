package notification

import (
	"strings"
	"time"

	"github.com/amir20/dozzle/internal/container"
)

// DispatcherConfig represents global dispatcher configuration
type DispatcherConfig struct {
	// Type determines the dispatcher type (webhook, dozzle-service, etc.)
	Type string `yaml:"type"` // "webhook" is default

	// URL is the endpoint for the dispatcher service
	URL string `yaml:"url"`

	// APIKey for authenticating with the dispatcher service
	APIKey string `yaml:"api_key,omitempty"`

	// Headers to include in all requests
	Headers map[string]string `yaml:"headers,omitempty"`
}

// Subscription represents a notification rule
type Subscription struct {
	// Name is a human-readable name for this subscription
	Name string `yaml:"name"`

	// ContainerFilter is an expr expression to match containers (REQUIRED)
	// Expression receives a Container object with: Name, Labels, Host, State
	// Example: Name == "nginx" || Labels["app"] == "web"
	ContainerFilter string `yaml:"container_filter"`

	// LogFilter is an expr expression to match log events (REQUIRED)
	// Expression receives a Log object with: Message, Level, Timestamp, Container
	// Example: Level == "error" || Message contains "panic"
	LogFilter string `yaml:"log_filter"`
}

// Container represents container information for expr evaluation (internal type)
type Container struct {
	Name   string            `expr:"Name"`
	Labels map[string]string `expr:"Labels"`
	Host   string            `expr:"Host"`
	State  string            `expr:"State"`
}

// Log represents log event information for expr evaluation
type Log struct {
	Message   any       `expr:"Message"`
	Level     string    `expr:"Level"`
	Timestamp time.Time `expr:"Timestamp"`
	Container Container `expr:"Container"`
}

// WebhookPayload is the structure sent to webhook endpoints
type WebhookPayload struct {
	SubscriptionName string    `json:"subscription_name"`
	Timestamp        time.Time `json:"timestamp"`
	Container        Container `json:"container"`
	Log              LogEvent  `json:"log"`
}

// LogEvent represents the log in the webhook payload
type LogEvent struct {
	Message   any       `json:"message"`
	Level     string    `json:"level"`
	Timestamp time.Time `json:"timestamp"`
}

// NewContainer converts container.Container to notification.Container
func NewContainer(c *container.Container) Container {
	return Container{
		Name:   c.Name,
		Labels: c.Labels,
		Host:   c.Host,
		State:  c.State,
	}
}

// NewLog creates a Log from container.LogEvent and notification.Container
func NewLog(logEvent *container.LogEvent, notifContainer Container) Log {
	timestamp := time.Unix(0, logEvent.Timestamp)

	// Extract message - keep as any type for expr flexibility
	var message any = logEvent.Message

	// For grouped logs, join fragments into single string for easier matching
	if fragments, ok := logEvent.Message.([]container.LogFragment); ok {
		parts := make([]string, len(fragments))
		for i, frag := range fragments {
			parts[i] = frag.Message
		}
		message = strings.Join(parts, "\n")
	}

	return Log{
		Message:   message,
		Level:     logEvent.Level,
		Timestamp: timestamp,
		Container: notifContainer,
	}
}

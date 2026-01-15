package notification

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/notification/dispatcher"
	"github.com/expr-lang/expr"
	"github.com/rs/zerolog/log"
	"go.yaml.in/yaml/v2"
)

// Manager manages notification subscriptions and dispatches notifications
type Manager struct {
	subscriptions map[string]*Subscription
	dispatchers   map[string][]dispatcher.Dispatcher // Multiple dispatchers per subscription
	listener      *ContainerLogListener
	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
}

// NewManager creates a new notification manager
func NewManager(listener *ContainerLogListener) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	m := &Manager{
		subscriptions: make(map[string]*Subscription),
		dispatchers:   make(map[string][]dispatcher.Dispatcher),
		listener:      listener,
		ctx:           ctx,
		cancel:        cancel,
	}

	// Start processing log events from the listener
	go m.processLogEvents()

	return m
}

// Start initializes the manager and starts the log listener
func (m *Manager) Start() error {
	if m.listener != nil {
		return m.listener.Start(m)
	}
	return nil
}

// ShouldListenToContainer implements ContainerMatcher interface
func (m *Manager) ShouldListenToContainer(c container.Container) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	notificationContainer := FromContainerModel(c)

	for _, sub := range m.subscriptions {
		if sub.MatchesContainer(notificationContainer) {
			return true
		}
	}
	return false
}

// AddSubscription adds a new subscription with compiled expressions
func (m *Manager) AddSubscription(sub *Subscription) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Compile container expression if provided
	if sub.ContainerExpression != "" {
		program, err := expr.Compile(sub.ContainerExpression, expr.Env(Container{}))
		if err != nil {
			return fmt.Errorf("failed to compile container expression: %w", err)
		}
		sub.ContainerProgram = program
	}

	// Compile log expression if provided
	if sub.LogExpression != "" {
		program, err := expr.Compile(sub.LogExpression, expr.Env(Log{}))
		if err != nil {
			return fmt.Errorf("failed to compile log expression: %w", err)
		}
		sub.LogProgram = program
	}

	m.subscriptions[sub.Name] = sub
	log.Info().Str("name", sub.Name).Msg("Added subscription")

	// Update listener to start/stop streams based on new subscription
	if m.listener != nil {
		if err := m.listener.UpdateStreams(); err != nil {
			log.Error().Err(err).Msg("Failed to update listener streams")
		}
	}

	return nil
}

// RemoveSubscription removes a subscription by name
func (m *Manager) RemoveSubscription(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.subscriptions, name)
	log.Info().Str("name", name).Msg("Removed subscription")

	// Update listener to stop streams that are no longer needed
	if m.listener != nil {
		if err := m.listener.UpdateStreams(); err != nil {
			log.Error().Err(err).Msg("Failed to update listener streams")
		}
	}
}

// AddDispatcher adds a dispatcher for a subscription
func (m *Manager) AddDispatcher(name string, d dispatcher.Dispatcher) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.dispatchers[name] = append(m.dispatchers[name], d)
	log.Info().Str("name", name).Msg("Added dispatcher")
}

// RemoveDispatcher removes all dispatchers for a subscription
func (m *Manager) RemoveDispatcher(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.dispatchers[name]; exists {
		delete(m.dispatchers, name)
		log.Info().Str("name", name).Msg("Removed dispatchers")
	}
}

// processLogEvents processes log events from the listener channel
func (m *Manager) processLogEvents() {
	for {
		select {
		case <-m.ctx.Done():
			return
		case logEvent := <-m.listener.LogChannel():
			if logEvent == nil {
				return
			}
			m.processLogEvent(logEvent)
		}
	}
}

// processLogEvent processes a single log event and sends notifications for matching subscriptions
func (m *Manager) processLogEvent(logEvent *container.LogEvent) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Get container from log event's ContainerID
	ctx, cancel := context.WithTimeout(m.ctx, 5*time.Second)
	defer cancel()

	c, err := m.listener.clientService.FindContainer(ctx, logEvent.ContainerID, nil)
	if err != nil {
		log.Error().Err(err).Str("containerID", logEvent.ContainerID).Msg("Failed to find container")
		return
	}

	notificationContainer := FromContainerModel(c)
	notificationLog := FromLogEvent(*logEvent)

	for name, sub := range m.subscriptions {
		// Check container filter
		if !sub.MatchesContainer(notificationContainer) {
			continue
		}

		// Check log filter
		if !sub.MatchesLog(notificationLog) {
			continue
		}

		// Create notification
		notification := Notification{
			ID:        fmt.Sprintf("%s-%d", c.ID, time.Now().UnixNano()),
			Container: notificationContainer,
			Log:       notificationLog,
			Timestamp: time.Now(),
		}

		// Send to all dispatchers for this subscription
		if dispatchers, exists := m.dispatchers[name]; exists {
			for _, d := range dispatchers {
				go m.sendNotification(d, notification, name)
			}
		}
	}
}

// sendNotification sends a notification using the dispatcher
func (m *Manager) sendNotification(d dispatcher.Dispatcher, notification Notification, name string) {
	ctx, cancel := context.WithTimeout(m.ctx, 30*time.Second)
	defer cancel()

	if err := d.Send(ctx, notification); err != nil {
		log.Error().Err(err).Str("subscription", name).Msg("Failed to send notification")
	}
}

// WriteConfig writes the current configuration to a writer in YAML format
func (m *Manager) WriteConfig(w io.Writer) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	config := Config{
		Subscriptions: make([]SubscriptionConfig, 0, len(m.subscriptions)),
	}

	for name, sub := range m.subscriptions {
		subConfig := SubscriptionConfig{
			Subscription: *sub,
		}

		// Add dispatchers for this subscription
		if dispatchers, exists := m.dispatchers[name]; exists {
			for _, d := range dispatchers {
				switch v := d.(type) {
				case *dispatcher.WebhookDispatcher:
					subConfig.Dispatchers = append(subConfig.Dispatchers, DispatcherConfig{
						Type: "webhook",
						URL:  v.URL,
					})
				}
			}
		}

		config.Subscriptions = append(config.Subscriptions, subConfig)
	}

	encoder := yaml.NewEncoder(w)
	defer encoder.Close()

	return encoder.Encode(config)
}

// LoadConfig reads configuration from a reader in YAML format and loads it
func (m *Manager) LoadConfig(r io.Reader) error {
	var config Config

	decoder := yaml.NewDecoder(r)
	if err := decoder.Decode(&config); err != nil {
		return fmt.Errorf("failed to decode config: %w", err)
	}

	// Load subscriptions and dispatchers
	for _, subConfig := range config.Subscriptions {
		// Add subscription
		if err := m.AddSubscription(&subConfig.Subscription); err != nil {
			return fmt.Errorf("failed to add subscription %s: %w", subConfig.Name, err)
		}

		// Add dispatchers
		for _, dispatcherConfig := range subConfig.Dispatchers {
			var d dispatcher.Dispatcher
			switch dispatcherConfig.Type {
			case "webhook":
				d = dispatcher.NewWebhookDispatcher(dispatcherConfig.URL)
			default:
				return fmt.Errorf("unknown dispatcher type: %s", dispatcherConfig.Type)
			}
			m.AddDispatcher(subConfig.Name, d)
		}
	}

	// Update listener to start streams for loaded subscriptions
	if m.listener != nil {
		if err := m.listener.UpdateStreams(); err != nil {
			return fmt.Errorf("failed to update listener streams: %w", err)
		}
	}

	return nil
}

// Close stops the manager and all active log streams
func (m *Manager) Close() {
	m.cancel()
	if m.listener != nil {
		m.listener.Close()
	}
}

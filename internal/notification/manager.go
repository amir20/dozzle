package notification

import (
	"context"
	"fmt"
	"io"
	"sync/atomic"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/notification/dispatcher"
	"github.com/expr-lang/expr"
	"github.com/puzpuzpuz/xsync/v4"
	"github.com/rs/zerolog/log"
	"go.yaml.in/yaml/v2"
)

// Manager manages notification subscriptions and dispatches notifications
type Manager struct {
	subscriptions       *xsync.Map[int, *Subscription]
	dispatchers         *xsync.Map[int, dispatcher.Dispatcher]
	subscriptionCounter atomic.Int32
	dispatcherCounter   atomic.Int32
	listener            *ContainerLogListener
	ctx                 context.Context
	cancel              context.CancelFunc
}

// NewManager creates a new notification manager
func NewManager(listener *ContainerLogListener) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	m := &Manager{
		subscriptions: xsync.NewMap[int, *Subscription](),
		dispatchers:   xsync.NewMap[int, dispatcher.Dispatcher](),
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
	notificationContainer := FromContainerModel(c)

	shouldListen := false
	m.subscriptions.Range(func(_ int, sub *Subscription) bool {
		if sub.Enabled && sub.MatchesContainer(notificationContainer) {
			shouldListen = true
			return false
		}
		return true
	})
	return shouldListen
}

// AddSubscription adds a new subscription with compiled expressions
func (m *Manager) AddSubscription(sub *Subscription) error {
	// Auto-increment ID using atomic counter
	sub.ID = int(m.subscriptionCounter.Add(1))
	sub.Enabled = true

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

	m.subscriptions.Store(sub.ID, sub)
	log.Info().Str("name", sub.Name).Int("id", sub.ID).Msg("Added subscription")

	// Update listener to start/stop streams based on new subscription
	if m.listener != nil {
		if err := m.listener.UpdateStreams(); err != nil {
			log.Error().Err(err).Msg("Failed to update listener streams")
		}
	}

	return nil
}

// RemoveSubscription removes a subscription by ID
func (m *Manager) RemoveSubscription(id int) {
	if sub, ok := m.subscriptions.LoadAndDelete(id); ok {
		log.Info().Int("id", id).Str("name", sub.Name).Msg("Removed subscription")

		// Update listener to stop streams that are no longer needed
		if m.listener != nil {
			if err := m.listener.UpdateStreams(); err != nil {
				log.Error().Err(err).Msg("Failed to update listener streams")
			}
		}
	}
}

// UpdateSubscription updates a subscription with the provided fields
func (m *Manager) UpdateSubscription(id int, updates map[string]any) error {
	sub, ok := m.subscriptions.Load(id)
	if !ok {
		return fmt.Errorf("subscription not found")
	}

	for key, value := range updates {
		switch key {
		case "name":
			if name, ok := value.(string); ok {
				sub.Name = name
			}
		case "enabled":
			if enabled, ok := value.(bool); ok {
				sub.Enabled = enabled
			}
		case "containerExpression":
			if exprStr, ok := value.(string); ok {
				program, err := expr.Compile(exprStr, expr.Env(Container{}))
				if err != nil {
					return fmt.Errorf("failed to compile container expression: %w", err)
				}
				sub.ContainerExpression = exprStr
				sub.ContainerProgram = program
			}
		case "logExpression":
			if exprStr, ok := value.(string); ok {
				if exprStr != "" {
					program, err := expr.Compile(exprStr, expr.Env(Log{}))
					if err != nil {
						return fmt.Errorf("failed to compile log expression: %w", err)
					}
					sub.LogExpression = exprStr
					sub.LogProgram = program
				}
			}
		}
	}

	log.Debug().Int("id", id).Interface("updates", updates).Msg("Updated subscription")

	// Update listener streams in case expressions changed
	if m.listener != nil {
		if err := m.listener.UpdateStreams(); err != nil {
			log.Error().Err(err).Msg("Failed to update listener streams")
		}
	}

	return nil
}

// AddDispatcher adds a dispatcher and returns its auto-generated ID
func (m *Manager) AddDispatcher(d dispatcher.Dispatcher) int {
	id := int(m.dispatcherCounter.Add(1))
	m.dispatchers.Store(id, d)
	log.Info().Int("id", id).Msg("Added dispatcher")
	return id
}

// RemoveDispatcher removes a dispatcher by ID
func (m *Manager) RemoveDispatcher(id int) {
	if _, ok := m.dispatchers.LoadAndDelete(id); ok {
		log.Info().Int("id", id).Msg("Removed dispatcher")
	}
}

// Subscriptions returns all subscriptions
func (m *Manager) Subscriptions() []Subscription {
	result := make([]Subscription, 0)
	m.subscriptions.Range(func(_ int, sub *Subscription) bool {
		result = append(result, *sub)
		return true
	})
	return result
}

// Dispatchers returns all dispatchers as DispatcherConfig
func (m *Manager) Dispatchers() []DispatcherConfig {
	result := make([]DispatcherConfig, 0)
	m.dispatchers.Range(func(id int, d dispatcher.Dispatcher) bool {
		switch v := d.(type) {
		case *dispatcher.WebhookDispatcher:
			result = append(result, DispatcherConfig{
				ID:   id,
				Type: "webhook",
				URL:  v.URL,
			})
		}
		return true
	})
	return result
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
	// Get container from log event's ContainerID
	ctx, cancel := context.WithTimeout(m.ctx, 5*time.Second)
	defer cancel()

	c, err := m.listener.FindContainer(ctx, logEvent.ContainerID, nil)
	if err != nil {
		log.Error().Err(err).Str("containerID", logEvent.ContainerID).Msg("Failed to find container")
		return
	}

	notificationContainer := FromContainerModel(c)
	notificationLog := FromLogEvent(*logEvent)

	m.subscriptions.Range(func(_ int, sub *Subscription) bool {
		// Skip disabled subscriptions
		if !sub.Enabled {
			return true
		}

		// Check container filter
		if !sub.MatchesContainer(notificationContainer) {
			return true
		}

		if sub.TriggeredContainerIDs == nil {
			sub.TriggeredContainerIDs = make(map[string]struct{})
		}
		sub.TriggeredContainerIDs[notificationContainer.ID] = struct{}{}

		// Check log filter
		if !sub.MatchesLog(notificationLog) {
			return true
		}

		// Update stats
		sub.TriggerCount++
		sub.LastTriggeredAt = time.Now()

		log.Debug().Str("containerID", notificationContainer.ID).Interface("log", notificationLog.Message).Msg("Matched subscription")

		// Create notification
		notification := Notification{
			ID:        fmt.Sprintf("%s-%d", c.ID, time.Now().UnixNano()),
			Container: notificationContainer,
			Log:       notificationLog,
			Timestamp: time.Now(),
		}

		// Send to all dispatchers
		m.dispatchers.Range(func(id int, d dispatcher.Dispatcher) bool {
			go m.sendNotification(d, notification, id)
			return true
		})
		return true
	})
}

// sendNotification sends a notification using the dispatcher
func (m *Manager) sendNotification(d dispatcher.Dispatcher, notification Notification, id int) {
	ctx, cancel := context.WithTimeout(m.ctx, 30*time.Second)
	defer cancel()

	if err := d.Send(ctx, notification); err != nil {
		log.Error().Err(err).Int("subscription", id).Msg("Failed to send notification")
	}
}

// WriteConfig writes the current configuration to a writer in YAML format
func (m *Manager) WriteConfig(w io.Writer) error {
	config := Config{
		Subscriptions: make([]Subscription, 0),
		Dispatchers:   make([]DispatcherConfig, 0),
	}

	m.subscriptions.Range(func(_ int, sub *Subscription) bool {
		config.Subscriptions = append(config.Subscriptions, *sub)
		return true
	})

	m.dispatchers.Range(func(id int, d dispatcher.Dispatcher) bool {
		switch v := d.(type) {
		case *dispatcher.WebhookDispatcher:
			config.Dispatchers = append(config.Dispatchers, DispatcherConfig{
				ID:   id,
				Type: "webhook",
				URL:  v.URL,
			})
		}
		return true
	})

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

	// Find max IDs to initialize counters
	var maxSubID, maxDispatcherID int
	for _, sub := range config.Subscriptions {
		if sub.ID > maxSubID {
			maxSubID = sub.ID
		}
	}
	for _, d := range config.Dispatchers {
		if d.ID > maxDispatcherID {
			maxDispatcherID = d.ID
		}
	}
	m.subscriptionCounter.Store(int32(maxSubID))
	m.dispatcherCounter.Store(int32(maxDispatcherID))

	// Load subscriptions
	for _, sub := range config.Subscriptions {
		subCopy := sub
		if err := m.loadSubscription(&subCopy); err != nil {
			return fmt.Errorf("failed to add subscription %s: %w", sub.Name, err)
		}
	}

	// Load dispatchers
	for _, dispatcherConfig := range config.Dispatchers {
		var d dispatcher.Dispatcher
		switch dispatcherConfig.Type {
		case "webhook":
			d = dispatcher.NewWebhookDispatcher(dispatcherConfig.URL)
		default:
			return fmt.Errorf("unknown dispatcher type: %s", dispatcherConfig.Type)
		}
		m.dispatchers.Store(dispatcherConfig.ID, d)
		log.Info().Int("id", dispatcherConfig.ID).Msg("Loaded dispatcher")
	}

	// Update listener to start streams for loaded subscriptions
	if m.listener != nil {
		if err := m.listener.UpdateStreams(); err != nil {
			return fmt.Errorf("failed to update listener streams: %w", err)
		}
	}

	return nil
}

// loadSubscription loads a subscription with its existing ID (used when loading from config)
func (m *Manager) loadSubscription(sub *Subscription) error {
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

	m.subscriptions.Store(sub.ID, sub)
	log.Info().Str("name", sub.Name).Int("id", sub.ID).Msg("Loaded subscription")
	return nil
}

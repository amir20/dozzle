package notification

import (
	"context"
	"fmt"
	"io"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/notification/dispatcher"
	"github.com/amir20/dozzle/types"
	"github.com/expr-lang/expr"
	"github.com/puzpuzpuz/xsync/v4"
	"github.com/rs/zerolog/log"
	"go.yaml.in/yaml/v3"
	"golang.org/x/sync/semaphore"
)

// Manager manages notification subscriptions and dispatches notifications
type Manager struct {
	subscriptions       *xsync.Map[int, *Subscription]
	dispatchers         *xsync.Map[int, dispatcher.Dispatcher]
	subscriptionCounter atomic.Int32
	dispatcherCounter   atomic.Int32
	listener            *ContainerLogListener
	statsListener       *ContainerStatsListener
	ctx                 context.Context
	cancel              context.CancelFunc
	sendSem             *semaphore.Weighted
}

// NewManager creates a new notification manager
func NewManager(listener *ContainerLogListener, statsListener *ContainerStatsListener) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	m := &Manager{
		subscriptions: xsync.NewMap[int, *Subscription](),
		dispatchers:   xsync.NewMap[int, dispatcher.Dispatcher](),
		listener:      listener,
		statsListener: statsListener,
		ctx:           ctx,
		cancel:        cancel,
		sendSem:       semaphore.NewWeighted(5),
	}

	// Start processing log events from the listener
	go m.processLogEvents()

	// Start processing stat events from the stats listener
	if statsListener != nil {
		go m.processStatEvents()
	}

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
// Only matches log-based subscriptions (metric-only subscriptions don't need log streaming)
func (m *Manager) ShouldListenToContainer(c container.Container) bool {
	// Pass empty host for matching - host fields aren't used in container expressions
	notificationContainer := FromContainerModel(c, container.Host{})

	shouldListen := false
	m.subscriptions.Range(func(_ int, sub *Subscription) bool {
		if sub.Enabled && sub.LogExpression != "" && sub.MatchesContainer(notificationContainer) {
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
		program, err := expr.Compile(sub.ContainerExpression, expr.Env(types.NotificationContainer{}))
		if err != nil {
			return fmt.Errorf("failed to compile container expression: %w", err)
		}
		sub.ContainerProgram = program
	}

	// Compile log expression if provided
	if sub.LogExpression != "" {
		program, err := expr.Compile(sub.LogExpression, expr.Env(types.NotificationLog{}))
		if err != nil {
			return fmt.Errorf("failed to compile log expression: %w", err)
		}
		sub.LogProgram = program
	}

	// Compile metric expression if provided
	if sub.MetricExpression != "" {
		program, err := expr.Compile(sub.MetricExpression, expr.Env(types.NotificationStat{}))
		if err != nil {
			return fmt.Errorf("failed to compile metric expression: %w", err)
		}
		sub.MetricProgram = program
	}

	sub.MetricCooldowns = xsync.NewMap[string, time.Time]()

	m.subscriptions.Store(sub.ID, sub)
	log.Debug().Str("name", sub.Name).Int("id", sub.ID).Msg("Added subscription")

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
		log.Debug().Int("id", id).Str("name", sub.Name).Msg("Removed subscription")

		// Update listener to stop streams that are no longer needed
		if m.listener != nil {
			if err := m.listener.UpdateStreams(); err != nil {
				log.Error().Err(err).Msg("Failed to update listener streams")
			}
		}
	}
}

// ReplaceSubscription replaces a subscription with new data
func (m *Manager) ReplaceSubscription(sub *Subscription) error {
	// Compile container expression if provided
	if sub.ContainerExpression != "" {
		program, err := expr.Compile(sub.ContainerExpression, expr.Env(types.NotificationContainer{}))
		if err != nil {
			return fmt.Errorf("failed to compile container expression: %w", err)
		}
		sub.ContainerProgram = program
	}

	// Compile log expression if provided
	if sub.LogExpression != "" {
		program, err := expr.Compile(sub.LogExpression, expr.Env(types.NotificationLog{}))
		if err != nil {
			return fmt.Errorf("failed to compile log expression: %w", err)
		}
		sub.LogProgram = program
	}

	// Compile metric expression if provided
	if sub.MetricExpression != "" {
		program, err := expr.Compile(sub.MetricExpression, expr.Env(types.NotificationStat{}))
		if err != nil {
			return fmt.Errorf("failed to compile metric expression: %w", err)
		}
		sub.MetricProgram = program
	}

	sub.MetricCooldowns = xsync.NewMap[string, time.Time]()

	// Preserve enabled state from existing subscription if it exists
	if existing, ok := m.subscriptions.Load(sub.ID); ok {
		sub.Enabled = existing.Enabled
	} else {
		sub.Enabled = true
	}

	m.subscriptions.Store(sub.ID, sub)
	log.Debug().Str("name", sub.Name).Int("id", sub.ID).Msg("Replaced subscription")

	// Update listener to start/stop streams based on new subscription
	if m.listener != nil {
		if err := m.listener.UpdateStreams(); err != nil {
			log.Error().Err(err).Msg("Failed to update listener streams")
		}
	}

	return nil
}

// UpdateSubscription updates a subscription with the provided fields
func (m *Manager) UpdateSubscription(id int, updates map[string]any) error {
	var updateErr error
	_, ok := m.subscriptions.Compute(id, func(sub *Subscription, loaded bool) (*Subscription, xsync.ComputeOp) {
		if !loaded {
			updateErr = fmt.Errorf("subscription not found")
			return nil, xsync.CancelOp
		}

		// Clone the subscription
		updated := &Subscription{
			ID:                  sub.ID,
			Name:                sub.Name,
			Enabled:             sub.Enabled,
			DispatcherID:        sub.DispatcherID,
			ContainerExpression: sub.ContainerExpression,
			ContainerProgram:    sub.ContainerProgram,
			LogExpression:       sub.LogExpression,
			LogProgram:          sub.LogProgram,
			MetricExpression:    sub.MetricExpression,
			MetricProgram:       sub.MetricProgram,
			Cooldown:            sub.Cooldown,
			MetricCooldowns:     sub.MetricCooldowns,
		}

		// Apply updates to the clone
		for key, value := range updates {
			switch key {
			case "name":
				if name, ok := value.(string); ok {
					updated.Name = name
				}
			case "enabled":
				if enabled, ok := value.(bool); ok {
					updated.Enabled = enabled
				}
			case "dispatcherId":
				if dispatcherID, ok := value.(int); ok {
					updated.DispatcherID = dispatcherID
				}
			case "containerExpression":
				if exprStr, ok := value.(string); ok {
					program, err := expr.Compile(exprStr, expr.Env(types.NotificationContainer{}))
					if err != nil {
						updateErr = fmt.Errorf("failed to compile container expression: %w", err)
						return nil, xsync.CancelOp
					}
					updated.ContainerExpression = exprStr
					updated.ContainerProgram = program
				}
			case "logExpression":
				if exprStr, ok := value.(string); ok {
					if exprStr != "" {
						program, err := expr.Compile(exprStr, expr.Env(types.NotificationLog{}))
						if err != nil {
							updateErr = fmt.Errorf("failed to compile log expression: %w", err)
							return nil, xsync.CancelOp
						}
						updated.LogExpression = exprStr
						updated.LogProgram = program
					}
				}
			case "metricExpression":
				if exprStr, ok := value.(string); ok {
					if exprStr != "" {
						program, err := expr.Compile(exprStr, expr.Env(types.NotificationStat{}))
						if err != nil {
							updateErr = fmt.Errorf("failed to compile metric expression: %w", err)
							return nil, xsync.CancelOp
						}
						updated.MetricExpression = exprStr
						updated.MetricProgram = program
					} else {
						updated.MetricExpression = ""
						updated.MetricProgram = nil
					}
				}
			case "cooldown":
				if cd, ok := value.(int); ok {
					updated.Cooldown = cd
				}
			}
		}

		return updated, xsync.UpdateOp
	})

	if updateErr != nil {
		return updateErr
	}

	if !ok {
		return fmt.Errorf("subscription not found")
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
	log.Debug().Int("id", id).Msg("Added dispatcher")
	return id
}

// UpdateDispatcher updates a dispatcher by ID
func (m *Manager) UpdateDispatcher(id int, d dispatcher.Dispatcher) {
	m.dispatchers.Store(id, d)
	log.Debug().Int("id", id).Msg("Updated dispatcher")
}

// RemoveDispatcher removes a dispatcher by ID
func (m *Manager) RemoveDispatcher(id int) {
	if _, ok := m.dispatchers.LoadAndDelete(id); ok {
		log.Debug().Int("id", id).Msg("Removed dispatcher")
	}
}

// Subscriptions returns all subscriptions sorted by ID
func (m *Manager) Subscriptions() []*Subscription {
	result := make([]*Subscription, 0)
	m.subscriptions.Range(func(_ int, sub *Subscription) bool {
		result = append(result, sub)
		return true
	})
	slices.SortFunc(result, func(a, b *Subscription) int {
		return a.ID - b.ID
	})
	return result
}

// Dispatchers returns all dispatchers as DispatcherConfig sorted by ID
func (m *Manager) Dispatchers() []DispatcherConfig {
	result := make([]DispatcherConfig, 0)
	m.dispatchers.Range(func(id int, d dispatcher.Dispatcher) bool {
		switch v := d.(type) {
		case *dispatcher.WebhookDispatcher:
			result = append(result, DispatcherConfig{
				ID:       id,
				Name:     v.Name,
				Type:     "webhook",
				URL:      v.URL,
				Template: v.TemplateText,
			})
		case *dispatcher.CloudDispatcher:
			result = append(result, DispatcherConfig{
				ID:        id,
				Name:      v.Name,
				Type:      "cloud",
				APIKey:    v.APIKey,
				Prefix:    v.Prefix,
				ExpiresAt: v.ExpiresAt,
			})
		}
		return true
	})
	slices.SortFunc(result, func(a, b DispatcherConfig) int {
		return a.ID - b.ID
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
	// Get container and host from log event's ContainerID
	ctx, cancel := context.WithTimeout(m.ctx, 5*time.Second)
	defer cancel()

	c, host, err := m.listener.FindContainerWithHost(ctx, logEvent.ContainerID, nil)
	if err != nil {
		log.Error().Err(err).Str("containerID", logEvent.ContainerID).Msg("Failed to find container")
		return
	}

	// Skip logs from Dozzle's own containers to avoid feedback loops
	if c.Image == "amir20/dozzle" || strings.HasPrefix(c.Image, "amir20/dozzle:") {
		return
	}

	notificationContainer := FromContainerModel(c, host)
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

		sub.AddTriggeredContainer(notificationContainer.ID)

		// Check log filter
		if !sub.MatchesLog(notificationLog) {
			return true
		}

		// Update stats
		sub.TriggerCount.Add(1)
		now := time.Now()
		sub.LastTriggeredAt.Store(&now)

		log.Debug().Str("containerID", notificationContainer.ID).Interface("log", notificationLog.Message).Msg("Matched subscription")

		// Create notification
		notification := types.Notification{
			ID:        fmt.Sprintf("%s-%d", c.ID, time.Now().UnixNano()),
			Container: notificationContainer,
			Log:       &notificationLog,
			Subscription: types.SubscriptionConfig{
				ID:                  sub.ID,
				Name:                sub.Name,
				Enabled:             sub.Enabled,
				DispatcherID:        sub.DispatcherID,
				LogExpression:       sub.LogExpression,
				ContainerExpression: sub.ContainerExpression,
			},
			Timestamp: time.Now(),
		}

		// Send to the subscription's dispatcher
		if d, ok := m.dispatchers.Load(sub.DispatcherID); ok {
			go m.sendNotification(d, notification, sub.DispatcherID)
		}
		return true
	})
}

// processStatEvents processes stat events from the stats listener channel
func (m *Manager) processStatEvents() {
	for {
		select {
		case <-m.ctx.Done():
			return
		case event := <-m.statsListener.Channel():
			m.processStatEvent(event)
		}
	}
}

// processStatEvent processes a single stat event and sends notifications for matching metric subscriptions
func (m *Manager) processStatEvent(event ContainerStatEvent) {
	notificationStat := types.NotificationStat{
		CPUPercent:    event.Stat.CPUPercent,
		MemoryPercent: event.Stat.MemoryPercent,
		MemoryUsage:   event.Stat.MemoryUsage,
	}

	notificationContainer := FromContainerModel(event.Container, event.Host)

	m.subscriptions.Range(func(_ int, sub *Subscription) bool {
		// Skip disabled or non-metric subscriptions
		if !sub.Enabled || !sub.IsMetricAlert() {
			return true
		}

		// Check container filter first
		if !sub.MatchesContainer(notificationContainer) {
			return true
		}

		// Check metric expression
		if !sub.MatchesMetric(notificationStat) {
			return true
		}

		// Check per-container cooldown
		if sub.IsMetricCooldownActive(event.Stat.ID) {
			return true
		}

		// Set cooldown and update stats
		sub.SetMetricCooldown(event.Stat.ID)
		sub.AddTriggeredContainer(event.Stat.ID)
		sub.TriggerCount.Add(1)
		now := time.Now()
		sub.LastTriggeredAt.Store(&now)

		log.Debug().
			Str("containerID", event.Stat.ID).
			Float64("cpu", event.Stat.CPUPercent).
			Float64("memory", event.Stat.MemoryPercent).
			Str("subscription", sub.Name).
			Msg("Metric alert triggered")

		notification := types.Notification{
			ID:        fmt.Sprintf("%s-metric-%d", event.Stat.ID, time.Now().UnixNano()),
			Container: notificationContainer,
			Stat:      &notificationStat,
			Subscription: types.SubscriptionConfig{
				ID:                  sub.ID,
				Name:                sub.Name,
				Enabled:             sub.Enabled,
				DispatcherID:        sub.DispatcherID,
				MetricExpression:    sub.MetricExpression,
				ContainerExpression: sub.ContainerExpression,
				Cooldown:            sub.Cooldown,
			},
			Timestamp: time.Now(),
		}

		if d, ok := m.dispatchers.Load(sub.DispatcherID); ok {
			go m.sendNotification(d, notification, sub.DispatcherID)
		}
		return true
	})
}

// sendNotification sends a notification using the dispatcher
func (m *Manager) sendNotification(d dispatcher.Dispatcher, notification types.Notification, id int) {
	acquireCtx, acquireCancel := context.WithTimeout(m.ctx, time.Minute)
	defer acquireCancel()
	if err := m.sendSem.Acquire(acquireCtx, 1); err != nil {
		log.Warn().Err(err).Int("subscription", id).Msg("Notification dropped: too many pending")
		return
	}
	defer m.sendSem.Release(1)

	ctx, cancel := context.WithTimeout(m.ctx, 30*time.Second)
	defer cancel()

	if err := d.Send(ctx, notification); err != nil {
		log.Error().Err(err).Int("subscription", id).Msg("Failed to send notification")
	}
}

// WriteConfig writes the current configuration to a writer in YAML format
func (m *Manager) WriteConfig(w io.Writer) error {
	config := Config{
		Subscriptions: m.Subscriptions(),
		Dispatchers:   m.Dispatchers(),
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

	// Convert to types for HandleNotificationConfig
	subscriptions := make([]types.SubscriptionConfig, len(config.Subscriptions))
	for i, sub := range config.Subscriptions {
		subscriptions[i] = types.SubscriptionConfig{
			ID:                  sub.ID,
			Name:                sub.Name,
			Enabled:             sub.Enabled,
			DispatcherID:        sub.DispatcherID,
			LogExpression:       sub.LogExpression,
			ContainerExpression: sub.ContainerExpression,
			MetricExpression:    sub.MetricExpression,
			Cooldown:            sub.Cooldown,
		}
	}

	dispatchers := make([]types.DispatcherConfig, len(config.Dispatchers))
	for i, d := range config.Dispatchers {
		dispatchers[i] = types.DispatcherConfig{
			ID:        d.ID,
			Name:      d.Name,
			Type:      d.Type,
			URL:       d.URL,
			Template:  d.Template,
			APIKey:    d.APIKey,
			Prefix:    d.Prefix,
			ExpiresAt: d.ExpiresAt,
		}
	}

	return m.HandleNotificationConfig(subscriptions, dispatchers)
}

// HandleNotificationConfig implements agent.NotificationConfigHandler interface
// It atomically replaces all subscriptions and dispatchers with new state from the main server
func (m *Manager) HandleNotificationConfig(subscriptions []types.SubscriptionConfig, dispatchers []types.DispatcherConfig) error {
	// Clear existing state (with nil checks for defensive programming)
	if m.subscriptions != nil {
		m.subscriptions.Clear()
	} else {
		m.subscriptions = xsync.NewMap[int, *Subscription]()
	}
	if m.dispatchers != nil {
		m.dispatchers.Clear()
	} else {
		m.dispatchers = xsync.NewMap[int, dispatcher.Dispatcher]()
	}

	// Find max IDs to initialize counters
	var maxSubID, maxDispatcherID int
	for _, sub := range subscriptions {
		if sub.ID > maxSubID {
			maxSubID = sub.ID
		}
	}
	for _, d := range dispatchers {
		if d.ID > maxDispatcherID {
			maxDispatcherID = d.ID
		}
	}
	m.subscriptionCounter.Store(int32(maxSubID))
	m.dispatcherCounter.Store(int32(maxDispatcherID))

	// Load subscriptions (convert from types.SubscriptionConfig to Subscription)
	for _, sub := range subscriptions {
		s := &Subscription{
			ID:                  sub.ID,
			Name:                sub.Name,
			Enabled:             sub.Enabled,
			DispatcherID:        sub.DispatcherID,
			LogExpression:       sub.LogExpression,
			ContainerExpression: sub.ContainerExpression,
			MetricExpression:    sub.MetricExpression,
			Cooldown:            sub.Cooldown,
		}
		if err := m.loadSubscription(s); err != nil {
			return fmt.Errorf("failed to load subscription %s: %w", sub.Name, err)
		}
	}

	// Load dispatchers
	for _, dc := range dispatchers {
		d, err := createDispatcher(DispatcherConfig{
			ID:        dc.ID,
			Name:      dc.Name,
			Type:      dc.Type,
			URL:       dc.URL,
			Template:  dc.Template,
			APIKey:    dc.APIKey,
			Prefix:    dc.Prefix,
			ExpiresAt: dc.ExpiresAt,
		})
		if err != nil {
			return fmt.Errorf("failed to create dispatcher %s: %w", dc.Name, err)
		}
		m.dispatchers.Store(dc.ID, d)
		log.Debug().Int("id", dc.ID).Msg("Loaded dispatcher from state sync")
	}

	// Update listener to start/stop streams based on new subscriptions
	if m.listener != nil {
		if err := m.listener.UpdateStreams(); err != nil {
			return fmt.Errorf("failed to update listener streams: %w", err)
		}
	}

	log.Debug().Int("subscriptions", len(subscriptions)).Int("dispatchers", len(dispatchers)).Msg("Replaced notification state")
	return nil
}

// createDispatcher creates a dispatcher from a DispatcherConfig
func createDispatcher(config DispatcherConfig) (dispatcher.Dispatcher, error) {
	switch config.Type {
	case "webhook":
		return dispatcher.NewWebhookDispatcher(config.Name, config.URL, config.Template)
	case "cloud":
		return dispatcher.NewCloudDispatcher(config.Name, config.APIKey, config.Prefix, config.ExpiresAt)
	default:
		return nil, fmt.Errorf("unknown dispatcher type: %s", config.Type)
	}
}

// loadSubscription loads a subscription with its existing ID (used when loading from config)
func (m *Manager) loadSubscription(sub *Subscription) error {
	// Compile container expression if provided
	if sub.ContainerExpression != "" {
		program, err := expr.Compile(sub.ContainerExpression, expr.Env(types.NotificationContainer{}))
		if err != nil {
			return fmt.Errorf("failed to compile container expression: %w", err)
		}
		sub.ContainerProgram = program
	}

	// Compile log expression if provided
	if sub.LogExpression != "" {
		program, err := expr.Compile(sub.LogExpression, expr.Env(types.NotificationLog{}))
		if err != nil {
			return fmt.Errorf("failed to compile log expression: %w", err)
		}
		sub.LogProgram = program
	}

	// Compile metric expression if provided
	if sub.MetricExpression != "" {
		program, err := expr.Compile(sub.MetricExpression, expr.Env(types.NotificationStat{}))
		if err != nil {
			return fmt.Errorf("failed to compile metric expression: %w", err)
		}
		sub.MetricProgram = program
	}

	if sub.MetricCooldowns == nil {
		sub.MetricCooldowns = xsync.NewMap[string, time.Time]()
	}

	m.subscriptions.Store(sub.ID, sub)
	log.Debug().Str("name", sub.Name).Int("id", sub.ID).Msg("Loaded subscription")
	return nil
}

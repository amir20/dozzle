package notification

import (
	"fmt"
	"io"
	"time"

	"github.com/amir20/dozzle/internal/notification/dispatcher"
	"github.com/amir20/dozzle/internal/utils"
	"github.com/amir20/dozzle/types"
	"github.com/puzpuzpuz/xsync/v4"
	"github.com/rs/zerolog/log"
	"go.yaml.in/yaml/v3"
)

// WriteConfig writes the current configuration to a writer in YAML format.
// Cloud dispatchers are excluded because they are persisted separately in cloud.yml.
func (m *Manager) WriteConfig(w io.Writer) error {
	allDispatchers := m.Dispatchers()
	dispatchers := make([]DispatcherConfig, 0, len(allDispatchers))
	for _, d := range allDispatchers {
		if d.Type != "cloud" {
			dispatchers = append(dispatchers, d)
		}
	}

	config := Config{
		Subscriptions: m.Subscriptions(),
		Dispatchers:   dispatchers,
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
			EventExpression:     sub.EventExpression,
			Cooldown:            sub.Cooldown,
			SampleWindow:        sub.SampleWindow,
		}
	}

	dispatchers := make([]types.DispatcherConfig, len(config.Dispatchers))
	for i, d := range config.Dispatchers {
		dispatchers[i] = types.DispatcherConfig{
			ID:       d.ID,
			Name:     d.Name,
			Type:     d.Type,
			URL:      d.URL,
			Template: d.Template,
			Headers:  d.Headers,
		}
	}

	return m.HandleNotificationConfig(subscriptions, dispatchers)
}

// HandleNotificationConfig implements agent.NotificationConfigHandler interface
// It atomically replaces all subscriptions and dispatchers with new state from the main server
func (m *Manager) HandleNotificationConfig(subscriptions []types.SubscriptionConfig, dispatchers []types.DispatcherConfig) error {
	// Snapshot existing subscriptions to preserve runtime stats
	existing := make(map[int]*Subscription)
	m.subscriptions.Range(func(id int, sub *Subscription) bool {
		existing[id] = sub
		return true
	})

	// Build set of incoming IDs and remove stale subscriptions
	incomingIDs := make(map[int]struct{}, len(subscriptions))
	for _, sub := range subscriptions {
		incomingIDs[sub.ID] = struct{}{}
	}
	for id := range existing {
		if _, ok := incomingIDs[id]; !ok {
			m.subscriptions.Delete(id)
		}
	}

	// Clear dispatchers (no stats to preserve)
	m.dispatchers.Clear()

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

	// Load subscriptions, preserving runtime stats from existing ones
	for _, sub := range subscriptions {
		s := &Subscription{
			ID:                  sub.ID,
			Name:                sub.Name,
			Enabled:             sub.Enabled,
			DispatcherID:        sub.DispatcherID,
			LogExpression:       sub.LogExpression,
			ContainerExpression: sub.ContainerExpression,
			MetricExpression:    sub.MetricExpression,
			EventExpression:     sub.EventExpression,
			Cooldown:            sub.Cooldown,
			SampleWindow:        sub.SampleWindow,
		}

		if old, ok := existing[sub.ID]; ok {
			s.TriggerCount.Store(old.TriggerCount.Load())
			s.LastTriggeredAt.Store(old.LastTriggeredAt.Load())

			// Clone TriggeredContainerIDs to avoid sharing with old subscription
			s.TriggeredContainerIDs = xsync.NewMap[string, struct{}]()
			if old.TriggeredContainerIDs != nil {
				old.TriggeredContainerIDs.Range(func(id string, v struct{}) bool {
					s.TriggeredContainerIDs.Store(id, v)
					return true
				})
			}

			// Clone MetricCooldowns to avoid sharing with old subscription
			s.MetricCooldowns = xsync.NewMap[string, time.Time]()
			if old.MetricCooldowns != nil {
				old.MetricCooldowns.Range(func(id string, t time.Time) bool {
					s.MetricCooldowns.Store(id, t)
					return true
				})
			}

			s.EventCooldowns = xsync.NewMap[string, time.Time]()
			if old.EventCooldowns != nil {
				old.EventCooldowns.Range(func(id string, t time.Time) bool {
					s.EventCooldowns.Store(id, t)
					return true
				})
			}

			// MetricSampleBuffers: start fresh since ring buffers can't be safely cloned
		}

		if err := m.loadSubscription(s); err != nil {
			return fmt.Errorf("failed to load subscription %s: %w", sub.Name, err)
		}
	}

	// Load dispatchers (cloud dispatchers are skipped; they are managed via cloud.yml)
	for _, dc := range dispatchers {
		if dc.Type == "cloud" {
			continue
		}
		d, err := createDispatcher(DispatcherConfig{
			ID:       dc.ID,
			Name:     dc.Name,
			Type:     dc.Type,
			URL:      dc.URL,
			Template: dc.Template,
			Headers:  dc.Headers,
		})
		if err != nil {
			log.Warn().Err(err).Str("name", dc.Name).Str("type", dc.Type).Msg("Skipping unknown dispatcher type")
			continue
		}
		m.dispatchers.Store(dc.ID, d)
		log.Debug().Int("id", dc.ID).Msg("Loaded dispatcher from state sync")
	}

	m.updateListeners()

	log.Debug().Int("subscriptions", len(subscriptions)).Int("dispatchers", len(dispatchers)).Msg("Replaced notification state")
	return nil
}

// createDispatcher creates a dispatcher from a DispatcherConfig.
// Cloud dispatchers are not created here; they are managed via cloud.yml and SetCloudDispatcher.
func createDispatcher(config DispatcherConfig) (dispatcher.Dispatcher, error) {
	switch config.Type {
	case "webhook":
		return dispatcher.NewWebhookDispatcher(config.Name, config.URL, config.Template, config.Headers)
	default:
		return nil, fmt.Errorf("unknown dispatcher type: %s", config.Type)
	}
}

// loadSubscription loads a subscription with its existing ID (used when loading from config)
func (m *Manager) loadSubscription(sub *Subscription) error {
	if err := sub.CompileExpressions(); err != nil {
		return err
	}

	if sub.MetricCooldowns == nil {
		sub.MetricCooldowns = xsync.NewMap[string, time.Time]()
	}
	if sub.MetricSampleBuffers == nil {
		sub.MetricSampleBuffers = xsync.NewMap[string, *utils.RingBuffer[bool]]()
	}
	if sub.EventCooldowns == nil {
		sub.EventCooldowns = xsync.NewMap[string, time.Time]()
	}

	m.subscriptions.Store(sub.ID, sub)
	log.Debug().Str("name", sub.Name).Int("id", sub.ID).Msg("Loaded subscription")
	return nil
}

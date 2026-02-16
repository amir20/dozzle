package notification

import (
	"fmt"
	"io"
	"time"

	"github.com/amir20/dozzle/internal/notification/dispatcher"
	"github.com/amir20/dozzle/types"
	"github.com/puzpuzpuz/xsync/v4"
	"github.com/rs/zerolog/log"
	"go.yaml.in/yaml/v3"
)

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
	// Clear existing state
	m.subscriptions.Clear()
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

	m.updateListeners()

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
	if err := sub.CompileExpressions(); err != nil {
		return err
	}

	if sub.MetricCooldowns == nil {
		sub.MetricCooldowns = xsync.NewMap[string, time.Time]()
	}

	m.subscriptions.Store(sub.ID, sub)
	log.Debug().Str("name", sub.Name).Int("id", sub.ID).Msg("Loaded subscription")
	return nil
}

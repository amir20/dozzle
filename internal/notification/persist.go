package notification

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the YAML configuration structure
type Config struct {
	Dispatcher    DispatcherConfig `yaml:"dispatcher"`
	Subscriptions []*Subscription  `yaml:"subscriptions"`
}

// LoadFromFile loads config and returns a fully configured Manager
func LoadFromFile(path string, containerService ContainerService) (*Manager, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No config file is fine, return nil
		}
		return nil, fmt.Errorf("failed to read notifications config: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse notifications config: %w", err)
	}

	// Validate dispatcher config
	if config.Dispatcher.Type == "" {
		config.Dispatcher.Type = "simple" // Default to simple
	}
	if config.Dispatcher.URL == "" {
		return nil, fmt.Errorf("dispatcher.url is required")
	}

	// Validate subscriptions
	for _, sub := range config.Subscriptions {
		if sub.Name == "" {
			return nil, fmt.Errorf("subscription missing name")
		}
		if sub.ContainerFilter == "" {
			return nil, fmt.Errorf("subscription %q missing container_filter", sub.Name)
		}
		if sub.LogFilter == "" {
			return nil, fmt.Errorf("subscription %q missing log_filter", sub.Name)
		}
	}

	// Create dispatcher based on type
	var dispatcher Dispatcher
	switch config.Dispatcher.Type {
	case "simple":
		dispatcher = NewWebhookDispatcher(&config.Dispatcher)
	default:
		return nil, fmt.Errorf("unsupported dispatcher type: %s", config.Dispatcher.Type)
	}

	// Create and configure manager
	manager := NewManager(containerService, dispatcher)
	if err := manager.LoadSubscriptions(config.Subscriptions); err != nil {
		return nil, err
	}

	return manager, nil
}

// SaveToFile saves dispatcher config and subscriptions to a YAML file
func SaveToFile(path string, dispatcher *DispatcherConfig, subs []*Subscription) error {
	config := Config{
		Dispatcher:    *dispatcher,
		Subscriptions: subs,
	}

	data, err := yaml.Marshal(&config)
	if err != nil {
		return fmt.Errorf("failed to marshal notifications config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write notifications config: %w", err)
	}

	return nil
}

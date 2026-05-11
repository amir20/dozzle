// Package migration contains one-time data migrations that run at startup.
// Delete this package once all users have migrated.
package migration

import (
	"os"
	"time"

	"github.com/rs/zerolog/log"
	"go.yaml.in/yaml/v3"
)

type notificationConfig struct {
	Subscriptions []subscription `yaml:"subscriptions"`
	Dispatchers   []dispatcher   `yaml:"dispatchers"`
}

type subscription struct {
	ID                  int    `yaml:"id"`
	Name                string `yaml:"name"`
	Enabled             bool   `yaml:"enabled"`
	DispatcherID        int    `yaml:"dispatcherId"`
	LogExpression       string `yaml:"logExpression"`
	ContainerExpression string `yaml:"containerExpression"`
	MetricExpression    string `yaml:"metricExpression,omitempty"`
	EventExpression     string `yaml:"eventExpression,omitempty"`
	Cooldown            int    `yaml:"cooldown,omitempty"`
	SampleWindow        int    `yaml:"sampleWindow,omitempty"`
}

type dispatcher struct {
	ID        int               `yaml:"id"`
	Name      string            `yaml:"name"`
	Type      string            `yaml:"type"`
	URL       string            `yaml:"url,omitempty"`
	Template  string            `yaml:"template,omitempty"`
	Headers   map[string]string `yaml:"headers,omitempty"`
	APIKey    string            `yaml:"apiKey,omitempty"`
	Prefix    string            `yaml:"prefix,omitempty"`
	ExpiresAt *time.Time        `yaml:"expiresAt,omitempty"`
}

type cloudConfig struct {
	APIKey    string     `yaml:"apiKey"`
	Prefix    string     `yaml:"prefix"`
	ExpiresAt *time.Time `yaml:"expiresAt,omitempty"`
}

// MigrateCloudConfig splits the old notifications.yml (which embedded cloud credentials
// in the dispatchers list) into two files: a clean notifications.yml and a new cloud.yml.
// Subscriptions are remapped from the old cloud dispatcher ID to 0.
// No-op if cloud.yml already exists or there is no cloud dispatcher to migrate.
func MigrateCloudConfig(notificationsPath, cloudPath string) {
	if fileExists(cloudPath) || !fileExists(notificationsPath) {
		return
	}

	data, err := os.ReadFile(notificationsPath)
	if err != nil {
		return
	}

	var config notificationConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return
	}

	// Find cloud dispatcher
	var cloud *cloudConfig
	var cloudID int
	remaining := make([]dispatcher, 0, len(config.Dispatchers))
	for _, d := range config.Dispatchers {
		if d.Type == "cloud" && d.APIKey != "" {
			cloud = &cloudConfig{APIKey: d.APIKey, Prefix: d.Prefix, ExpiresAt: d.ExpiresAt}
			cloudID = d.ID
		} else {
			remaining = append(remaining, d)
		}
	}

	if cloud == nil {
		return
	}

	log.Info().Int("oldDispatcherId", cloudID).Msg("Migrating cloud config from notifications.yml to cloud.yml")

	// Remap subscriptions from old cloud dispatcher ID to 0
	for i := range config.Subscriptions {
		if config.Subscriptions[i].DispatcherID == cloudID {
			config.Subscriptions[i].DispatcherID = 0
		}
	}
	config.Dispatchers = remaining

	// Write cloud.yml
	if err := writeYAML(cloudPath, cloud); err != nil {
		log.Error().Err(err).Msg("Could not write cloud.yml")
		return
	}

	// Rewrite notifications.yml
	if err := writeYAML(notificationsPath, config); err != nil {
		log.Error().Err(err).Msg("Could not rewrite notifications.yml")
		return
	}

	log.Info().Msg("Migration complete: created cloud.yml and updated notifications.yml")
}

func writeYAML(path string, v any) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := yaml.NewEncoder(file)
	defer encoder.Close()
	return encoder.Encode(v)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

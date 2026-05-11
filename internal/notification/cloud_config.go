package notification

import (
	"io"
	"time"

	"go.yaml.in/yaml/v3"
)

// CloudConfig holds the cloud dispatcher credentials and metadata.
type CloudConfig struct {
	APIKey    string     `yaml:"apiKey"`
	Prefix    string     `yaml:"prefix"`
	ExpiresAt *time.Time `yaml:"expiresAt,omitempty"`
	// StreamLogs controls whether container logs are streamed to Dozzle Cloud.
	// nil means default (enabled) — preserves behavior for configs written
	// before this field existed.
	StreamLogs *bool `yaml:"streamLogs,omitempty"`
}

// StreamLogsEnabled reports whether the bulk log stream to cloud should run.
// Defaults to true when the field is unset or the config is nil.
func (c *CloudConfig) StreamLogsEnabled() bool {
	if c == nil || c.StreamLogs == nil {
		return true
	}
	return *c.StreamLogs
}

// WriteCloudConfig encodes the given CloudConfig to the writer in YAML format.
func WriteCloudConfig(w io.Writer, config CloudConfig) error {
	encoder := yaml.NewEncoder(w)
	defer encoder.Close()
	return encoder.Encode(config)
}

// LoadCloudConfig decodes a CloudConfig from the reader.
func LoadCloudConfig(r io.Reader) (CloudConfig, error) {
	var config CloudConfig
	decoder := yaml.NewDecoder(r)
	if err := decoder.Decode(&config); err != nil {
		return CloudConfig{}, err
	}
	return config, nil
}


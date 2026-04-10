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

// MigrateCloudFromDispatchers extracts the first cloud dispatcher entry from the
// list, returns a CloudConfig built from it, the remaining dispatchers (without
// the cloud entry), and ok=true. If no cloud dispatcher is found ok=false and
// the original slice is returned unchanged.
func MigrateCloudFromDispatchers(dispatchers []DispatcherConfig) (CloudConfig, []DispatcherConfig, bool) {
	for i, d := range dispatchers {
		if d.Type == "cloud" {
			cc := CloudConfig{
				APIKey:    d.APIKey,
				Prefix:    d.Prefix,
				ExpiresAt: d.ExpiresAt,
			}
			remaining := make([]DispatcherConfig, 0, len(dispatchers)-1)
			remaining = append(remaining, dispatchers[:i]...)
			remaining = append(remaining, dispatchers[i+1:]...)
			return cc, remaining, true
		}
	}
	return CloudConfig{}, dispatchers, false
}

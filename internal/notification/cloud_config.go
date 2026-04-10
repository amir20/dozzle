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


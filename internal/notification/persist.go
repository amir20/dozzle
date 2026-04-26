package notification

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/amir20/dozzle/internal/notification/dispatcher"
	"github.com/rs/zerolog/log"
)

const (
	DefaultNotificationConfigPath = "./data/notifications.yml"
	DefaultCloudConfigPath        = "./data/cloud.yml"
)

// Persister loads and saves notification and cloud configs, applying cloud
// configs to the underlying Manager. Safe for concurrent use.
type Persister struct {
	Manager          *Manager
	NotificationPath string
	CloudPath        string

	mu          sync.RWMutex
	cloudConfig *CloudConfig
}

// Load reads notification and cloud configs from disk and applies them to the
// manager. Missing files are ignored; parse errors are logged and skipped.
func (p *Persister) Load() {
	if file, err := os.Open(p.NotificationPath); err == nil {
		defer file.Close()
		if err := p.Manager.LoadConfig(file); err != nil {
			log.Warn().Err(err).Msg("Could not load notification config")
		} else {
			log.Debug().Str("path", p.NotificationPath).Msg("Loaded notification config")
		}
	}

	if file, err := os.Open(p.CloudPath); err == nil {
		defer file.Close()
		cc, err := LoadCloudConfig(file)
		if err != nil {
			log.Warn().Err(err).Msg("Could not load cloud config")
			return
		}
		p.mu.Lock()
		p.cloudConfig = &cc
		p.mu.Unlock()
		p.applyCloudDispatcher(&cc)
		log.Debug().Str("path", p.CloudPath).Msg("Loaded cloud config")
	}
}

// SaveNotifications writes the manager's current notification config to disk.
func (p *Persister) SaveNotifications() {
	if err := ensureDir(p.NotificationPath); err != nil {
		log.Error().Err(err).Msg("Could not create data directory")
		return
	}
	file, err := os.Create(p.NotificationPath)
	if err != nil {
		log.Error().Err(err).Msg("Could not create notification config file")
		return
	}
	defer file.Close()

	if err := p.Manager.WriteConfig(file); err != nil {
		log.Error().Err(err).Msg("Could not write notification config")
	}
}

// SaveCloud writes the current cloud config to disk. No-op when unset.
func (p *Persister) SaveCloud() {
	p.mu.RLock()
	cc := p.cloudConfig
	p.mu.RUnlock()

	if cc == nil {
		return
	}

	if err := ensureDir(p.CloudPath); err != nil {
		log.Error().Err(err).Msg("Could not create data directory")
		return
	}
	file, err := os.Create(p.CloudPath)
	if err != nil {
		log.Error().Err(err).Msg("Could not create cloud config file")
		return
	}
	defer file.Close()

	if err := WriteCloudConfig(file, *cc); err != nil {
		log.Error().Err(err).Msg("Could not write cloud config")
	}
}

// CloudConfig returns the current cloud config, or nil if unset.
func (p *Persister) CloudConfig() *CloudConfig {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.cloudConfig
}

// SetCloudConfig stores the cloud config, updates the cloud dispatcher, and
// persists the config to disk.
func (p *Persister) SetCloudConfig(cc *CloudConfig) {
	p.mu.Lock()
	p.cloudConfig = cc
	p.mu.Unlock()
	p.applyCloudDispatcher(cc)
	p.SaveCloud()
}

// SetCloudStreamLogs updates the StreamLogs flag on the current cloud config
// and persists it. No-op when no cloud config is set.
func (p *Persister) SetCloudStreamLogs(enabled bool) {
	p.mu.Lock()
	if p.cloudConfig == nil {
		p.mu.Unlock()
		return
	}
	v := enabled
	p.cloudConfig.StreamLogs = &v
	p.mu.Unlock()
	p.SaveCloud()
}

// RemoveCloudConfig clears the cloud config, clears the cloud dispatcher, and
// removes the cloud config file from disk.
func (p *Persister) RemoveCloudConfig() {
	p.mu.Lock()
	p.cloudConfig = nil
	p.mu.Unlock()
	p.Manager.ClearCloudDispatcher()
	if err := os.Remove(p.CloudPath); err != nil && !os.IsNotExist(err) {
		log.Error().Err(err).Msg("Could not remove cloud config file")
	}
}

func (p *Persister) applyCloudDispatcher(cc *CloudConfig) {
	if cc == nil {
		return
	}
	d, err := dispatcher.NewCloudDispatcher("Dozzle Cloud", cc.APIKey, cc.Prefix, cc.ExpiresAt)
	if err != nil {
		log.Error().Err(err).Msg("Could not create cloud dispatcher from config")
		return
	}
	p.Manager.SetCloudDispatcher(d)
}

func ensureDir(path string) error {
	return os.MkdirAll(filepath.Dir(path), 0755)
}

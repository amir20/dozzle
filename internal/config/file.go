// Package config reads and writes /data/dozzle.yml, the settings the setup
// wizard saves. The file is read once at startup and sits under flags and env
// vars: a value set either of those ways always wins, so the file never
// silently fights what an operator wrote in their compose file.
package config

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// Path is relative to the working directory, like every other file under
// ./data, which is / inside the image.
const Path = "./data/dozzle.yml"

// File mirrors dozzle.yml. Pointers keep "not in the file" apart from "false".
type File struct {
	AuthProvider  *string `yaml:"authProvider,omitempty"`
	EnableActions *bool   `yaml:"enableActions,omitempty"`
	EnableShell   *bool   `yaml:"enableShell,omitempty"`
	// AutoUpdate is off, daily or weekly. Unlike the settings above it is read
	// again every minute, so changing it needs no restart.
	AutoUpdate *string `yaml:"autoUpdate,omitempty"`
	// AutoUpdateTime is "HH:MM" in the server's local time.
	AutoUpdateTime *string `yaml:"autoUpdateTime,omitempty"`
	// SetupWindowStartedAt is written just before the wizard restarts Dozzle,
	// so the restart carries the no-login window forward instead of reopening it.
	SetupWindowStartedAt *time.Time `yaml:"setupWindowStartedAt,omitempty"`
}

const (
	AutoUpdateOff    = "off"
	AutoUpdateDaily  = "daily"
	AutoUpdateWeekly = "weekly"
	// DefaultAutoUpdateTime is quiet on most servers.
	DefaultAutoUpdateTime = "03:00"
)

var autoUpdateTimePattern = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)

// ValidAutoUpdateMode reports whether mode is off, daily or weekly.
func ValidAutoUpdateMode(mode string) bool {
	return mode == AutoUpdateOff || mode == AutoUpdateDaily || mode == AutoUpdateWeekly
}

// ValidAutoUpdateTime reports whether t is a 24h "HH:MM".
func ValidAutoUpdateTime(t string) bool {
	return autoUpdateTimePattern.MatchString(t)
}

var mu sync.Mutex

// FreshDataDir reports whether dir holds nothing from an earlier run: no
// profiles, users, notification rules or dozzle.yml. It has to be called
// before this process writes anything there. Dotfiles and lost+found are
// filesystem noise, not Dozzle state. A missing or unreadable dir is not fresh,
// so an error never opens anything up.
func FreshDataDir(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return errors.Is(err, os.ErrNotExist)
	}
	for _, entry := range entries {
		name := entry.Name()
		if name == "lost+found" || name[0] == '.' {
			continue
		}
		return false
	}
	return true
}

// Load returns an empty File when the file does not exist yet.
func Load(path string) (File, error) {
	mu.Lock()
	defer mu.Unlock()
	return load(path)
}

func load(path string) (File, error) {
	var f File
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return f, nil
	}
	if err != nil {
		return f, err
	}
	if err := yaml.Unmarshal(data, &f); err != nil {
		return f, err
	}
	return f, nil
}

// Update applies change to the current file and writes it back atomically, so
// a crash mid-write can never leave a half-written file that fails startup.
func Update(path string, change func(*File)) error {
	mu.Lock()
	defer mu.Unlock()

	f, err := load(path)
	if err != nil {
		return err
	}
	change(&f)

	data, err := yaml.Marshal(f)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".dozzle-*.yml")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

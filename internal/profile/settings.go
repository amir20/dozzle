package profile

import (
	"encoding/json"
	"errors"

	"os"
	"path/filepath"

	"github.com/amir20/dozzle/internal/auth"
	log "github.com/sirupsen/logrus"
)

type Settings struct {
	Search            bool    `json:"search"`
	MenuWidth         float32 `json:"menuWidth"`
	SmallerScrollbars bool    `json:"smallerScrollbars"`
	ShowTimestamp     bool    `json:"showTimestamp"`
	ShowStd           bool    `json:"showStd"`
	ShowAllContainers bool    `json:"showAllContainers"`
	SoftWrap          bool    `json:"softWrap"`
	CollapseNav       bool    `json:"collapseNav"`
	AutomaticRedirect bool    `json:"automaticRedirect"`
	Size              string  `json:"size,omitempty"`
	LightTheme        string  `json:"lightTheme,omitempty"`
	HourStyle         string  `json:"hourStyle,omitempty"`
}

var data_path string

func init() {
	path, err := filepath.Abs("./data")
	if err != nil {
		log.Fatalf("Unable to get absolute path for data directory: %s", err)
		return
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.Mkdir(path, 0755); err != nil {
			log.Fatalf("Unable to create data directory: %s", err)
			return
		}
	}
	data_path = path
}

func SaveUserSettings(user auth.User, settings Settings) error {
	path := filepath.Join(data_path, user.Username)

	// Create user directory if it doesn't exist
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.Mkdir(path, 0755); err != nil {
			return err
		}
	}

	settings_path := filepath.Join(path, "settings.json")

	data, err := json.MarshalIndent(settings, "", " ")

	if err != nil {
		return err
	}

	f, err := os.Create(settings_path)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.Write(data); err != nil {
		return err
	}

	log.Debugf("Saved settings for user %s", user.Username)

	return f.Sync()
}

func LoadUserSettings(user auth.User) (Settings, error) {
	path := filepath.Join(data_path, user.Username)
	settings_path := filepath.Join(path, "settings.json")

	if _, err := os.Stat(settings_path); os.IsNotExist(err) {
		return Settings{}, errors.New("Settings file does not exist")
	}

	f, err := os.Open(settings_path)
	if err != nil {
		return Settings{}, err
	}
	defer f.Close()

	var settings Settings
	if err := json.NewDecoder(f).Decode(&settings); err != nil {
		return Settings{}, err
	}

	return settings, nil
}

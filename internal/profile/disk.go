package profile

import (
	"encoding/json"
	"errors"
	"io"
	"sync"

	"os"
	"path/filepath"

	"github.com/amir20/dozzle/internal/auth"
	log "github.com/sirupsen/logrus"
)

const (
	PROFILE_FILENAME = "profile.json"
)

var missingProfileErr = errors.New("Profile file does not exist")

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

type Profile struct {
	Settings Settings `json:"settings,omitempty"`
	Starred  []string `json:"starred,omitempty"`
}

var dataPath string
var mux = &sync.Mutex{}

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
	dataPath = path
}

func SaveUserProfile(user auth.User, reader io.Reader) error {
	mux.Lock()
	defer mux.Unlock()
	var existingProfile Profile
	existingProfile, err := LoadUserProfile(user)
	if err != nil && err != missingProfileErr {
		return err
	}

	// Write the new settings to the existing profile
	if err := json.NewDecoder(reader).Decode(&existingProfile); err != nil {
		return err
	}

	// Create directory if it doesn't exist
	path := filepath.Join(dataPath, user.Username)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.Mkdir(path, 0755); err != nil {
			return err
		}
	}

	filePath := filepath.Join(path, PROFILE_FILENAME)
	data, err := json.MarshalIndent(existingProfile, "", "  ")

	if err != nil {
		return err
	}

	f, err := os.Create(filePath)
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

func LoadUserProfile(user auth.User) (Profile, error) {
	path := filepath.Join(dataPath, user.Username)
	profilePath := filepath.Join(path, PROFILE_FILENAME)

	if _, err := os.Stat(profilePath); os.IsNotExist(err) {
		return Profile{}, missingProfileErr
	}

	f, err := os.Open(profilePath)
	if err != nil {
		return Profile{}, err
	}
	defer f.Close()

	var profile Profile
	if err := json.NewDecoder(f).Decode(&profile); err != nil {
		return Profile{}, err
	}

	return profile, nil
}

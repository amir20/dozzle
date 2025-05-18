package profile

import (
	"encoding/json"
	"errors"
	"io"
	"sync"

	"os"
	"path/filepath"

	"github.com/amir20/dozzle/internal/auth"
	"github.com/rs/zerolog/log"
)

const (
	profileFilename = "profile.json"
)

var errMissingProfileErr = errors.New("Profile file does not exist")

type Settings struct {
	Search            bool    `json:"search"`
	MenuWidth         float32 `json:"menuWidth"`
	SmallerScrollbars bool    `json:"smallerScrollbars"`
	ShowTimestamp     bool    `json:"showTimestamp"`
	ShowStd           bool    `json:"showStd"`
	ShowAllContainers bool    `json:"showAllContainers"`
	SoftWrap          bool    `json:"softWrap"`
	CollapseNav       bool    `json:"collapseNav"`
	AutomaticRedirect string  `json:"automaticRedirect"`
	Size              string  `json:"size,omitempty"`
	Compact           bool    `json:"compact"`
	LightTheme        string  `json:"lightTheme,omitempty"`
	HourStyle         string  `json:"hourStyle,omitempty"`
	DateLocale        string  `json:"dateLocale,omitempty"`
	Locale            string  `json:"locale"`
}

type Profile struct {
	Settings        *Settings     `json:"settings,omitempty"`
	Pinned          []string      `json:"pinned"`
	VisibleKeys     []interface{} `json:"visibleKeys,omitempty"`
	ReleaseSeen     string        `json:"releaseSeen,omitempty"`
	CollapsedGroups []string      `json:"collapsedGroups"`
}

var dataPath string
var mux = &sync.Mutex{}

func init() {
	path, err := filepath.Abs("./data")
	if err != nil {
		log.Fatal().Err(err).Msg("Unable to get absolute path")
		return
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.Mkdir(path, 0755); err != nil {
			log.Fatal().Err(err).Msg("Unable to create data directory")
			return
		}
	}
	dataPath = path
}

func UpdateFromReader(user auth.User, reader io.Reader) error {
	mux.Lock()
	defer mux.Unlock()
	existingProfile, err := Load(user)
	if err != nil && err != errMissingProfileErr {
		log.Error().Err(err).Msg("Unable to load profile. Overwriting it.")
	}

	if err := json.NewDecoder(reader).Decode(&existingProfile); err != nil {
		return err
	}

	return Save(user, existingProfile)
}

func Save(user auth.User, profile Profile) error {
	path := filepath.Join(dataPath, user.Username)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.Mkdir(path, 0755); err != nil {
			return err
		}
	}

	filePath := filepath.Join(path, profileFilename)
	data, err := json.MarshalIndent(profile, "", "  ")

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

	log.Debug().Str("path", filePath).Msg("Profile saved")

	return f.Sync()
}

func Load(user auth.User) (Profile, error) {
	path := filepath.Join(dataPath, user.Username)
	profilePath := filepath.Join(path, profileFilename)

	if _, err := os.Stat(profilePath); os.IsNotExist(err) {
		return Profile{}, errMissingProfileErr
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

	if profile.Pinned == nil {
		profile.Pinned = []string{}
	}

	return profile, nil
}

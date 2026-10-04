package web

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/amir20/dozzle/internal/container"
	"github.com/rs/zerolog/log"
)

// The auto-update schedule must not apply again an image that was already
// taken off a container. Two things do that: a scheduled update of Dozzle
// itself that its helper rolled back, and someone rolling a container back.
// Either way the tag still names that image, so every later night would offer
// the same update and repeat the outage.
//
// Each is written down as the digest to skip, keyed by what it was skipped
// for, in auto-update-skips.json next to dozzle.yml. A skip lasts until the
// registry serves a different digest: someone pushed a fix.

// selfSkipKey is Dozzle's own container. Container keys are host/name, which
// always holds a slash, so the two never meet.
const selfSkipKey = "dozzle"

var updateSkipsMu sync.Mutex

func updateSkipsPath() string {
	return filepath.Join(filepath.Dir(setupConfigPath), "auto-update-skips.json")
}

// autoUpdateAttemptPath is where an older Dozzle wrote its own skip, as a bare
// digest. It is still read, so an upgrade keeps a skip it inherits.
func autoUpdateAttemptPath() string {
	return filepath.Join(filepath.Dir(setupConfigPath), "auto-update-attempt")
}

// containerSkipKey is a container's key: its name, since an update or a
// rollback gives it a new id.
func containerSkipKey(c container.Container) string {
	return c.Host + "/" + c.Name
}

// readUpdateSkipsLocked reads the skips. A missing or unreadable file is no
// skips at all: the worst it costs is one more attempt.
func readUpdateSkipsLocked() map[string]string {
	skips := map[string]string{}
	if b, err := os.ReadFile(updateSkipsPath()); err == nil {
		if err := json.Unmarshal(b, &skips); err != nil {
			log.Warn().Err(err).Msg("auto update: ignoring unreadable skip list")
			skips = map[string]string{}
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		log.Warn().Err(err).Msg("auto update: could not read skip list")
	}
	if _, ok := skips[selfSkipKey]; !ok {
		if b, err := os.ReadFile(autoUpdateAttemptPath()); err == nil {
			if digest := strings.TrimSpace(string(b)); digest != "" {
				skips[selfSkipKey] = digest
			}
		}
	}
	return skips
}

func writeUpdateSkipsLocked(skips map[string]string) {
	// The legacy file is folded in on every read, so it goes once the list
	// holds what it said.
	_ = os.Remove(autoUpdateAttemptPath())
	path := updateSkipsPath()
	if len(skips) == 0 {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			log.Warn().Err(err).Msg("auto update: could not clear skip list")
		}
		return
	}
	b, err := json.MarshalIndent(skips, "", "  ")
	if err != nil {
		return
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0644); err != nil {
		log.Warn().Err(err).Msg("auto update: could not record skipped image")
		return
	}
	if err := os.Rename(tmp, path); err != nil {
		log.Warn().Err(err).Msg("auto update: could not record skipped image")
		_ = os.Remove(tmp)
	}
}

// recordUpdateSkip makes the schedule skip digest for key.
func recordUpdateSkip(key, digest string) {
	digest = container.DigestOf(strings.TrimSpace(digest))
	if digest == "" {
		return
	}
	updateSkipsMu.Lock()
	defer updateSkipsMu.Unlock()
	skips := readUpdateSkipsLocked()
	skips[key] = digest
	writeUpdateSkipsLocked(skips)
}

// clearUpdateSkip forgets key's skip, if it has one.
func clearUpdateSkip(key string) {
	updateSkipsMu.Lock()
	defer updateSkipsMu.Unlock()
	skips := readUpdateSkipsLocked()
	if _, ok := skips[key]; !ok {
		return
	}
	delete(skips, key)
	writeUpdateSkipsLocked(skips)
}

// skippedUpdate reports whether the schedule must leave key's update to
// remote alone. A skip for any other digest is stale, since a newer image
// replaced the one it was for, and is forgotten.
func skippedUpdate(key, remote string) bool {
	remote = container.DigestOf(remote)
	if remote == "" {
		return false
	}
	updateSkipsMu.Lock()
	defer updateSkipsMu.Unlock()
	skips := readUpdateSkipsLocked()
	skipped, ok := skips[key]
	if !ok {
		return false
	}
	if skipped == remote {
		return true
	}
	delete(skips, key)
	writeUpdateSkipsLocked(skips)
	return false
}

// RecordRolledBack tells the schedule that c, as it was before a rollback that
// committed, was rolled back: the digest it ran is not applied to it again
// until its tag moves on. Both the rollback action and the cloud's
// rollback_container report here.
func RecordRolledBack(c container.Container) {
	if c.ImageDigest == "" {
		// Built locally: the registry check never offers it an update anyway.
		return
	}
	log.Info().Str("container", c.Name).Str("digest", c.ImageDigest).Msg("auto update: skipping this image for the rolled back container until a newer one is pushed")
	recordUpdateSkip(containerSkipKey(c), c.ImageDigest)
}

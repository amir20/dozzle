package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
)

// SessionSecretFile is the name of the file the signing secret is persisted to,
// alongside users.yml.
const SessionSecretFile = "session_secret"

const sessionSecretSize = 32

// SessionSecret returns the random key session tokens are signed with, reading
// it from dir and creating it on first run.
//
// The signing key used to be derived from users.yml alone. That was fine while
// every user had a bcrypt hash, which carries its own salt, but an account can
// now be proven by OAuth instead and carry no password at all. In a deployment
// where nobody has one, every input to that digest is public — a role name, an
// email address, a GitHub login — so anyone who can guess them can derive the
// key and mint a session for any user. This file is what puts real entropy back
// in front of that.
//
// It fails soft. A read-only or unwritable /data is a real deployment (people
// mount users.yml read-only), and refusing to boot over it would break upgrades
// for installs that are not even at risk. An ephemeral secret is still
// unguessable; it only costs sessions on restart, which is why it warns.
func SessionSecret(dir string) []byte {
	path := filepath.Join(dir, SessionSecretFile)

	if secret, err := readSessionSecret(path); err == nil {
		return secret
	} else if !errors.Is(err, os.ErrNotExist) {
		log.Warn().Err(err).Str("path", path).Msg("Could not read the session secret; generating a new one")
	}

	secret := make([]byte, sessionSecretSize)
	if _, err := rand.Read(secret); err != nil {
		// crypto/rand failing means the process has no usable entropy source.
		// Continuing would sign sessions with zeros.
		log.Fatal().Err(err).Msg("Could not generate a session secret")
	}

	encoded := base64.StdEncoding.EncodeToString(secret) + "\n"

	// O_EXCL rather than a plain create: replicas sharing the same volume race
	// here on first boot, and the loser has to adopt the winner's secret instead
	// of overwriting it and invalidating the other's sessions.
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			if existing, readErr := readSessionSecret(path); readErr == nil {
				return existing
			}
		}

		log.Warn().Err(err).Str("path", path).Msg("Could not persist the session secret; sessions will not survive a restart")
		return secret
	}
	defer file.Close()

	if _, err := file.WriteString(encoded); err != nil {
		log.Warn().Err(err).Str("path", path).Msg("Could not write the session secret; sessions will not survive a restart")
		return secret
	}

	log.Info().Str("path", path).Msg("Generated a new session secret")

	return secret
}

func readSessionSecret(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	secret, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(data)))
	if err != nil {
		return nil, err
	}

	// A truncated or half-written file is worse than no file: it would sign
	// every session with a key that has almost no entropy.
	if len(secret) < sessionSecretSize {
		return nil, errors.New("session secret is too short")
	}

	return secret, nil
}

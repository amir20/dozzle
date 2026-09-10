package web

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"sync"

	"github.com/rs/zerolog/log"
)

// userRefSecretPath sits beside the other data files. It never leaves this
// machine: it exists so Cloud can tell two local users apart without being told
// who either of them is.
const userRefSecretPath = "./data/.user-ref-secret"

var (
	userRefOnce   sync.Once
	userRefSecret []byte
)

// loadUserRefSecret reads the secret, creating it on first use.
//
// It has to be stored rather than generated per process. A secret that changes
// on restart changes every reference with it, and everybody silently loses
// their chat history with no error anywhere — the one way this can fail without
// anyone noticing why.
func loadUserRefSecret() []byte {
	userRefOnce.Do(func() {
		if existing, err := os.ReadFile(userRefSecretPath); err == nil {
			if decoded, err := hex.DecodeString(string(existing)); err == nil && len(decoded) == 32 {
				userRefSecret = decoded
				return
			}
			log.Warn().Msg("user ref secret is unreadable, generating a new one; chat history will not resume")
		}

		secret := make([]byte, 32)
		if _, err := rand.Read(secret); err != nil {
			log.Error().Err(err).Msg("could not generate a user ref secret")
			return
		}

		if err := os.MkdirAll(filepath.Dir(userRefSecretPath), 0o700); err != nil {
			log.Warn().Err(err).Msg("could not create the data directory for the user ref secret")
		} else if err := os.WriteFile(userRefSecretPath, []byte(hex.EncodeToString(secret)), 0o600); err != nil {
			// Worth continuing: chat works this session, it just won't resume
			// after a restart.
			log.Warn().Err(err).Msg("could not persist the user ref secret; chat history will not survive a restart")
		}
		userRefSecret = secret
	})
	return userRefSecret
}

// userRef is the opaque handle Cloud partitions chat threads by.
//
// Deliberately not the username. Cloud has no business knowing who signs in to
// somebody's Dozzle, and a username is both identifying and unstable — proxy
// auth renames happen. This is stable for as long as the secret file is, which
// is exactly as long as the history should last.
func userRef(username string) string {
	if username == "" {
		return ""
	}
	secret := loadUserRefSecret()
	if len(secret) == 0 {
		return ""
	}
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(username))
	return hex.EncodeToString(mac.Sum(nil))[:32]
}

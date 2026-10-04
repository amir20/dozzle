package web

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"sync"
	"time"

	"github.com/amir20/dozzle/internal/auth"
)

const (
	cloudLinkStateTTL = 10 * time.Minute
	cloudLinkStateMax = 64
)

// cloudLinkStates holds the one-time states that tie a cloud callback to a link
// the same person started from this instance. The callback is a GET that a
// cross-site page can navigate to with the session cookie attached, so the
// cookie alone does not prove the person meant to link. A state does: it is
// minted by a POST whose response a cross-site page cannot read, and it is
// handed back only by Dozzle Cloud's link page.
type cloudLinkStates struct {
	mu     sync.Mutex
	states map[string]cloudLinkState
}

type cloudLinkState struct {
	owner   string
	expires time.Time
}

func (s *cloudLinkStates) issue(owner string) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	state := base64.RawURLEncoding.EncodeToString(b)
	now := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.states == nil {
		s.states = make(map[string]cloudLinkState)
	}
	// Minting needs no CSRF check because the caller never learns a forged
	// state, but a forged burst must not grow the map without bound. One pass
	// drops what expired and finds the oldest live state, which goes if the
	// map is still full. States are added one at a time, so one is enough.
	var oldest string
	var oldestExpires time.Time
	for k, v := range s.states {
		if now.After(v.expires) {
			delete(s.states, k)
		} else if oldest == "" || v.expires.Before(oldestExpires) {
			oldest, oldestExpires = k, v.expires
		}
	}
	if len(s.states) >= cloudLinkStateMax {
		delete(s.states, oldest)
	}
	s.states[state] = cloudLinkState{owner: owner, expires: now.Add(cloudLinkStateTTL)}
	return state, nil
}

// consume reports whether state was issued to owner and is still live. A state
// is spent on first use, valid or not.
func (s *cloudLinkStates) consume(state, owner string) bool {
	if state == "" {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.states[state]
	if !ok {
		return false
	}
	delete(s.states, state)
	return v.owner == owner && time.Now().Before(v.expires)
}

func cloudLinkOwner(r *http.Request) string {
	if user := auth.UserFromContext(r.Context()); user != nil {
		return user.Username
	}
	return ""
}

func (h *handler) startCloudLink(w http.ResponseWriter, r *http.Request) {
	state, err := h.cloudLinks.issue(cloudLinkOwner(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to start cloud link")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]string{"state": state})
}

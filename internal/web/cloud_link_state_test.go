package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/amir20/dozzle/internal/auth"
	"github.com/amir20/dozzle/internal/notification"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type cloudLinkRecorder struct {
	HostService
	linked *notification.CloudConfig
}

func (s *cloudLinkRecorder) SetCloudConfig(cc *notification.CloudConfig) { s.linked = cc }

// The callback is a GET a cross-site page can navigate to with the session
// cookie attached, so it links only when it carries back a state the same
// person minted here.
func Test_cloudCallback_requires_state_from_same_user(t *testing.T) {
	var exchanges atomic.Int32
	cloud := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		exchanges.Add(1)
		w.Write([]byte(`{"key":"attacker-key","prefix":"att"}`))
	}))
	defer cloud.Close()
	t.Setenv("DOLIGENCE_URL", cloud.URL)

	h := restrictedHandler(t)
	rec := &cloudLinkRecorder{HostService: h.hostService}
	h.hostService = rec

	as := func(name string, r *http.Request) *http.Request {
		return r.WithContext(auth.WithUser(context.Background(), auth.User{Username: name, Roles: auth.Cloud}))
	}
	mint := func(name string) string {
		rr := httptest.NewRecorder()
		h.startCloudLink(rr, as(name, httptest.NewRequest(http.MethodPost, "/api/cloud/link", nil)))
		require.Equal(t, http.StatusOK, rr.Code)
		var body struct{ State string }
		require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
		require.NotEmpty(t, body.State)
		return body.State
	}
	callback := func(name, query string) int {
		rr := httptest.NewRecorder()
		h.cloudCallback(rr, as(name, httptest.NewRequest(http.MethodGet, "/api/cloud/callback?"+query, nil)))
		return rr.Code
	}

	// The drive-by from the advisory: a token and nothing else.
	assert.Equal(t, http.StatusForbidden, callback("victim", "token=attacker"))
	assert.Equal(t, http.StatusForbidden, callback("victim", "token=attacker&state=guessed"))

	// A state minted by someone else does not link for the victim.
	assert.Equal(t, http.StatusForbidden, callback("victim", "token=attacker&state="+mint("attacker")))

	assert.Nil(t, rec.linked)
	assert.Zero(t, exchanges.Load(), "a refused callback must not reach the cloud")

	state := mint("victim")
	assert.Equal(t, http.StatusFound, callback("victim", "token=good&state="+state))
	require.NotNil(t, rec.linked)
	assert.Equal(t, "attacker-key", rec.linked.APIKey)

	// One use only.
	rec.linked = nil
	assert.Equal(t, http.StatusForbidden, callback("victim", "token=good&state="+state))
	assert.Nil(t, rec.linked)
}

func Test_cloudLinkStates_bounded(t *testing.T) {
	var s cloudLinkStates
	first, err := s.issue("a")
	require.NoError(t, err)
	for range cloudLinkStateMax {
		_, err := s.issue("a")
		require.NoError(t, err)
	}
	assert.Len(t, s.states, cloudLinkStateMax)
	assert.False(t, s.consume(first, "a"), "the oldest state is evicted once the store is full")
}

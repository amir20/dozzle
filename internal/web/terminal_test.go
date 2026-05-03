package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newUpgradeServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		conn.Close()
	}))
}

func dialUpgrade(t *testing.T, srv *httptest.Server, origin string) (*http.Response, error) {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	header := http.Header{}
	if origin != "" {
		header.Set("Origin", origin)
	}
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, header)
	if conn != nil {
		conn.Close()
	}
	return resp, err
}

func TestUpgrader_SameOriginSucceeds(t *testing.T) {
	srv := newUpgradeServer(t)
	defer srv.Close()

	resp, err := dialUpgrade(t, srv, srv.URL)
	require.NoError(t, err)
	if resp != nil {
		assert.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)
	}
}

func TestUpgrader_NoOriginSucceeds(t *testing.T) {
	// Non-browser clients (curl, scripts) don't send Origin — must still work.
	srv := newUpgradeServer(t)
	defer srv.Close()

	resp, err := dialUpgrade(t, srv, "")
	require.NoError(t, err)
	if resp != nil {
		assert.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)
	}
}

func TestUpgrader_CrossOriginRejected(t *testing.T) {
	srv := newUpgradeServer(t)
	defer srv.Close()

	resp, err := dialUpgrade(t, srv, "http://evil.example.com")
	assert.Error(t, err, "cross-origin upgrade must be rejected")
	if resp != nil {
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	}
}

func TestUpgrader_DifferentPortSameHostRejected(t *testing.T) {
	// Same-site different-origin (e.g. localhost:8888 vs localhost:9090) is
	// the realistic CSWSH vector. The default origin check rejects it because
	// host:port differs from the request's Host header.
	srv := newUpgradeServer(t)
	defer srv.Close()

	resp, err := dialUpgrade(t, srv, "http://localhost:1")
	assert.Error(t, err, "different-port same-host upgrade must be rejected")
	if resp != nil {
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	}
}

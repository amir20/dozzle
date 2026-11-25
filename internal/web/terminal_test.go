package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/coder/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newUpgradeServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrade(w, r)
		if err != nil {
			return
		}
		conn.CloseNow()
	}))
}

func dialUpgrade(t *testing.T, srv *httptest.Server, origin string) (*http.Response, error) {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	header := http.Header{}
	if origin != "" {
		header.Set("Origin", origin)
	}
	conn, resp, err := websocket.Dial(context.Background(), wsURL, &websocket.DialOptions{HTTPHeader: header})
	if conn != nil {
		conn.CloseNow()
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

// A DNS-rebound page reaches Dozzle under its own hostname with a matching
// Origin, so without auth the shell only answers on a name nobody can register.
func TestShellRefusesHostnameWithoutAuth(t *testing.T) {
	h := createHandler(nil, nil, Config{Base: "/", Mode: "server", EnableShell: true, Authorization: Authorization{Provider: NONE}})

	for _, route := range []string{"exec", "attach"} {
		path := "/api/hosts/localhost/containers/123/" + route
		for host, refused := range map[string]bool{
			"rebind.example:8080": true,
			"logs.example.com":    true,
			"dozzle.lan.example":  true,
			"dozzle.lan":          false,
			"NAS.LOCAL:8080":      false,
			"dozzle.home.arpa":    false,
			"dozzle.internal.":    false,
			"nas:8080":            false,
			"127.0.0.1:8080":      false,
			"192.168.1.10":        false,
			"[::1]:8080":          false,
			"localhost:8080":      false,
		} {
			req := httptest.NewRequest("GET", path, nil)
			req.Host = host
			req.Header.Set("Origin", "http://"+host)
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)
			if refused {
				assert.Equal(t, http.StatusForbidden, rr.Code, "%s on %s", route, host)
			} else {
				// Not a real websocket handshake, so the upgrade itself fails, but not with our 403.
				assert.NotEqual(t, http.StatusForbidden, rr.Code, "%s on %s", route, host)
			}
		}
	}
}

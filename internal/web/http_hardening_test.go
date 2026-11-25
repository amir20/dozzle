package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/amir20/dozzle/internal/container"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

// A cross-site page can POST text/plain or a form without a CORS preflight, so
// these routes only run on application/json. Nothing here looks at Origin or
// Host, so a proxy that rewrites Host is not affected.
func TestJSONOnlyWriteRoutes(t *testing.T) {
	setupTestEnv(t, true)
	started := false
	stubSelfUpdateStart(t, func(context.Context, string, func(container.UpdateProgress)) (bool, error) {
		started = true
		return false, nil
	})
	h := createHandler(nil, nil, Config{Base: "/", Mode: "server", EnableActions: true, Authorization: Authorization{Provider: NONE}})

	routes := []string{
		"/api/update/self",
		"/api/hosts/localhost/prune",
		"/api/swarm/services/web/scale",
		"/api/notifications/rules",
		"/api/notifications/dispatchers",
		"/api/notifications/preview",
		"/api/notifications/test-webhook",
		"/api/cloud/patterns",
		"/api/cloud/chat",
	}
	for _, path := range routes {
		for _, ct := range []string{"text/plain", "application/x-www-form-urlencoded", ""} {
			req := httptest.NewRequest("POST", path, strings.NewReader(`{"enabled":true}`))
			if ct != "" {
				req.Header.Set("Content-Type", ct)
			}
			// What a cross-site no-cors fetch carries; it changes nothing.
			req.Header.Set("Origin", "https://evil.example")
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)
			assert.Equal(t, http.StatusUnsupportedMediaType, rr.Code, "%s %q", path, ct)
		}
	}
	assert.False(t, started)

	// The UI behind a proxy that rewrites Host, and curl with -H, still get through.
	rr := doSetup(h, "POST", "/api/update/self", "{}", "Origin", "https://logs.example.com", "Host", "dozzle:8080")
	assert.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	assert.True(t, started)
}

// pprof dumps goroutine stacks, heap and the command line, so it needs a login
// and refuses a user confined by a filter.
func TestProfilerRequiresAuthentication(t *testing.T) {
	old := zerolog.GlobalLevel()
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	t.Cleanup(func() { zerolog.SetGlobalLevel(old) })

	h := createHandler(nil, nil, *proxyConfig())
	assert.Equal(t, http.StatusUnauthorized, doSetup(h, "GET", "/debug/pprof/", "").Code)
	assert.Equal(t, http.StatusForbidden, doSetup(h, "GET", "/debug/pprof/cmdline", "", filteredUser...).Code)
	assert.Equal(t, http.StatusOK, doSetup(h, "GET", "/debug/pprof/", "", unfiltered...).Code)
}

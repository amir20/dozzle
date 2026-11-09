package web

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
	"time"

	"github.com/amir20/dozzle/internal/auth"
	"github.com/amir20/dozzle/internal/hostservice"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// No Remote-Roles header means every role, the same as an empty roles key in
// users.yml, so the filter is all that separates these two.
var (
	filteredUser = []string{"Remote-User", "guest", "Remote-Filter", "label=com.example.app"}
	unfiltered   = []string{"Remote-User", "amir"}
)

func proxyConfig() *Config {
	return &Config{Base: "/", Mode: "server",
		Authorization: Authorization{
			Provider:   FORWARD_PROXY,
			Authorizer: auth.NewForwardProxyAuth("Remote-User", "Remote-Email", "Remote-Name", "Remote-Filter", "Remote-Roles"),
		},
		Setup: SetupConfig{StartedAt: time.Now()},
	}
}

func proxyAgentsHandler(hosts *fakeAgentHosts) *chi.Mux {
	manager := hostservice.NewRetriableClientManager(nil, nil, time.Second, tls.Certificate{})
	hosts.MultiHostService = hostservice.NewMultiHostService(manager, time.Second)
	return createRouter(&handler{
		hostService: hosts,
		content:     fstest.MapFS{"index.html": {Data: []byte("index page")}},
		config:      proxyConfig(),
	})
}

func TestSetup_FilteredUserCannotWrite(t *testing.T) {
	setupTestEnv(t, true)
	h := proxyAgentsHandler(&fakeAgentHosts{connected: map[string]string{}})

	writes := []struct{ method, path, body string }{
		{"POST", "/api/setup/agent-cert", ""},
		{"PATCH", "/api/setup/config", `{"enableShell":true}`},
		{"PATCH", "/api/setup/config", `{"autoPrune":{"mode":"daily","time":"03:00"}}`},
		{"POST", "/api/setup/restart", ""},
		{"POST", "/api/setup/agents", `{"address":"nas:7007"}`},
		{"DELETE", "/api/setup/agents", `{"endpoint":"nas:7007"}`},
	}
	for _, write := range writes {
		rr := doSetup(h, write.method, write.path, write.body, filteredUser...)
		assert.Equal(t, http.StatusForbidden, rr.Code, "%s %s %s", write.method, write.path, write.body)
		assert.NotContains(t, rr.Body.String(), "PRIVATE KEY")
	}

	// Writing the roles out changes nothing: the filter is what disqualifies.
	explicit := append([]string{"Remote-Roles", "all"}, filteredUser...)
	assert.Equal(t, http.StatusForbidden, doSetup(h, "POST", "/api/setup/agent-cert", "", explicit...).Code)

	rr := doSetup(h, "GET", "/api/setup", "", filteredUser...)
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	var state setupState
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &state))
	assert.False(t, state.CanWrite)
	assert.False(t, state.CanWriteUpdates)
}

func TestSetup_UnfilteredUserStillWrites(t *testing.T) {
	setupTestEnv(t, true)
	h := proxyAgentsHandler(&fakeAgentHosts{connected: map[string]string{}})

	rr := doSetup(h, "POST", "/api/setup/agent-cert", "", unfiltered...)
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	var pair setupAgentCertResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &pair))
	assert.Contains(t, pair.Key, "PRIVATE KEY")

	assert.Equal(t, http.StatusNoContent, doSetup(h, "PATCH", "/api/setup/config", `{"enableShell":true}`, unfiltered...).Code)
}

func Test_setupCanWrite_EveryLoginProvider(t *testing.T) {
	for _, provider := range []AuthProvider{SIMPLE, OIDC, FORWARD_PROXY} {
		h := &handler{config: &Config{Authorization: Authorization{Provider: provider}}}
		canWrite := func(user auth.User) bool {
			req := httptest.NewRequest(http.MethodPost, "/api/setup/agent-cert", nil)
			return h.setupCanWrite(req.WithContext(auth.WithUser(context.Background(), user)))
		}

		assert.True(t, canWrite(auth.User{Roles: auth.All}), string(provider))
		assert.False(t, canWrite(auth.User{Roles: auth.All, ContainerLabels: devLabels}), string(provider))
		assert.False(t, canWrite(auth.User{Roles: auth.Actions}), string(provider))
	}
}

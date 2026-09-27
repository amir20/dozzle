package web

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/config"
	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/hostservice"
	"github.com/go-chi/chi/v5"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

// fakeAgentHosts is a server-mode host service whose agents never dial out.
type fakeAgentHosts struct {
	*hostservice.MultiHostService
	connected map[string]string // endpoint -> host id
	addErr    error
	lastCert  *tls.Certificate
}

func (f *fakeAgentHosts) CanAddAgents() bool { return true }

func (f *fakeAgentHosts) AddAgent(_ context.Context, endpoint string, cert *tls.Certificate) (container.Host, error) {
	if f.addErr != nil {
		return container.Host{}, f.addErr
	}
	f.lastCert = cert
	f.connected[endpoint] = "id-" + endpoint
	return container.Host{ID: "id-" + endpoint, Name: "remote"}, nil
}

func (f *fakeAgentHosts) RemoveAgent(endpoint string) error {
	if _, ok := f.connected[endpoint]; !ok {
		return hostservice.ErrAgentNotFound
	}
	delete(f.connected, endpoint)
	return nil
}

func (f *fakeAgentHosts) AgentHostID(endpoint string) string { return f.connected[endpoint] }

func agentsHandler(hosts *fakeAgentHosts, setup SetupConfig) *chi.Mux {
	if setup.StartedAt.IsZero() {
		setup.StartedAt = time.Now()
	}
	manager := hostservice.NewRetriableClientManager(nil, nil, time.Second, tls.Certificate{})
	hosts.MultiHostService = hostservice.NewMultiHostService(manager, time.Second)
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "index.html", []byte("index page"), 0644)
	return createRouter(&handler{
		hostService: hosts,
		content:     afero.NewIOFS(fs),
		config:      &Config{Base: "/", Mode: "server", Authorization: Authorization{Provider: NONE}, Setup: setup},
	})
}

func readSetupState(t *testing.T, h http.Handler) setupState {
	t.Helper()
	rr := doSetup(h, "GET", "/api/setup", "")
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	var state setupState
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &state))
	return state
}

func TestSetupAgents_AddSavesAndLists(t *testing.T) {
	setupTestEnv(t, true)
	hosts := &fakeAgentHosts{connected: map[string]string{}}
	h := agentsHandler(hosts, SetupConfig{EnvAgents: []string{"env:7007"}})

	rr := doSetup(h, "POST", "/api/setup/agents", `{"address":" nas:7007 ","name":"nas"}`)
	require.Equal(t, http.StatusCreated, rr.Code, rr.Body.String())
	assert.JSONEq(t, `{"id":"id-nas:7007|nas","name":"remote","endpoint":"nas:7007|nas"}`, rr.Body.String())

	file, err := config.Load(setupConfigPath)
	require.NoError(t, err)
	assert.Equal(t, []string{"nas:7007|nas"}, file.RemoteAgents)

	state := readSetupState(t, h)
	assert.True(t, state.CanAddAgents)
	assert.Equal(t, []setupAgent{
		{Endpoint: "env:7007", Address: "env:7007", Locked: true},
		{Endpoint: "nas:7007|nas", Address: "nas:7007", Name: "nas", HostID: "id-nas:7007|nas"},
	}, state.Agents)

	// The same address again, under any name, is a duplicate. So is an env one.
	assert.Equal(t, http.StatusConflict, doSetup(h, "POST", "/api/setup/agents", `{"address":"nas:7007"}`).Code)
	assert.Equal(t, http.StatusConflict, doSetup(h, "POST", "/api/setup/agents", `{"address":"env:7007"}`).Code)
}

func TestSetupAgents_ConnectFailureSavesNothing(t *testing.T) {
	setupTestEnv(t, true)
	hosts := &fakeAgentHosts{connected: map[string]string{}, addErr: errors.New("dial tcp 10.0.0.9:7007: connect: connection refused")}
	h := agentsHandler(hosts, SetupConfig{})

	rr := doSetup(h, "POST", "/api/setup/agents", `{"address":"down:7007"}`)
	assert.Equal(t, http.StatusBadGateway, rr.Code)
	assert.Equal(t, "could not connect to agent: connection refused\n", rr.Body.String(), "a reason, never the raw error")
	assert.Equal(t, "unreachable", rr.Header().Get(agentErrorHeader))

	file, err := config.Load(setupConfigPath)
	require.NoError(t, err)
	assert.Empty(t, file.RemoteAgents)
}

func TestSetupAgents_RejectsBadInput(t *testing.T) {
	setupTestEnv(t, true)
	h := agentsHandler(&fakeAgentHosts{connected: map[string]string{}}, SetupConfig{})

	for _, body := range []string{
		`{"address":""}`,
		`{"address":"nas"}`,
		`{"address":"nas:0"}`,
		`{"address":"nas:99999"}`,
		`{"address":"tcp://nas:7007"}`,
		`{"address":"nas:7007|evil"}`,
		`{"address":"nas:7007","name":"a|b"}`,
	} {
		rr := doSetup(h, "POST", "/api/setup/agents", body)
		assert.Equal(t, http.StatusBadRequest, rr.Code, body)
		assert.Equal(t, "invalid", rr.Header().Get(agentErrorHeader), body)
	}
}

// The UI explains an error by its code, so each kind keeps one.
func TestSetupAgents_ErrorCodes(t *testing.T) {
	setupTestEnv(t, true)
	hosts := &fakeAgentHosts{connected: map[string]string{}}
	h := agentsHandler(hosts, SetupConfig{EnvAgents: []string{"env:7007"}})
	code := func(method, path, body string) string {
		return doSetup(h, method, path, body).Header().Get(agentErrorHeader)
	}

	require.Equal(t, http.StatusCreated, doSetup(h, "POST", "/api/setup/agents", `{"address":"nas:7007"}`).Code)
	assert.Equal(t, "exists", code("POST", "/api/setup/agents", `{"address":"nas:7007"}`))
	assert.Equal(t, "no-private-cert", code("POST", "/api/setup/agents", `{"address":"pi:7007","private":true}`))
	assert.Equal(t, "env-agent", code("DELETE", "/api/setup/agents", `{"endpoint":"env:7007"}`))
	assert.Equal(t, "not-found", code("DELETE", "/api/setup/agents", `{"endpoint":"other:7007"}`))

	hosts.addErr = hostservice.ErrDuplicateHost
	assert.Equal(t, "duplicate-host", code("POST", "/api/setup/agents", `{"address":"alias:7007"}`))
	hosts.addErr = errors.New("remote error: tls: unknown certificate authority")
	assert.Equal(t, "cert-mismatch", code("POST", "/api/setup/agents", `{"address":"alias:7007"}`))

	custom := agentsHandler(&fakeAgentHosts{connected: map[string]string{}}, SetupConfig{CustomCert: true})
	assert.Equal(t, "custom-cert", doSetup(custom, "POST", "/api/setup/agent-cert", "").Header().Get(agentErrorHeader))

	// Success and a forbidden request carry no code.
	assert.Empty(t, doSetup(h, "DELETE", "/api/setup/agents", `{"endpoint":"nas:7007"}`).Header().Get(agentErrorHeader))
}

// A hand edit can leave whitespace around an entry. Startup trims it before it
// dials, so the UI has to match the trimmed form or remove would report success
// while the agent stays connected.
func TestSetupAgents_RemoveHandEditedEntry(t *testing.T) {
	setupTestEnv(t, true)
	require.NoError(t, config.Update(setupConfigPath, func(c *config.File) {
		c.RemoteAgents = []string{" nas:7007 "}
	}))
	hosts := &fakeAgentHosts{connected: map[string]string{"nas:7007": "id-nas"}}
	h := agentsHandler(hosts, SetupConfig{})

	assert.Equal(t, http.StatusNoContent, doSetup(h, "DELETE", "/api/setup/agents", `{"endpoint":"nas:7007"}`).Code)
	file, err := config.Load(setupConfigPath)
	require.NoError(t, err)
	assert.Empty(t, file.RemoteAgents)
	assert.Empty(t, hosts.connected)
}

func TestSetupAgents_Remove(t *testing.T) {
	setupTestEnv(t, true)
	hosts := &fakeAgentHosts{connected: map[string]string{}}
	h := agentsHandler(hosts, SetupConfig{EnvAgents: []string{"env:7007"}})
	require.Equal(t, http.StatusCreated, doSetup(h, "POST", "/api/setup/agents", `{"address":"nas:7007"}`).Code)

	assert.Equal(t, http.StatusConflict, doSetup(h, "DELETE", "/api/setup/agents", `{"endpoint":"env:7007"}`).Code)
	assert.Equal(t, http.StatusNotFound, doSetup(h, "DELETE", "/api/setup/agents", `{"endpoint":"other:7007"}`).Code)
	assert.Equal(t, http.StatusNoContent, doSetup(h, "DELETE", "/api/setup/agents", `{"endpoint":"nas:7007"}`).Code)

	file, err := config.Load(setupConfigPath)
	require.NoError(t, err)
	assert.Empty(t, file.RemoteAgents)
	assert.Empty(t, hosts.connected)
}

// Same gate as every other setup write: no login and a closed window means no.
func TestSetupAgents_ClosedWindowForbidden(t *testing.T) {
	setupTestEnv(t, true)
	hosts := &fakeAgentHosts{connected: map[string]string{}}
	h := agentsHandler(hosts, SetupConfig{StartedAt: time.Now().Add(-time.Hour)})

	assert.Equal(t, http.StatusForbidden, doSetup(h, "POST", "/api/setup/agents", `{"address":"nas:7007"}`).Code)
	assert.Equal(t, http.StatusForbidden, doSetup(h, "DELETE", "/api/setup/agents", `{"endpoint":"nas:7007"}`).Code)
	assert.Empty(t, hosts.connected)
}

func TestSetupAgents_RefusedWithoutPersistedData(t *testing.T) {
	setupTestEnv(t, false)
	hosts := &fakeAgentHosts{connected: map[string]string{}}
	h := agentsHandler(hosts, SetupConfig{})

	rr := doSetup(h, "POST", "/api/setup/agents", `{"address":"nas:7007"}`)
	assert.Equal(t, http.StatusPreconditionFailed, rr.Code)
	assert.Equal(t, "not-persisted", rr.Header().Get(agentErrorHeader))
	assert.Empty(t, hosts.connected)
}

func TestSetupAgents_PrivateCert(t *testing.T) {
	setupTestEnv(t, true)
	hosts := &fakeAgentHosts{connected: map[string]string{}}
	h := agentsHandler(hosts, SetupConfig{})

	// No pair yet, so a private add has nothing to present.
	assert.Equal(t, http.StatusPreconditionFailed, doSetup(h, "POST", "/api/setup/agents", `{"address":"nas:7007","private":true}`).Code)

	rr := doSetup(h, "POST", "/api/setup/agent-cert", "")
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	var first setupAgentCertResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &first))
	assert.Contains(t, first.Cert, "BEGIN CERTIFICATE")
	assert.Contains(t, first.Key, "PRIVATE KEY")
	assert.Equal(t, "no-store", rr.Header().Get("Cache-Control"))

	rr = doSetup(h, "POST", "/api/setup/agent-cert", "")
	var second setupAgentCertResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &second))
	assert.Equal(t, first.Cert, second.Cert, "the pair is made once")

	require.Equal(t, http.StatusCreated, doSetup(h, "POST", "/api/setup/agents", `{"address":"nas:7007","private":true}`).Code)
	require.NotNil(t, hosts.lastCert)
	require.Equal(t, http.StatusCreated, doSetup(h, "POST", "/api/setup/agents", `{"address":"pi:7007"}`).Code)
	assert.Nil(t, hosts.lastCert)

	file, err := config.Load(setupConfigPath)
	require.NoError(t, err)
	assert.Equal(t, []string{"nas:7007"}, file.PrivateAgents)
	state := readSetupState(t, h)
	require.Len(t, state.Agents, 2)
	assert.True(t, state.Agents[0].Private)
	assert.False(t, state.Agents[1].Private)

	require.Equal(t, http.StatusNoContent, doSetup(h, "DELETE", "/api/setup/agents", `{"endpoint":"nas:7007"}`).Code)
	file, err = config.Load(setupConfigPath)
	require.NoError(t, err)
	assert.Empty(t, file.PrivateAgents)
}

// A hub with its own pair already hands it to agents; a second one would only confuse.
func TestSetupAgents_PrivateCertRefusedWithCustomCert(t *testing.T) {
	setupTestEnv(t, true)
	h := agentsHandler(&fakeAgentHosts{connected: map[string]string{}}, SetupConfig{CustomCert: true})
	assert.Equal(t, http.StatusConflict, doSetup(h, "POST", "/api/setup/agent-cert", "").Code)
}

func TestSetupAgents_PrivateCertForbiddenWhenWindowClosed(t *testing.T) {
	setupTestEnv(t, true)
	h := agentsHandler(&fakeAgentHosts{connected: map[string]string{}}, SetupConfig{StartedAt: time.Now().Add(-time.Hour)})
	assert.Equal(t, http.StatusForbidden, doSetup(h, "POST", "/api/setup/agent-cert", "").Code)
}

func TestDialFailureNamesOnlyTheKind(t *testing.T) {
	cases := map[string][2]string{
		`rpc error: code = Unavailable desc = connection error: desc = "error reading server preface: remote error: tls: unknown certificate authority"`: {"the agent refused this Dozzle's certificate", "host.add.cert"},
		"dial tcp: lookup nope on 127.0.0.11:53: no such host":                            {"no such host", "host.add.refused"},
		"dial tcp 10.0.0.9:7007: connect: connection refused":                             {"connection refused", "host.add.refused"},
		"context deadline exceeded":                                                       {"timed out", "host.add.timeout"},
		`connection error: desc = "error reading server preface: http2: frame too large"`: {"no Dozzle agent answered at that address", "host.add.refused"},
	}
	for raw, want := range cases {
		reason, outcome := dialFailure(errors.New(raw))
		assert.Equal(t, want, [2]string{reason, outcome}, raw)
	}
}

func TestSetupAgents_DialsAreRationed(t *testing.T) {
	setupTestEnv(t, true)
	old := agentDialLimiter
	agentDialLimiter = rate.NewLimiter(rate.Every(time.Hour), 2)
	t.Cleanup(func() { agentDialLimiter = old })

	hosts := &fakeAgentHosts{connected: map[string]string{}, addErr: errors.New("connection refused")}
	h := agentsHandler(hosts, SetupConfig{})
	for range 2 {
		assert.Equal(t, http.StatusBadGateway, doSetup(h, "POST", "/api/setup/agents", `{"address":"down:7007"}`).Code)
	}
	rr := doSetup(h, "POST", "/api/setup/agents", `{"address":"down:7007"}`)
	assert.Equal(t, http.StatusTooManyRequests, rr.Code)
	assert.Equal(t, "rate-limited", rr.Header().Get(agentErrorHeader))
}

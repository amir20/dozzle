package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/auth"
	"github.com/amir20/dozzle/internal/config"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSelfID = "4f1c9b2e8d7a6b5c4d3e2f1a0b9c8d7e6f5a4b3c2d1e0f9a8b7c6d5e4f3a2b1c"

// setupTestEnv points every setup seam at a temp dir and returns a channel
// that receives the id of each restart that would have happened.
func setupTestEnv(t *testing.T, persisted bool) (string, chan string) {
	t.Helper()
	dir := t.TempDir()
	restarts := make(chan string, 4)

	oldPath, oldPersisted, oldSelf, oldDelay, oldRestarter := setupConfigPath, setupPersisted, setupSelfID, setupRestartDelay, setupRestarter
	setupConfigPath = filepath.Join(dir, "dozzle.yml")
	setupPersisted = func() bool { return persisted }
	setupSelfID = func() string { return testSelfID }
	setupRestartDelay = 0
	setupRestarter = func(_ *handler, id string) error {
		restarts <- id
		return nil
	}
	t.Cleanup(func() {
		setupConfigPath, setupPersisted, setupSelfID, setupRestartDelay, setupRestarter = oldPath, oldPersisted, oldSelf, oldDelay, oldRestarter
	})
	return dir, restarts
}

func setupNoneHandler(startedAt time.Time, setup SetupConfig) *chi.Mux {
	setup.StartedAt = startedAt
	return createHandler(nil, nil, Config{Base: "/", Mode: "server", Authorization: Authorization{Provider: NONE}, Setup: setup})
}

func setupProxyHandler() *chi.Mux {
	return createHandler(nil, nil, Config{Base: "/", Mode: "server",
		Authorization: Authorization{
			Provider:   FORWARD_PROXY,
			Authorizer: auth.NewForwardProxyAuth("Remote-User", "Remote-Email", "Remote-Name", "Remote-Filter", "Remote-Roles"),
		},
		Setup: SetupConfig{StartedAt: time.Now()},
	})
}

func doSetup(h http.Handler, method, path, body string, headers ...string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

func TestSetup_AccountCreatedThenConflict(t *testing.T) {
	dir, _ := setupTestEnv(t, true)
	h := setupNoneHandler(time.Now(), SetupConfig{})

	rr := doSetup(h, "POST", "/api/setup/account", `{"username":"amir","password":"supersecret","email":"a@b.c"}`)
	require.Equal(t, http.StatusCreated, rr.Code, rr.Body.String())

	info, err := os.Stat(filepath.Join(dir, "users.yml"))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())

	db, err := auth.ReadUsersFromFile(filepath.Join(dir, "users.yml"))
	require.NoError(t, err)
	require.Contains(t, db.Users, "amir")
	assert.NotEqual(t, "supersecret", db.Users["amir"].Password)
	assert.Equal(t, "a@b.c", db.Users["amir"].Email)

	file, err := config.Load(setupConfigPath)
	require.NoError(t, err)
	require.NotNil(t, file.AuthProvider)
	assert.Equal(t, "simple", *file.AuthProvider)

	rr = doSetup(h, "POST", "/api/setup/account", `{"username":"other","password":"supersecret"}`)
	assert.Equal(t, http.StatusConflict, rr.Code)
}

func TestSetup_AccountConflictOnUsersYaml(t *testing.T) {
	dir, _ := setupTestEnv(t, true)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "users.yaml"), []byte("users: {}\n"), 0600))
	h := setupNoneHandler(time.Now(), SetupConfig{})

	rr := doSetup(h, "POST", "/api/setup/account", `{"username":"amir","password":"supersecret"}`)
	assert.Equal(t, http.StatusConflict, rr.Code)
}

func TestSetup_AccountValidation(t *testing.T) {
	setupTestEnv(t, true)
	h := setupNoneHandler(time.Now(), SetupConfig{})

	assert.Equal(t, http.StatusBadRequest, doSetup(h, "POST", "/api/setup/account", `{"username":"a b","password":"supersecret"}`).Code)
	assert.Equal(t, http.StatusBadRequest, doSetup(h, "POST", "/api/setup/account", `{"username":"../x","password":"supersecret"}`).Code)
	assert.Equal(t, http.StatusBadRequest, doSetup(h, "POST", "/api/setup/account", `{"username":"amir","password":"short"}`).Code)
}

func TestSetup_AccountNotPersisted(t *testing.T) {
	dir, _ := setupTestEnv(t, false)
	h := setupNoneHandler(time.Now(), SetupConfig{})

	rr := doSetup(h, "POST", "/api/setup/account", `{"username":"amir","password":"supersecret"}`)
	assert.Equal(t, http.StatusPreconditionFailed, rr.Code)
	_, err := os.Stat(filepath.Join(dir, "users.yml"))
	assert.True(t, os.IsNotExist(err))

	rr = doSetup(h, "POST", "/api/setup/auth", `{"provider":"forward-proxy"}`)
	assert.Equal(t, http.StatusPreconditionFailed, rr.Code)
}

func TestSetup_AuthLockedAndProviders(t *testing.T) {
	setupTestEnv(t, true)

	locked := setupNoneHandler(time.Now(), SetupConfig{LockedAuthProvider: true})
	assert.Equal(t, http.StatusConflict, doSetup(locked, "POST", "/api/setup/auth", `{"provider":"forward-proxy"}`).Code)
	assert.Equal(t, http.StatusConflict, doSetup(locked, "POST", "/api/setup/account", `{"username":"amir","password":"supersecret"}`).Code)

	h := setupNoneHandler(time.Now(), SetupConfig{})
	assert.Equal(t, http.StatusBadRequest, doSetup(h, "POST", "/api/setup/auth", `{"provider":"simple"}`).Code)
	assert.Equal(t, http.StatusNoContent, doSetup(h, "POST", "/api/setup/auth", `{"provider":"forward-proxy"}`).Code)

	file, err := config.Load(setupConfigPath)
	require.NoError(t, err)
	require.NotNil(t, file.AuthProvider)
	assert.Equal(t, "forward-proxy", *file.AuthProvider)
}

func TestSetup_PatchWindowClosed(t *testing.T) {
	setupTestEnv(t, true)
	h := setupNoneHandler(time.Now().Add(-time.Hour), SetupConfig{})

	rr := doSetup(h, "PATCH", "/api/setup/config", `{"enableActions":true}`)
	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestSetup_PatchLocked(t *testing.T) {
	setupTestEnv(t, true)
	h := setupNoneHandler(time.Now(), SetupConfig{LockedEnableShell: true})

	assert.Equal(t, http.StatusConflict, doSetup(h, "PATCH", "/api/setup/config", `{"enableShell":true}`).Code)
	assert.Equal(t, http.StatusNoContent, doSetup(h, "PATCH", "/api/setup/config", `{"enableActions":true}`).Code)

	rr := doSetup(h, "GET", "/api/setup", "")
	require.Equal(t, http.StatusOK, rr.Code)
	var state setupState
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &state))
	assert.True(t, state.WindowOpen)
	assert.True(t, state.CanWrite)
	assert.True(t, state.CanRestart)
	assert.True(t, state.Locked.EnableShell)
	require.NotNil(t, state.Pending.EnableActions)
	assert.True(t, *state.Pending.EnableActions)
	assert.Nil(t, state.Pending.EnableShell)
	assert.Nil(t, state.Pending.AuthProvider)
}

func TestSetup_LoginRoutesClosedAfterWindow(t *testing.T) {
	dir, _ := setupTestEnv(t, true)
	h := setupNoneHandler(time.Now().Add(-time.Hour), SetupConfig{})

	assert.Equal(t, http.StatusForbidden, doSetup(h, "POST", "/api/setup/account", `{"username":"amir","password":"supersecret"}`).Code)
	assert.Equal(t, http.StatusForbidden, doSetup(h, "POST", "/api/setup/auth", `{"provider":"forward-proxy"}`).Code)

	_, err := os.Stat(filepath.Join(dir, "users.yml"))
	assert.True(t, os.IsNotExist(err))
	file, err := config.Load(setupConfigPath)
	require.NoError(t, err)
	assert.Nil(t, file.AuthProvider)
}

func TestSetup_ConfigRefusedWithoutPersistedData(t *testing.T) {
	setupTestEnv(t, false)
	h := setupNoneHandler(time.Now(), SetupConfig{})

	assert.Equal(t, http.StatusPreconditionFailed, doSetup(h, "PATCH", "/api/setup/config", `{"enableActions":true}`).Code)

	file, err := config.Load(setupConfigPath)
	require.NoError(t, err)
	assert.Nil(t, file.EnableActions)
}

func TestSetup_RestartAllowedWithPendingAuthAfterWindow(t *testing.T) {
	_, restarts := setupTestEnv(t, true)
	started := time.Now()
	h := setupNoneHandler(started, SetupConfig{})

	// Login chosen while the window is open, restart applied after it closed.
	require.Equal(t, http.StatusCreated, doSetup(h, "POST", "/api/setup/account", `{"username":"amir","password":"supersecret"}`).Code)
	h = setupNoneHandler(started.Add(-time.Hour), SetupConfig{})
	rr := doSetup(h, "POST", "/api/setup/restart", "")
	require.Equal(t, http.StatusAccepted, rr.Code)

	select {
	case id := <-restarts:
		assert.Equal(t, testSelfID, id)
	case <-time.After(2 * time.Second):
		t.Fatal("restart was not triggered")
	}
}

func TestSetup_RestartForbiddenAfterWindowWithoutPendingAuth(t *testing.T) {
	setupTestEnv(t, true)
	enabled := true
	require.NoError(t, config.Update(setupConfigPath, func(c *config.File) { c.EnableActions = &enabled }))
	h := setupNoneHandler(time.Now().Add(-time.Hour), SetupConfig{})

	assert.Equal(t, http.StatusForbidden, doSetup(h, "POST", "/api/setup/restart", "").Code)
}

func TestSetup_RestartRequiresPendingChanges(t *testing.T) {
	_, restarts := setupTestEnv(t, true)
	h := setupNoneHandler(time.Now(), SetupConfig{})

	assert.Equal(t, http.StatusConflict, doSetup(h, "POST", "/api/setup/restart", "").Code)

	// A locked setting ignores the file, so it is not something to apply.
	provider := "simple"
	require.NoError(t, config.Update(setupConfigPath, func(c *config.File) { c.AuthProvider = &provider }))
	locked := setupNoneHandler(time.Now().Add(-time.Hour), SetupConfig{LockedAuthProvider: true})
	assert.Equal(t, http.StatusForbidden, doSetup(locked, "POST", "/api/setup/restart", "").Code)
	locked = setupNoneHandler(time.Now(), SetupConfig{LockedAuthProvider: true})
	assert.Equal(t, http.StatusConflict, doSetup(locked, "POST", "/api/setup/restart", "").Code)

	select {
	case <-restarts:
		t.Fatal("restart should not have been triggered")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestSetup_RestartCarriesWindowForward(t *testing.T) {
	_, restarts := setupTestEnv(t, true)
	started := time.Now().Add(-14 * time.Minute)
	h := setupNoneHandler(started, SetupConfig{})

	require.Equal(t, http.StatusNoContent, doSetup(h, "PATCH", "/api/setup/config", `{"enableActions":true}`).Code)
	require.Equal(t, http.StatusAccepted, doSetup(h, "POST", "/api/setup/restart", "").Code)
	<-restarts

	// The next process keeps the old start while it is still inside the window,
	// even though /data is no longer empty by then.
	assert.True(t, SetupWindowStart(time.Now(), false).Equal(started))
	// Once it has passed, a restart of an install with data never reopens it.
	later := started.Add(setupWindow + time.Minute)
	assert.True(t, SetupWindowStart(later, false).IsZero())
}

func TestSetup_WindowOnlyOnFreshInstall(t *testing.T) {
	setupTestEnv(t, true)
	now := time.Now()

	assert.True(t, SetupWindowStart(now, true).Equal(now), "a brand new install gets the window")
	assert.True(t, SetupWindowStart(now, false).IsZero(), "an install with earlier data does not")

	// An existing install that restarts with no login stays closed to anonymous writes.
	h := setupNoneHandler(SetupWindowStart(now, false), SetupConfig{})
	assert.Equal(t, http.StatusForbidden, doSetup(h, "PATCH", "/api/setup/config", `{"enableActions":true}`).Code)
	assert.Equal(t, http.StatusForbidden, doSetup(h, "POST", "/api/setup/account", `{"username":"amir","password":"supersecret"}`).Code)

	rr := doSetup(h, "GET", "/api/setup", "")
	require.Equal(t, http.StatusOK, rr.Code)
	var state setupState
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &state))
	assert.False(t, state.WindowOpen)
	assert.False(t, state.CanWrite)
}

func TestSetup_AccountRollsBackUsersFileWhenConfigFails(t *testing.T) {
	dir, _ := setupTestEnv(t, true)
	require.NoError(t, os.WriteFile(setupConfigPath, []byte("authProvider: [not a string\n"), 0600))
	h := setupNoneHandler(time.Now(), SetupConfig{})

	rr := doSetup(h, "POST", "/api/setup/account", `{"username":"amir","password":"supersecret"}`)
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	_, err := os.Stat(filepath.Join(dir, "users.yml"))
	assert.True(t, os.IsNotExist(err))
}

func TestSetup_RestartWithoutSelfID(t *testing.T) {
	setupTestEnv(t, true)
	setupSelfID = func() string { return "" }
	h := setupNoneHandler(time.Now(), SetupConfig{})
	enabled := true
	require.NoError(t, config.Update(setupConfigPath, func(c *config.File) { c.EnableActions = &enabled }))

	assert.Equal(t, http.StatusConflict, doSetup(h, "POST", "/api/setup/restart", "").Code)
}

func TestSetup_ProxyRoles(t *testing.T) {
	setupTestEnv(t, true)
	h := setupProxyHandler()

	limited := []string{"Remote-User", "bob", "Remote-Roles", "actions"}
	admin := []string{"Remote-User", "amir", "Remote-Roles", "all"}

	assert.Equal(t, http.StatusForbidden, doSetup(h, "PATCH", "/api/setup/config", `{"enableActions":true}`, limited...).Code)
	assert.Equal(t, http.StatusForbidden, doSetup(h, "POST", "/api/setup/restart", "", limited...).Code)
	assert.Equal(t, http.StatusNoContent, doSetup(h, "PATCH", "/api/setup/config", `{"enableActions":true}`, admin...).Code)
	assert.Equal(t, http.StatusAccepted, doSetup(h, "POST", "/api/setup/restart", "", admin...).Code)

	// Login routes do not exist once a provider is on.
	assert.NotEqual(t, http.StatusCreated, doSetup(h, "POST", "/api/setup/account", `{"username":"x","password":"supersecret"}`, admin...).Code)
	assert.Equal(t, http.StatusUnauthorized, doSetup(h, "GET", "/api/setup", "").Code)
}

func TestSetup_NotRegisteredOutsideServerMode(t *testing.T) {
	setupTestEnv(t, true)
	h := createHandler(nil, nil, Config{Base: "/", Mode: "swarm", Authorization: Authorization{Provider: NONE}, Setup: SetupConfig{StartedAt: time.Now()}})

	rr := doSetup(h, "GET", "/api/setup", "")
	assert.NotContains(t, rr.Header().Get("Content-Type"), "application/json")
	assert.NotEqual(t, http.StatusNoContent, doSetup(h, "PATCH", "/api/setup/config", `{}`).Code)
	assert.NotEqual(t, http.StatusAccepted, doSetup(h, "POST", "/api/setup/restart", "").Code)
}

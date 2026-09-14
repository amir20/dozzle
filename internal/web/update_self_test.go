package web

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/auth"
	"github.com/amir20/dozzle/internal/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func stubSelfUpdateStart(t *testing.T, fn func(ctx context.Context, id string, progress func(container.UpdateProgress)) (bool, error)) {
	t.Helper()
	old := selfUpdateStart
	selfUpdateStart = fn
	t.Cleanup(func() { selfUpdateStart = old })
}

func updateSelfEvents(body string) []string {
	var statuses []string
	for line := range strings.SplitSeq(body, "\n") {
		if rest, ok := strings.CutPrefix(line, "data: "); ok {
			_, status, _ := strings.Cut(rest, `"status":"`)
			status, _, _ = strings.Cut(status, `"`)
			statuses = append(statuses, status)
		}
	}
	return statuses
}

func actionsHandler() http.Handler {
	return createHandler(nil, nil, Config{Base: "/", Mode: "server", EnableActions: true, Authorization: Authorization{Provider: NONE}, Setup: SetupConfig{StartedAt: time.Now()}})
}

func TestUpdateSelf_StreamsProgress(t *testing.T) {
	setupTestEnv(t, true)
	stubSelfUpdateStart(t, func(_ context.Context, id string, progress func(container.UpdateProgress)) (bool, error) {
		assert.Equal(t, testSelfID, id)
		progress(container.UpdateProgress{Status: "pulling", Layer: "abc", Current: 1, Total: 2})
		progress(container.UpdateProgress{Status: "recreating"})
		return true, nil
	})

	rr := doSetup(actionsHandler(), "POST", "/api/update/self", "")
	require.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Header().Get("Content-Type"), "text/event-stream")
	assert.Contains(t, rr.Body.String(), "event: update-progress")
	assert.Equal(t, []string{"pulling", "recreating", "done"}, updateSelfEvents(rr.Body.String()))
}

func TestUpdateSelf_UpToDateAndError(t *testing.T) {
	setupTestEnv(t, true)
	stubSelfUpdateStart(t, func(context.Context, string, func(container.UpdateProgress)) (bool, error) { return false, nil })
	assert.Equal(t, []string{"up-to-date"}, updateSelfEvents(doSetup(actionsHandler(), "POST", "/api/update/self", "").Body.String()))

	stubSelfUpdateStart(t, func(_ context.Context, _ string, progress func(container.UpdateProgress)) (bool, error) {
		progress(container.UpdateProgress{Status: "error", Error: "auto-remove"})
		return false, errors.New("auto-remove")
	})
	assert.Equal(t, []string{"error"}, updateSelfEvents(doSetup(actionsHandler(), "POST", "/api/update/self", "").Body.String()))

	stubSelfUpdateStart(t, func(context.Context, string, func(container.UpdateProgress)) (bool, error) {
		return false, errors.New("boom")
	})
	body := doSetup(actionsHandler(), "POST", "/api/update/self", "").Body.String()
	assert.Equal(t, []string{"error"}, updateSelfEvents(body))
	assert.Contains(t, body, "boom")
}

func TestUpdateSelf_NoContainer(t *testing.T) {
	setupTestEnv(t, true)
	setupSelfID = func() string { return "" }
	stubSelfUpdateStart(t, func(context.Context, string, func(container.UpdateProgress)) (bool, error) {
		t.Fatal("start should not be called")
		return false, nil
	})
	assert.Equal(t, []string{"error"}, updateSelfEvents(doSetup(actionsHandler(), "POST", "/api/update/self", "").Body.String()))
}

func TestUpdateSelf_Busy(t *testing.T) {
	setupTestEnv(t, true)
	selfUpdateMu.Lock()
	defer selfUpdateMu.Unlock()
	body := doSetup(actionsHandler(), "POST", "/api/update/self", "").Body.String()
	assert.Equal(t, []string{"error"}, updateSelfEvents(body))
	assert.Contains(t, body, "already in progress")
}

func TestUpdateSelf_Registration(t *testing.T) {
	setupTestEnv(t, true)
	stubSelfUpdateStart(t, func(context.Context, string, func(container.UpdateProgress)) (bool, error) { return false, nil })

	noActions := createHandler(nil, nil, Config{Base: "/", Mode: "server", Authorization: Authorization{Provider: NONE}})
	assert.NotContains(t, doSetup(noActions, "POST", "/api/update/self", "").Header().Get("Content-Type"), "text/event-stream")

	swarm := createHandler(nil, nil, Config{Base: "/", Mode: "swarm", EnableActions: true, Authorization: Authorization{Provider: NONE}})
	assert.NotContains(t, doSetup(swarm, "POST", "/api/update/self", "").Header().Get("Content-Type"), "text/event-stream")
}

func TestUpdateSelf_RequiresActionsRole(t *testing.T) {
	setupTestEnv(t, true)
	stubSelfUpdateStart(t, func(context.Context, string, func(container.UpdateProgress)) (bool, error) { return false, nil })
	proxy := createHandler(nil, nil, Config{Base: "/", Mode: "server", EnableActions: true,
		Authorization: Authorization{
			Provider:   FORWARD_PROXY,
			Authorizer: auth.NewForwardProxyAuth("Remote-User", "Remote-Email", "Remote-Name", "Remote-Filter", "Remote-Roles"),
		},
		Setup: SetupConfig{StartedAt: time.Now()},
	})

	assert.Equal(t, http.StatusForbidden, doSetup(proxy, "POST", "/api/update/self", "", "Remote-User", "bob", "Remote-Roles", "shell").Code)
	rr := doSetup(proxy, "POST", "/api/update/self", "", "Remote-User", "amir", "Remote-Roles", "actions")
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, []string{"up-to-date"}, updateSelfEvents(rr.Body.String()))
}

package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/amir20/dozzle/internal/auth"
	"github.com/amir20/dozzle/internal/cloud"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The exact shape useViewContext() produces and cloudChat.ts posts. visibleAt
// is an ISO string because that is what the browser has; sending it as one used
// to fail the decode and answer 400 for every turn.
const realChatBody = `{"message":"why is this crashing","view":{` +
	`"kind":"container","target":"dev123",` +
	`"containers":[{"id":"dev123","name":"dev","host":"localhost"}],` +
	`"hosts":["localhost"],"search":"connection refused","levels":["warn","error"],` +
	`"visibleAt":"2026-09-09T14:02:44.123Z","historical":false}}`

func chatHandler(t *testing.T, chat func(context.Context, string, cloud.ViewContext, string, cloud.Principal, func(cloud.ChatEvent)) error) *handler {
	t.Helper()
	// userRef writes a secret next to the other data files; keep it in a temp
	// dir rather than the repo.
	t.Chdir(t.TempDir())
	userRefOnce = sync.Once{}
	userRefSecret = nil

	h := restrictedHandler(t)
	h.config.Cloud = CloudHooks{Chat: chat}
	return h
}

func postChat(h *handler, body string, user auth.User) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/cloud/chat", strings.NewReader(body))
	req = req.WithContext(auth.WithUser(req.Context(), user))
	rr := httptest.NewRecorder()
	h.cloudChat(rr, req)
	return rr
}

func Test_cloudChat_acceptsWhatTheBrowserSends(t *testing.T) {
	var got cloud.ViewContext
	h := chatHandler(t, func(_ context.Context, message string, view cloud.ViewContext, _ string, _ cloud.Principal, emit func(cloud.ChatEvent)) error {
		got = view
		emit(cloud.ChatEvent{Kind: "delta", Text: "postgres restarted"})
		emit(cloud.ChatEvent{Kind: "done"})
		return nil
	})

	rr := postChat(h, realChatBody, auth.User{Username: "amir", Roles: auth.All})

	require.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Header().Get("Content-Type"), "text/event-stream")
	assert.Contains(t, rr.Body.String(), `"kind":"delta"`)
	assert.Contains(t, rr.Body.String(), "postgres restarted")
	assert.Equal(t, "2026-09-09T14:02:44.123Z", got.VisibleAt)
	assert.Equal(t, "connection refused", got.Search)
	assert.Equal(t, []string{"warn", "error"}, got.Levels)
}

func Test_cloudChat_rejectsAnEmptyMessage(t *testing.T) {
	h := chatHandler(t, func(context.Context, string, cloud.ViewContext, string, cloud.Principal, func(cloud.ChatEvent)) error {
		t.Fatal("cloud should not be called for an empty message")
		return nil
	})

	rr := postChat(h, `{"message":"","view":{"kind":"index"}}`, auth.User{Username: "amir", Roles: auth.All})
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

// The turn runs as the person who asked. This is the whole point of opening the
// stream inside the request rather than reusing the shared ToolStream.
func Test_cloudChat_runsAsTheAskingUser(t *testing.T) {
	var got cloud.Principal
	h := chatHandler(t, func(_ context.Context, _ string, _ cloud.ViewContext, _ string, p cloud.Principal, _ func(cloud.ChatEvent)) error {
		got = p
		return nil
	})

	postChat(h, realChatBody, auth.User{Username: "amir", Roles: auth.Download, ContainerLabels: devLabels})

	assert.Equal(t, cloud.PrincipalUser, got.Kind)
	assert.Equal(t, devLabels, got.Labels)
	assert.Equal(t, auth.Download, got.Roles)
}

// A filtered user must not be able to put a container they cannot see into the
// context and have the assistant discuss it.
func Test_cloudChat_scopesTheClaimedView(t *testing.T) {
	var got cloud.ViewContext
	h := chatHandler(t, func(_ context.Context, _ string, view cloud.ViewContext, _ string, _ cloud.Principal, _ func(cloud.ChatEvent)) error {
		got = view
		return nil
	})

	body := `{"message":"what is wrong","view":{"kind":"index","containers":[` +
		`{"id":"dev123","name":"dev","host":"localhost"},` +
		`{"id":"prod456","name":"prod","host":"localhost"}]}}`
	postChat(h, body, auth.User{Username: "amir", Roles: auth.All, ContainerLabels: devLabels})

	require.Len(t, got.Containers, 1)
	assert.Equal(t, "dev123", got.Containers[0].ID)
}

// Cloud partitions threads by this, so it has to be stable and it must never be
// the username.
func Test_cloudChat_sendsAnOpaqueUserRef(t *testing.T) {
	var refs []string
	h := chatHandler(t, func(_ context.Context, _ string, _ cloud.ViewContext, ref string, _ cloud.Principal, _ func(cloud.ChatEvent)) error {
		refs = append(refs, ref)
		return nil
	})

	postChat(h, realChatBody, auth.User{Username: "amir", Roles: auth.All})
	postChat(h, realChatBody, auth.User{Username: "amir", Roles: auth.All})
	postChat(h, realChatBody, auth.User{Username: "someone-else", Roles: auth.All})

	require.Len(t, refs, 3)
	assert.NotEmpty(t, refs[0])
	assert.NotContains(t, refs[0], "amir")
	assert.Equal(t, refs[0], refs[1])
	assert.NotEqual(t, refs[0], refs[2])
}

package dispatcher

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWebhookDispatcher_KeepsPlaceholder(t *testing.T) {
	t.Setenv("SLACK_WEBHOOK", "https://hooks.slack.com/services/T/B/secret")

	w, err := NewWebhookDispatcher("t", "${SLACK_WEBHOOK}", "", map[string]string{"Authorization": "Bearer ${SLACK_TOKEN}"})
	require.NoError(t, err)
	assert.Equal(t, "${SLACK_WEBHOOK}", w.URL)
	assert.Equal(t, "Bearer ${SLACK_TOKEN}", w.Headers["Authorization"])
}

func TestNewWebhookDispatcher_RejectsBlockedEnv(t *testing.T) {
	cases := []struct {
		url     string
		headers map[string]string
	}{
		{url: "https://example.com/?k=${DOZZLE_AUTH_PROVIDER}"},
		{url: "https://example.com/?k=${aws_secret_access_key}"},
		{url: "https://example.com/", headers: map[string]string{"X-Key": "${DOCKER_CERT_PATH}"}},
	}
	for _, c := range cases {
		_, err := NewWebhookDispatcher("t", c.url, "", c.headers)
		assert.Error(t, err, "%q / %v should be rejected", c.url, c.headers)
	}
}

func TestNewWebhookDispatcher_AllowsUnsetEnv(t *testing.T) {
	// The variable may only exist on the agent that sends.
	_, err := NewWebhookDispatcher("t", "${NOT_SET_ON_THIS_HOST}", "", nil)
	assert.NoError(t, err)
}

func TestNewWebhookDispatcher_ValidatesExpandedScheme(t *testing.T) {
	t.Setenv("HOOK_URL", "file:///etc/passwd")
	_, err := NewWebhookDispatcher("t", "${HOOK_URL}", "", nil)
	assert.Error(t, err)
}

func TestSendTest_ExpandsURLAndHeaders(t *testing.T) {
	var gotPath, gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		rw.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	t.Setenv("HOOK_URL", srv.URL+"/services/secret")
	t.Setenv("HOOK_TOKEN", "abc")

	w, err := NewWebhookDispatcher("t", "${HOOK_URL}", "", map[string]string{"Authorization": "Bearer ${HOOK_TOKEN}"})
	require.NoError(t, err)
	w.client = &http.Client{Timeout: 5 * time.Second}

	result := w.SendTest(context.Background(), newTestNotification("x"))
	assert.True(t, result.Success, result.Error)
	assert.Equal(t, "/services/secret", gotPath)
	assert.Equal(t, "Bearer abc", gotAuth)
}

func TestSendTest_UnsetEnvFails(t *testing.T) {
	w, err := NewWebhookDispatcher("t", "${NOT_SET_ON_THIS_HOST}", "", nil)
	require.NoError(t, err)

	result := w.SendTest(context.Background(), newTestNotification("x"))
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "NOT_SET_ON_THIS_HOST is not set")
}

func TestSendTest_RedactsURLInError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {}))
	addr := srv.URL
	srv.Close() // nothing listening, so the request fails with a *url.Error

	w, err := NewWebhookDispatcher("t", addr+"/services/T/B/supersecret", "", nil)
	require.NoError(t, err)
	w.client = &http.Client{Timeout: 5 * time.Second}

	result := w.SendTest(context.Background(), newTestNotification("x"))
	assert.False(t, result.Success)
	assert.NotContains(t, result.Error, "supersecret")
	assert.True(t, strings.Contains(result.Error, "/[redacted]"), result.Error)
}

func TestExpandEnv_LeavesBareDollarAlone(t *testing.T) {
	t.Setenv("FOO", "bar")
	out, err := expandEnv("https://example.com/$FOO?x=${FOO}")
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/$FOO?x=bar", out)
}

func TestExpandedURLErrorsDoNotLeakValue(t *testing.T) {
	cases := map[string]string{
		"http://[${HOOK_SECRET}":  "hunter2",
		"${HOOK_SECRET}":          "tok3n:abc",
		"http://x:${HOOK_SECRET}": "notaport",
	}
	for raw, value := range cases {
		t.Setenv("HOOK_SECRET", value)

		_, err := NewWebhookDispatcher("t", raw, "", nil)
		require.Error(t, err, raw)
		assert.NotContains(t, err.Error(), value, raw)
		assert.NotContains(t, err.Error(), strings.SplitN(value, ":", 2)[0], raw)
	}

	// Same check at send time, when the value changes after the dispatcher was built.
	t.Setenv("HOOK_SECRET", "example.com")
	w, err := NewWebhookDispatcher("t", "https://${HOOK_SECRET}/hook", "", nil)
	require.NoError(t, err)
	t.Setenv("HOOK_SECRET", "[hunter2")
	result := w.SendTest(context.Background(), newTestNotification("x"))
	assert.False(t, result.Success)
	assert.NotContains(t, result.Error, "hunter2")
}

func TestSendTest_ScrubsEnvValueFromHostErrors(t *testing.T) {
	t.Setenv("HOOK_SECRET", "hunter2")
	w, err := NewWebhookDispatcher("t", "http://${HOOK_SECRET}.invalid/hook", "", nil)
	require.NoError(t, err)

	result := w.SendTest(context.Background(), newTestNotification("x"))
	assert.False(t, result.Success)
	assert.NotContains(t, result.Error, "hunter2")
}

func TestSendTest_EnvURLErrorsNameNoPartOfValue(t *testing.T) {
	for _, value := range []string{
		"hunter2.invalid/rest-of-token",
		"user:pw@hunter2.invalid",
		"%68unter2.invalid",
	} {
		t.Setenv("HOOK_SECRET", value)
		w, err := NewWebhookDispatcher("t", "http://${HOOK_SECRET}", "", nil)
		if err != nil {
			// Rejected up front (url.Parse refuses %68 in a host); the message must not leak either.
			assert.NotContains(t, err.Error(), "hunter2", value)
			continue
		}

		result := w.SendTest(context.Background(), newTestNotification("x"))
		assert.False(t, result.Success)
		assert.NotContains(t, result.Error, "hunter2", value)
		assert.Contains(t, result.Error, "${HOOK_SECRET}", value)
	}
}

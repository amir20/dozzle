package dispatcher

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/amir20/dozzle/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestNotification(detail string) types.Notification {
	return types.Notification{
		ID:     "test-123",
		Type:   types.LogNotification,
		Detail: detail,
		Container: types.NotificationContainer{
			ID:       "abc123",
			Name:     "my-container",
			Image:    "nginx:latest",
			HostName: "docker-host",
		},
		Log: &types.NotificationLog{
			Message:   detail,
			Level:     "info",
			Stream:    "stdout",
			Timestamp: time.Now().Unix(),
		},
		Timestamp: time.Now(),
	}
}

func TestExecuteJSONTemplate_EscapesQuotes(t *testing.T) {
	templateText := `{"message": "{{ .Detail }}"}`
	notification := newTestNotification(`Server started {"service":"scoutarr","port":5839}`)

	payload, err := executeJSONTemplate(templateText, notification)
	require.NoError(t, err)

	var result map[string]any
	err = json.Unmarshal(payload, &result)
	require.NoError(t, err, "output should be valid JSON, got: %s", string(payload))
	assert.Equal(t, `Server started {"service":"scoutarr","port":5839}`, result["message"])
}

func TestExecuteJSONTemplate_EscapesNewlines(t *testing.T) {
	templateText := `{"message": "{{ .Detail }}"}`
	notification := newTestNotification("line1\nline2\nline3")

	payload, err := executeJSONTemplate(templateText, notification)
	require.NoError(t, err)

	var result map[string]any
	err = json.Unmarshal(payload, &result)
	require.NoError(t, err, "output should be valid JSON, got: %s", string(payload))
	assert.Equal(t, "line1\nline2\nline3", result["message"])
}

func TestExecuteJSONTemplate_EscapesBackslashes(t *testing.T) {
	templateText := `{"message": "{{ .Detail }}"}`
	notification := newTestNotification(`path\to\file`)

	payload, err := executeJSONTemplate(templateText, notification)
	require.NoError(t, err)

	var result map[string]any
	err = json.Unmarshal(payload, &result)
	require.NoError(t, err, "output should be valid JSON, got: %s", string(payload))
	assert.Equal(t, `path\to\file`, result["message"])
}

func TestExecuteJSONTemplate_MultiplePlaceholders(t *testing.T) {
	templateText := `{"title": "{{ .Container.Name }}", "description": "{{ .Detail }}"}`
	notification := newTestNotification(`error: "something" broke`)

	payload, err := executeJSONTemplate(templateText, notification)
	require.NoError(t, err)

	var result map[string]any
	err = json.Unmarshal(payload, &result)
	require.NoError(t, err, "output should be valid JSON, got: %s", string(payload))
	assert.Equal(t, "my-container", result["title"])
	assert.Equal(t, `error: "something" broke`, result["description"])
}

func TestExecuteJSONTemplate_NestedObjects(t *testing.T) {
	templateText := `{
		"embeds": [
			{
				"title": "{{ .Container.Name }}",
				"description": "{{ .Detail }}",
				"fields": [
					{"name": "Host", "value": "{{ .Container.HostName }}"}
				]
			}
		]
	}`
	notification := newTestNotification(`log with "quotes" and {braces}`)

	payload, err := executeJSONTemplate(templateText, notification)
	require.NoError(t, err)

	var result map[string]any
	err = json.Unmarshal(payload, &result)
	require.NoError(t, err, "output should be valid JSON, got: %s", string(payload))

	embeds := result["embeds"].([]any)
	embed := embeds[0].(map[string]any)
	assert.Equal(t, "my-container", embed["title"])
	assert.Equal(t, `log with "quotes" and {braces}`, embed["description"])

	fields := embed["fields"].([]any)
	field := fields[0].(map[string]any)
	assert.Equal(t, "docker-host", field["value"])
}

func TestExecuteJSONTemplate_StaticStringsUnchanged(t *testing.T) {
	templateText := `{"type": "section", "text": "{{ .Detail }}"}`
	notification := newTestNotification("hello")

	payload, err := executeJSONTemplate(templateText, notification)
	require.NoError(t, err)

	var result map[string]any
	err = json.Unmarshal(payload, &result)
	require.NoError(t, err)
	assert.Equal(t, "section", result["type"])
	assert.Equal(t, "hello", result["text"])
}

func TestExecuteJSONTemplate_ConcatenatedPlaceholders(t *testing.T) {
	templateText := `{"text": "{{ .Container.Name }}: {{ .Detail }}"}`
	notification := newTestNotification(`"critical" failure`)

	payload, err := executeJSONTemplate(templateText, notification)
	require.NoError(t, err)

	var result map[string]any
	err = json.Unmarshal(payload, &result)
	require.NoError(t, err, "output should be valid JSON, got: %s", string(payload))
	assert.Equal(t, `my-container: "critical" failure`, result["text"])
}

func TestExecuteJSONTemplate_InvalidJSONFallsBackToTextTemplate(t *testing.T) {
	// Not valid JSON, but valid Go template — should fall back to raw execution
	templateText := `{{ .Container.Name }}: {{ .Detail }}`
	notification := newTestNotification("some log")

	payload, err := executeJSONTemplate(templateText, notification)
	require.NoError(t, err)
	assert.Equal(t, "my-container: some log", string(payload))
}

func TestNewWebhookDispatcher_RejectsNonHTTPSchemes(t *testing.T) {
	cases := []string{
		"file:///etc/passwd",
		"gopher://example.com/",
		"ftp://example.com/",
		"javascript:alert(1)",
	}
	for _, raw := range cases {
		_, err := NewWebhookDispatcher("t", raw, "", nil)
		assert.Error(t, err, "scheme %q should be rejected", raw)
	}
}

func TestNewWebhookDispatcher_AcceptsHTTPAndHTTPS(t *testing.T) {
	for _, raw := range []string{"http://example.com/hook", "https://example.com/hook", "HTTP://example.com/hook"} {
		_, err := NewWebhookDispatcher("t", raw, "", nil)
		assert.NoError(t, err, "scheme in %q should be allowed", raw)
	}
}

func TestSendTest_RejectsLoopbackTarget(t *testing.T) {
	w, err := NewWebhookDispatcher("t", "http://127.0.0.1:1/hook", "", nil)
	require.NoError(t, err)

	result := w.SendTest(context.Background(), newTestNotification("x"))
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "blocked address range")
}

func TestSendTest_RejectsLinkLocalTarget(t *testing.T) {
	w, err := NewWebhookDispatcher("t", "http://169.254.169.254/latest/meta-data/", "", nil)
	require.NoError(t, err)

	result := w.SendTest(context.Background(), newTestNotification("x"))
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "blocked address range")
}

// TestSendTest_DoesNotReflectResponseBody confirms that an attacker controlling
// a webhook target that returns non-2xx with sensitive body content cannot
// recover that body through the TestResult.Error field.
func TestSendTest_DoesNotReflectResponseBody(t *testing.T) {
	const secret = "SUPER_SECRET_TOKEN_abcdef123"
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusUnauthorized)
		rw.Write([]byte(secret))
	}))
	defer srv.Close()

	w, err := NewWebhookDispatcher("t", srv.URL, "", nil)
	require.NoError(t, err)
	// Allow loopback for this test only by swapping in a default transport.
	w.client = &http.Client{Timeout: 5 * time.Second}

	result := w.SendTest(context.Background(), newTestNotification("x"))
	assert.False(t, result.Success)
	require.NotNil(t, result.StatusCode)
	assert.Equal(t, http.StatusUnauthorized, result.StatusCode)
	assert.NotContains(t, result.Error, secret, "response body must not leak into Error")
}

func TestIsBlockedIP(t *testing.T) {
	blocked := []string{
		"127.0.0.1",
		"::1",
		"169.254.169.254",
		"fe80::1",
		"224.0.0.1",
		"0.0.0.0",
		"0.1.2.3",       // 0.0.0.0/8 — routes to localhost on Linux
		"0.255.255.255", // top of 0.0.0.0/8
		"255.255.255.255", // limited broadcast
	}
	for _, s := range blocked {
		ip := net.ParseIP(s)
		require.NotNil(t, ip, s)
		assert.True(t, isBlockedIP(ip), "%s should be blocked", s)
	}

	allowed := []string{
		"192.168.1.50",
		"10.0.0.5",
		"172.16.5.10",
		"8.8.8.8",
		"2606:4700:4700::1111",
	}
	for _, s := range allowed {
		ip := net.ParseIP(s)
		require.NotNil(t, ip, s)
		assert.False(t, isBlockedIP(ip), "%s should be allowed", s)
	}
}

// guard against accidental reintroduction of the body in the Error field
func TestSendTest_ErrorOmitsResponseBodySubstring(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte("internal-marker-9f7c"))
	}))
	defer srv.Close()

	w, err := NewWebhookDispatcher("t", srv.URL, "", nil)
	require.NoError(t, err)
	w.client = &http.Client{Timeout: 5 * time.Second}

	result := w.SendTest(context.Background(), newTestNotification("x"))
	assert.False(t, strings.Contains(result.Error, "internal-marker"))
}

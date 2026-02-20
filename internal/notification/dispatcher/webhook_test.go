package dispatcher

import (
	"encoding/json"
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

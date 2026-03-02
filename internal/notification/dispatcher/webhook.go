package dispatcher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"text/template"
	"time"

	"github.com/amir20/dozzle/types"
	"github.com/rs/zerolog/log"
)

// UserAgent is set by the application at startup
var UserAgent = "Dozzle/head"

// WebhookDispatcher sends notifications to a webhook URL
type WebhookDispatcher struct {
	Name         string
	URL          string
	Template     *template.Template
	TemplateText string // Original template string for serialization
	client       *http.Client
}

// NewWebhookDispatcher creates a new webhook dispatcher
// If templateStr is empty, the notification will be marshaled as JSON directly
func NewWebhookDispatcher(name, url, templateStr string) (*WebhookDispatcher, error) {
	w := &WebhookDispatcher{
		Name:         name,
		URL:          url,
		TemplateText: templateStr,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	if templateStr != "" {
		tmpl, err := template.New("webhook").Parse(templateStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse template: %w", err)
		}
		w.Template = tmpl
	}

	return w, nil
}

// TestResult contains the result of a webhook test
type TestResult struct {
	Success    bool
	StatusCode int
	Error      string
}

// Send sends a notification to the webhook URL
func (w *WebhookDispatcher) Send(ctx context.Context, notification types.Notification) error {
	result := w.SendTest(ctx, notification)
	if !result.Success {
		return fmt.Errorf("webhook notification failed: %s", result.Error)
	}
	return nil
}

// SendTest sends a notification and returns detailed result for testing
func (w *WebhookDispatcher) SendTest(ctx context.Context, notification types.Notification) TestResult {
	var payload []byte
	var err error

	if w.Template != nil {
		payload, err = executeJSONTemplate(w.TemplateText, notification)
		if err != nil {
			return TestResult{Success: false, Error: fmt.Sprintf("failed to execute template: %v", err)}
		}
	} else {
		payload, err = json.Marshal(notification)
		if err != nil {
			return TestResult{Success: false, Error: fmt.Sprintf("failed to marshal notification: %v", err)}
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.URL, bytes.NewReader(payload))
	if err != nil {
		return TestResult{Success: false, Error: fmt.Sprintf("failed to create request: %v", err)}
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", UserAgent)

	resp, err := w.client.Do(req)
	if err != nil {
		return TestResult{Success: false, Error: fmt.Sprintf("failed to send webhook: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Limit response body to 1MB to prevent memory exhaustion
		limitedReader := io.LimitReader(resp.Body, 1024*1024)
		responseBody, _ := io.ReadAll(limitedReader)
		log.Debug().
			Str("webhook", w.Name).
			Str("url", w.URL).
			Int("status_code", resp.StatusCode).
			Str("payload", string(payload)).
			Str("response_body", string(responseBody)).
			Msg("webhook returned non-success status code")
		return TestResult{
			Success:    false,
			StatusCode: resp.StatusCode,
			Error:      fmt.Sprintf("webhook returned status code %d: %s", resp.StatusCode, string(responseBody)),
		}
	}

	return TestResult{Success: true, StatusCode: resp.StatusCode}
}

// executeJSONTemplate parses the template as JSON, resolves Go template placeholders
// in string values, and marshals back to JSON. This ensures all values are properly
// JSON-escaped regardless of their content (e.g., log messages containing quotes or braces).
func executeJSONTemplate(templateText string, data any) ([]byte, error) {
	var structure any
	if err := json.Unmarshal([]byte(templateText), &structure); err != nil {
		// Not valid JSON — fall back to raw text/template execution
		tmpl, parseErr := template.New("webhook").Parse(templateText)
		if parseErr != nil {
			return nil, fmt.Errorf("failed to parse template: %w", parseErr)
		}
		var buf bytes.Buffer
		if execErr := tmpl.Execute(&buf, data); execErr != nil {
			return nil, fmt.Errorf("failed to execute template: %w", execErr)
		}
		return buf.Bytes(), nil
	}

	resolved, err := resolveTemplateValues(structure, data)
	if err != nil {
		return nil, err
	}

	return json.Marshal(resolved)
}

// resolveTemplateValues recursively walks a JSON structure and executes
// Go template expressions found in string values.
func resolveTemplateValues(v any, data any) (any, error) {
	switch val := v.(type) {
	case map[string]any:
		result := make(map[string]any, len(val))
		for k, child := range val {
			resolved, err := resolveTemplateValues(child, data)
			if err != nil {
				return nil, err
			}
			result[k] = resolved
		}
		return result, nil
	case []any:
		result := make([]any, len(val))
		for i, child := range val {
			resolved, err := resolveTemplateValues(child, data)
			if err != nil {
				return nil, err
			}
			result[i] = resolved
		}
		return result, nil
	case string:
		if !strings.Contains(val, "{{") {
			return val, nil
		}
		tmpl, err := template.New("field").Parse(val)
		if err != nil {
			return nil, fmt.Errorf("failed to parse template field %q: %w", val, err)
		}
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data); err != nil {
			return nil, fmt.Errorf("failed to execute template field %q: %w", val, err)
		}
		return buf.String(), nil
	default:
		return val, nil
	}
}

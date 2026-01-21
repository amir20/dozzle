package dispatcher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"text/template"
	"time"

	"github.com/amir20/dozzle/types"
	"github.com/rs/zerolog/log"
)

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
		return fmt.Errorf("%s", result.Error)
	}
	return nil
}

// SendTest sends a notification and returns detailed result for testing
func (w *WebhookDispatcher) SendTest(ctx context.Context, notification types.Notification) TestResult {
	var payload []byte
	var err error

	if w.Template != nil {
		var buf bytes.Buffer
		if err := w.Template.Execute(&buf, notification); err != nil {
			return TestResult{Success: false, Error: fmt.Sprintf("failed to execute template: %v", err)}
		}
		payload = buf.Bytes()
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

	resp, err := w.client.Do(req)
	if err != nil {
		return TestResult{Success: false, Error: fmt.Sprintf("failed to send webhook: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		responseBody, _ := io.ReadAll(resp.Body)
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

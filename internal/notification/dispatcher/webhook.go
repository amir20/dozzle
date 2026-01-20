package dispatcher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"text/template"
	"time"
)

// WebhookDispatcher sends notifications to a webhook URL
type WebhookDispatcher struct {
	Name     string
	URL      string
	Template *template.Template
	client   *http.Client
}

// NewWebhookDispatcher creates a new webhook dispatcher
// If templateStr is empty, the notification will be marshaled as JSON directly
func NewWebhookDispatcher(name, url, templateStr string) (*WebhookDispatcher, error) {
	w := &WebhookDispatcher{
		Name: name,
		URL:  url,
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

// Send sends a notification to the webhook URL
func (w *WebhookDispatcher) Send(ctx context.Context, notification any) error {
	var payload []byte
	var err error

	if w.Template != nil {
		var buf bytes.Buffer
		if err := w.Template.Execute(&buf, notification); err != nil {
			return fmt.Errorf("failed to execute template: %w", err)
		}
		payload = buf.Bytes()
	} else {
		payload, err = json.Marshal(notification)
		if err != nil {
			return fmt.Errorf("failed to marshal notification: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.URL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned non-success status code: %d", resp.StatusCode)
	}

	return nil
}

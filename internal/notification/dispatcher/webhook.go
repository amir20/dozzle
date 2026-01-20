package dispatcher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// WebhookDispatcher sends notifications to a webhook URL
type WebhookDispatcher struct {
	Name   string
	URL    string
	client *http.Client
}

// NewWebhookDispatcher creates a new webhook dispatcher
func NewWebhookDispatcher(name, url string) *WebhookDispatcher {
	return &WebhookDispatcher{
		Name: name,
		URL:  url,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Send sends a notification to the webhook URL
func (w *WebhookDispatcher) Send(ctx context.Context, notification any) error {
	payload, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
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

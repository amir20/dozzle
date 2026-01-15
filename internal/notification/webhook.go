package notification

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

// Dispatcher is the interface for sending notifications
type Dispatcher interface {
	Dispatch(subscription *Subscription, payload WebhookPayload)
}

// WebhookDispatcher handles async webhook delivery
type WebhookDispatcher struct {
	client *http.Client
	config *DispatcherConfig
}

// NewWebhookDispatcher creates a new webhook dispatcher
func NewWebhookDispatcher(config *DispatcherConfig) *WebhookDispatcher {
	return &WebhookDispatcher{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		config: config,
	}
}

// Dispatch sends a webhook payload asynchronously
func (d *WebhookDispatcher) Dispatch(subscription *Subscription, payload WebhookPayload) {
	url := d.config.URL
	go func() {
		body, err := json.Marshal(payload)
		if err != nil {
			log.Error().Err(err).Str("url", url).Msg("failed to marshal webhook payload")
			return
		}

		req, err := http.NewRequest("POST", url, bytes.NewReader(body))
		if err != nil {
			log.Error().Err(err).Str("url", url).Msg("failed to create webhook request")
			return
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "Dozzle-Notifications/1.0")

		// Add API key if configured
		if d.config.APIKey != "" {
			req.Header.Set("Authorization", "Bearer "+d.config.APIKey)
		}

		// Add custom headers
		for key, value := range d.config.Headers {
			req.Header.Set(key, value)
		}

		resp, err := d.client.Do(req)
		if err != nil {
			log.Error().Err(err).Str("url", url).Msg("failed to send webhook")
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 400 {
			log.Warn().
				Int("status", resp.StatusCode).
				Str("url", url).
				Str("subscription", payload.SubscriptionName).
				Msg("webhook returned error status")
		} else {
			log.Debug().
				Int("status", resp.StatusCode).
				Str("url", url).
				Str("subscription", payload.SubscriptionName).
				Msg("webhook delivered successfully")
		}
	}()
}

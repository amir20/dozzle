package dispatcher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/amir20/dozzle/types"
	"github.com/rs/zerolog/log"
)

// CloudDispatcher sends notifications to Dozzle Cloud
type CloudDispatcher struct {
	Name      string
	URL       string
	APIKey    string
	Prefix    string
	ExpiresAt *time.Time
	client    *http.Client
}

// NewCloudDispatcher creates a new cloud dispatcher
func NewCloudDispatcher(name string, apiKey string, prefix string, expiresAt *time.Time) (*CloudDispatcher, error) {
	url := os.Getenv("DOLIGENCE_URL")
	if url == "" {
		url = "https://doligence.dozzle.dev"
	}
	url = url + "/api/events"

	if apiKey == "" {
		return nil, fmt.Errorf("API key is required for cloud dispatcher")
	}

	return &CloudDispatcher{
		Name:      name,
		URL:       url,
		APIKey:    apiKey,
		Prefix:    prefix,
		ExpiresAt: expiresAt,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}, nil
}

// Send sends a notification to Dozzle Cloud
func (c *CloudDispatcher) Send(ctx context.Context, notification types.Notification) error {
	payload, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.URL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", c.APIKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send to cloud: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		limitedReader := io.LimitReader(resp.Body, 1024*1024)
		responseBody, _ := io.ReadAll(limitedReader)
		log.Debug().
			Str("cloud", c.Name).
			Str("url", c.URL).
			Int("status_code", resp.StatusCode).
			Str("payload", string(payload)).
			Str("response_body", string(responseBody)).
			Msg("cloud returned non-success status code")
		return fmt.Errorf("cloud returned status code %d: %s", resp.StatusCode, string(responseBody))
	}

	return nil
}

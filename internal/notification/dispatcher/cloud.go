package dispatcher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/amir20/dozzle/types"
	"github.com/rs/zerolog/log"
)

// CloudDispatcher sends notifications to Dozzle Cloud
type CloudDispatcher struct {
	Name         string
	URL          string
	APIKey       string
	Prefix       string
	ExpiresAt    *time.Time
	client       *http.Client
	blockedUntil atomic.Int64
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

const defaultRetryAfter = 60 * time.Second

// unauthorizedRetryAfter is how long to back off after an auth failure (invalid/expired
// API key). Retrying won't help until the user fixes their key, which recreates the
// dispatcher and resets the breaker.
const unauthorizedRetryAfter = 6 * time.Hour

// ResetBreaker clears the circuit breaker so the next Send dials cloud again.
// Called when a cloud status check succeeds, proving the API key is valid.
func (c *CloudDispatcher) ResetBreaker() {
	c.blockedUntil.Store(0)
}

// Send sends a notification to Dozzle Cloud
func (c *CloudDispatcher) Send(ctx context.Context, notification types.Notification) error {
	if blockedUntil := c.blockedUntil.Load(); blockedUntil > 0 && time.Now().UnixNano() < blockedUntil {
		t := time.Unix(0, blockedUntil)
		log.Debug().
			Str("cloud", c.Name).
			Time("blocked_until", t).
			Msg("circuit breaker open, skipping cloud request")
		return fmt.Errorf("cloud dispatcher rate limited, retry after %s", t.Format(time.RFC3339))
	}

	payload, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.URL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("X-API-Key", c.APIKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send to cloud: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := defaultRetryAfter
		if ra := resp.Header.Get("Retry-After"); ra != "" {
			if seconds, err := strconv.Atoi(ra); err == nil {
				retryAfter = time.Duration(seconds) * time.Second
			}
		}
		c.blockedUntil.Store(time.Now().Add(retryAfter).UnixNano())
		log.Warn().
			Str("cloud", c.Name).
			Dur("retry_after", retryAfter).
			Msg("rate limited by cloud, circuit breaker tripped")
		return fmt.Errorf("cloud rate limited, backing off for %s", retryAfter)
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		limitedReader := io.LimitReader(resp.Body, 1024*1024)
		responseBody, _ := io.ReadAll(limitedReader)
		c.blockedUntil.Store(time.Now().Add(unauthorizedRetryAfter).UnixNano())
		log.Warn().
			Str("cloud", c.Name).
			Int("status_code", resp.StatusCode).
			Dur("retry_after", unauthorizedRetryAfter).
			Msg("cloud rejected API key, circuit breaker tripped")
		return fmt.Errorf("cloud returned status code %d: %s", resp.StatusCode, string(responseBody))
	}

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

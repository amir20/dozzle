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
	Name      string
	URL       string
	APIKey    string
	Prefix    string
	ExpiresAt *time.Time
	client    *http.Client
	breaker   atomic.Pointer[breakerState]
}

// breakerState records until when sends are skipped and why, so the error a
// skipped send returns names the failure that tripped it.
type breakerState struct {
	until  time.Time
	reason string
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

// serverErrorRetryAfter is how long to back off after a 5xx that carries no
// Retry-After, such as a proxy's 502 while cloud restarts during a deploy. It
// must stay short or alerts are dropped for nothing.
const serverErrorRetryAfter = 30 * time.Second

// unauthorizedRetryAfter is how long to back off after an auth failure (invalid/expired
// API key). Retrying won't help until the user fixes their key, which recreates the
// dispatcher and resets the breaker.
const unauthorizedRetryAfter = 6 * time.Hour

// ResetBreaker clears the circuit breaker so the next Send dials cloud again.
// Called when a cloud status check succeeds, proving the API key is valid.
func (c *CloudDispatcher) ResetBreaker() {
	c.breaker.Store(nil)
}

func (c *CloudDispatcher) trip(retryAfter time.Duration, reason string) {
	c.breaker.Store(&breakerState{until: time.Now().Add(retryAfter), reason: reason})
}

// parseRetryAfter reads a Retry-After header given in seconds, falling back to
// def when it is missing or unparsable.
func parseRetryAfter(header string, def time.Duration) time.Duration {
	if seconds, err := strconv.Atoi(header); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	return def
}

// Send sends a notification to Dozzle Cloud
func (c *CloudDispatcher) Send(ctx context.Context, notification types.Notification) error {
	if b := c.breaker.Load(); b != nil && time.Now().Before(b.until) {
		log.Debug().
			Str("cloud", c.Name).
			Str("reason", b.reason).
			Time("blocked_until", b.until).
			Msg("circuit breaker open, skipping cloud request")
		return fmt.Errorf("cloud dispatcher paused after %s, retry after %s", b.reason, b.until.Format(time.RFC3339))
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
		retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"), defaultRetryAfter)
		c.trip(retryAfter, "rate limit (429)")
		log.Warn().
			Str("cloud", c.Name).
			Dur("retry_after", retryAfter).
			Msg("rate limited by cloud, circuit breaker tripped")
		return fmt.Errorf("cloud rate limited, backing off for %s", retryAfter)
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		limitedReader := io.LimitReader(resp.Body, 1024*1024)
		responseBody, _ := io.ReadAll(limitedReader)
		c.trip(unauthorizedRetryAfter, fmt.Sprintf("API key rejected (%d)", resp.StatusCode))
		log.Warn().
			Str("cloud", c.Name).
			Int("status_code", resp.StatusCode).
			Dur("retry_after", unauthorizedRetryAfter).
			Msg("cloud rejected API key, circuit breaker tripped")
		return fmt.Errorf("cloud returned status code %d: %s", resp.StatusCode, string(responseBody))
	}

	if resp.StatusCode >= 500 {
		limitedReader := io.LimitReader(resp.Body, 1024*1024)
		responseBody, _ := io.ReadAll(limitedReader)
		retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"), serverErrorRetryAfter)
		c.trip(retryAfter, fmt.Sprintf("server error (%d)", resp.StatusCode))
		log.Warn().
			Str("cloud", c.Name).
			Int("status_code", resp.StatusCode).
			Dur("retry_after", retryAfter).
			Msg("cloud unavailable, circuit breaker tripped")
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

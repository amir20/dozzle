package dispatcher

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"sync"
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

	// mu guards the retry queue. Notifications that hit a transient failure wait
	// here and are resent oldest first once the breaker lets requests through.
	mu         sync.Mutex
	queue      []*queuedNotification
	flushTimer *time.Timer
	flushing   bool
	closed     bool
	dropped    int
}

// breakerState records until when sends are skipped and why, so the error a
// skipped send returns names the failure that tripped it. A retryable breaker
// queues sends instead of failing them.
type breakerState struct {
	until     time.Time
	reason    string
	retryable bool
}

type queuedNotification struct {
	notification types.Notification
	queuedAt     time.Time
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

// maxQueued caps the retry queue so a long cloud outage cannot grow memory.
// When full, the oldest notification is dropped.
const maxQueued = 100

// maxQueueAge is how long a notification may wait for cloud. Older ones are
// dropped so a long outage does not deliver a burst of stale alerts.
const maxQueueAge = 5 * time.Minute

// ResetBreaker clears the circuit breaker so the next Send dials cloud again,
// and resends anything queued. Called when a cloud status check succeeds,
// proving the API key is valid.
func (c *CloudDispatcher) ResetBreaker() {
	c.breaker.Store(nil)
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.queue) > 0 && !c.flushing && !c.closed {
		c.stopTimerLocked()
		c.flushing = true
		go c.flush()
	}
}

func (c *CloudDispatcher) trip(retryAfter time.Duration, reason string, retryable bool) {
	c.breaker.Store(&breakerState{until: time.Now().Add(retryAfter), reason: reason, retryable: retryable})
}

// parseRetryAfter reads a Retry-After header given in seconds, falling back to
// def when it is missing or unparsable.
func parseRetryAfter(header string, def time.Duration) time.Duration {
	if seconds, err := strconv.Atoi(header); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	return def
}

// Send sends a notification to Dozzle Cloud. When cloud is briefly unavailable
// (5xx, 429 or a network error) the notification is queued and resent later,
// and Send returns nil.
func (c *CloudDispatcher) Send(ctx context.Context, notification types.Notification) error {
	if b := c.breaker.Load(); b != nil && time.Now().Before(b.until) {
		if b.retryable && c.enqueue(notification) {
			return nil
		}
		log.Debug().
			Str("cloud", c.Name).
			Str("reason", b.reason).
			Time("blocked_until", b.until).
			Msg("circuit breaker open, skipping cloud request")
		return fmt.Errorf("cloud dispatcher paused after %s, retry after %s", b.reason, b.until.Format(time.RFC3339))
	}

	// Keep order: while older notifications are waiting, this one waits behind them.
	c.mu.Lock()
	pending := len(c.queue) > 0
	c.mu.Unlock()
	if pending && c.enqueue(notification) {
		return nil
	}

	err := c.post(ctx, notification)
	if errors.Is(err, errTransient) && c.enqueue(notification) {
		return nil
	}
	return err
}

// errTransient marks a failure worth retrying: cloud was unreachable or
// overloaded, not a problem with the request or the API key.
var errTransient = errors.New("transient cloud failure")

// post makes one request to cloud and trips the breaker on failure. Errors
// wrapping errTransient are worth retrying.
func (c *CloudDispatcher) post(ctx context.Context, notification types.Notification) error {
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
		// A cancelled caller (shutdown) is not cloud's fault and is not retried.
		if ctx.Err() != nil {
			return fmt.Errorf("failed to send to cloud: %w", err)
		}
		c.trip(serverErrorRetryAfter, "network error", true)
		log.Warn().
			Err(err).
			Str("cloud", c.Name).
			Dur("retry_after", serverErrorRetryAfter).
			Msg("cloud unreachable, circuit breaker tripped")
		return fmt.Errorf("%w: failed to send to cloud: %w", errTransient, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"), defaultRetryAfter)
		c.trip(retryAfter, "rate limit (429)", true)
		log.Warn().
			Str("cloud", c.Name).
			Dur("retry_after", retryAfter).
			Msg("rate limited by cloud, circuit breaker tripped")
		return fmt.Errorf("%w: cloud rate limited, backing off for %s", errTransient, retryAfter)
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		limitedReader := io.LimitReader(resp.Body, 1024*1024)
		responseBody, _ := io.ReadAll(limitedReader)
		c.trip(unauthorizedRetryAfter, fmt.Sprintf("API key rejected (%d)", resp.StatusCode), false)
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
		c.trip(retryAfter, fmt.Sprintf("server error (%d)", resp.StatusCode), true)
		log.Warn().
			Str("cloud", c.Name).
			Int("status_code", resp.StatusCode).
			Dur("retry_after", retryAfter).
			Msg("cloud unavailable, circuit breaker tripped")
		return fmt.Errorf("%w: cloud returned status code %d: %s", errTransient, resp.StatusCode, string(responseBody))
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

// Close stops any scheduled resend and discards the queue. Called when the
// dispatcher is replaced or cloud is disconnected, so queued notifications are
// never sent with a key the user has removed.
func (c *CloudDispatcher) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.queue) > 0 {
		log.Warn().Str("cloud", c.Name).Int("dropped", len(c.queue)).Msg("dropped queued notifications, cloud dispatcher replaced")
	}
	c.closeLocked()
}

func (c *CloudDispatcher) closeLocked() []*queuedNotification {
	queue := c.queue
	c.closed = true
	c.stopTimerLocked()
	c.queue = nil
	return queue
}

// TakeOver moves old's queue and breaker into c and closes old. Used when the
// config is rebroadcast with the same key, so pending alerts survive the swap.
func (c *CloudDispatcher) TakeOver(old *CloudDispatcher) {
	old.mu.Lock()
	queue := old.closeLocked()
	old.mu.Unlock()

	c.breaker.Store(old.breaker.Load())
	if len(queue) == 0 {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.queue = append(queue, c.queue...)
	if over := len(c.queue) - maxQueued; over > 0 {
		c.queue = c.queue[over:]
		c.dropped += over
	}
	if !c.flushing && c.flushTimer == nil {
		c.scheduleLocked()
	}
}

// enqueue adds a notification to the retry queue, dropping the oldest when
// full, and makes sure a flush is scheduled. It reports false once closed.
func (c *CloudDispatcher) enqueue(notification types.Notification) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return false
	}
	if len(c.queue) >= maxQueued {
		c.queue = c.queue[1:]
		c.dropped++
	}
	c.queue = append(c.queue, &queuedNotification{notification: notification, queuedAt: time.Now()})
	log.Debug().Str("cloud", c.Name).Int("queued", len(c.queue)).Msg("queued notification for retry")
	if !c.flushing && c.flushTimer == nil {
		c.scheduleLocked()
	}
	return true
}

// scheduleLocked arms a one-shot timer for when the breaker closes. A timer
// rather than a goroutine, so a replaced dispatcher leaves nothing running.
func (c *CloudDispatcher) scheduleLocked() {
	delay := time.Duration(0)
	if b := c.breaker.Load(); b != nil {
		delay = time.Until(b.until)
	}
	c.flushTimer = time.AfterFunc(max(delay, 0), func() {
		c.mu.Lock()
		c.flushTimer = nil
		if c.flushing {
			c.mu.Unlock()
			return
		}
		c.flushing = true
		c.mu.Unlock()
		c.flush()
	})
}

func (c *CloudDispatcher) stopTimerLocked() {
	if c.flushTimer != nil {
		c.flushTimer.Stop()
		c.flushTimer = nil
	}
}

// flush resends queued notifications oldest first until the queue is empty or
// cloud fails again, in which case it reschedules itself. The caller must have
// set c.flushing.
func (c *CloudDispatcher) flush() {
	for {
		c.mu.Lock()
		if c.closed {
			c.flushing = false
			c.mu.Unlock()
			return
		}
		c.dropExpiredLocked()
		if c.dropped > 0 {
			log.Warn().Str("cloud", c.Name).Int("dropped", c.dropped).Msg("dropped queued notifications, cloud unavailable for too long")
			c.dropped = 0
		}
		if len(c.queue) == 0 {
			c.flushing = false
			c.mu.Unlock()
			return
		}
		if b := c.breaker.Load(); b != nil && time.Now().Before(b.until) {
			if !b.retryable {
				// The API key was rejected; resending cannot succeed.
				log.Warn().Str("cloud", c.Name).Int("dropped", len(c.queue)).Str("reason", b.reason).Msg("dropped queued notifications")
				c.queue = nil
				c.flushing = false
				c.mu.Unlock()
				return
			}
			c.flushing = false
			c.scheduleLocked()
			c.mu.Unlock()
			return
		}
		next := c.queue[0]
		c.mu.Unlock()

		err := c.post(context.Background(), next.notification)

		c.mu.Lock()
		if !errors.Is(err, errTransient) {
			if err != nil {
				log.Warn().Err(err).Str("cloud", c.Name).Msg("failed to resend queued notification")
			}
			// The queue may have shifted under an overflow while posting.
			if len(c.queue) > 0 && c.queue[0] == next {
				c.queue = c.queue[1:]
			}
		}
		c.mu.Unlock()
	}
}

func (c *CloudDispatcher) dropExpiredLocked() {
	cutoff := time.Now().Add(-maxQueueAge)
	i := 0
	for i < len(c.queue) && c.queue[i].queuedAt.Before(cutoff) {
		i++
	}
	c.dropped += i
	c.queue = c.queue[i:]
}

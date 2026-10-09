package dispatcher

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/amir20/dozzle/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestCloudDispatcher(url string) *CloudDispatcher {
	return &CloudDispatcher{
		Name:   "Dozzle Cloud",
		URL:    url,
		APIKey: "test-key",
		client: &http.Client{Timeout: 5 * time.Second},
	}
}

// On a 401/403 the breaker trips and subsequent sends short-circuit without
// hitting cloud until the breaker is reset.
func TestCloudDispatcher_AuthFailureTripsBreaker(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusForbidden} {
		var hits atomic.Int32
		srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			hits.Add(1)
			rw.WriteHeader(status)
			rw.Write([]byte("Invalid API key\n"))
		}))

		d := newTestCloudDispatcher(srv.URL)

		err := d.Send(context.Background(), newTestNotification("first"))
		require.Error(t, err)
		assert.EqualValues(t, 1, hits.Load(), "first send should reach cloud")

		err = d.Send(context.Background(), newTestNotification("second"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "API key rejected")
		assert.NotContains(t, err.Error(), "rate limit")
		assert.EqualValues(t, 1, hits.Load(), "breaker should block second send (status %d)", status)

		srv.Close()
	}
}

// ResetBreaker clears the circuit so the next send dials cloud again.
func TestCloudDispatcher_ResetBreaker(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		rw.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	d := newTestCloudDispatcher(srv.URL)

	require.Error(t, d.Send(context.Background(), newTestNotification("first")))
	require.EqualValues(t, 1, hits.Load())

	// Blocked while breaker is open.
	require.Error(t, d.Send(context.Background(), newTestNotification("blocked")))
	require.EqualValues(t, 1, hits.Load())

	d.ResetBreaker()

	require.Error(t, d.Send(context.Background(), newTestNotification("after-reset")))
	assert.EqualValues(t, 2, hits.Load(), "send after reset should reach cloud again")
}

// A 5xx trips a short breaker, never the 6h auth one: cloud restarting during a
// deploy must not silence notifications for hours.
func TestCloudDispatcher_ServerErrorTripsShortBreaker(t *testing.T) {
	tests := []struct {
		name       string
		retryAfter string
		want       time.Duration
	}{
		{"no Retry-After", "", serverErrorRetryAfter},
		{"honors Retry-After", "10", 10 * time.Second},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var hits atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				hits.Add(1)
				if tc.retryAfter != "" {
					rw.Header().Set("Retry-After", tc.retryAfter)
				}
				rw.WriteHeader(http.StatusServiceUnavailable)
			}))
			defer srv.Close()

			d := newTestCloudDispatcher(srv.URL)

			start := time.Now()
			require.NoError(t, d.Send(context.Background(), newTestNotification("first")))
			require.EqualValues(t, 1, hits.Load())

			b := d.breaker.Load()
			require.NotNil(t, b)
			assert.WithinDuration(t, start.Add(tc.want), b.until, 2*time.Second)

			require.NoError(t, d.Send(context.Background(), newTestNotification("second")))
			assert.EqualValues(t, 1, hits.Load(), "breaker should block second send")
			assert.Equal(t, 2, d.queueLen(), "both sends wait in the retry queue")
		})
	}
}

// Once a 5xx breaker expires, sends reach cloud again.
func TestCloudDispatcher_ServerErrorBreakerExpires(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		rw.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	d := newTestCloudDispatcher(srv.URL)
	d.breaker.Store(&breakerState{until: time.Now().Add(-time.Second), reason: "server error (503)"})

	require.NoError(t, d.Send(context.Background(), newTestNotification("after-expiry")))
	assert.EqualValues(t, 1, hits.Load())
}

func (c *CloudDispatcher) queueLen() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.queue)
}

// A notification that hits a 5xx is held and delivered, in order, once cloud
// is back.
func TestCloudDispatcher_QueuedNotificationsResentInOrder(t *testing.T) {
	var hits atomic.Int32
	received := make(chan string, 10)
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if hits.Add(1) == 1 {
			rw.Header().Set("Retry-After", "1")
			rw.WriteHeader(http.StatusBadGateway)
			return
		}
		var n types.Notification
		_ = json.NewDecoder(r.Body).Decode(&n)
		received <- n.Detail
		rw.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	d := newTestCloudDispatcher(srv.URL)

	require.NoError(t, d.Send(context.Background(), newTestNotification("first")))
	require.NoError(t, d.Send(context.Background(), newTestNotification("second")))
	require.EqualValues(t, 1, hits.Load())

	for _, want := range []string{"first", "second"} {
		select {
		case got := <-received:
			assert.Equal(t, want, got)
		case <-time.After(5 * time.Second):
			t.Fatalf("timed out waiting for %q", want)
		}
	}
	assert.Eventually(t, func() bool { return d.queueLen() == 0 }, time.Second, 10*time.Millisecond)
}

// A network failure is retried just like a 5xx.
func TestCloudDispatcher_NetworkErrorQueues(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {}))
	srv.Close()

	d := newTestCloudDispatcher(srv.URL)
	require.NoError(t, d.Send(context.Background(), newTestNotification("first")))
	assert.Equal(t, 1, d.queueLen())
	b := d.breaker.Load()
	require.NotNil(t, b)
	assert.True(t, b.retryable)
}

// ResetBreaker resends the queue right away instead of waiting out the pause.
func TestCloudDispatcher_ResetBreakerFlushesQueue(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		rw.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	d := newTestCloudDispatcher(srv.URL)
	d.trip(time.Hour, "server error (503)", true)
	require.NoError(t, d.Send(context.Background(), newTestNotification("queued")))
	require.EqualValues(t, 0, hits.Load())

	d.ResetBreaker()
	assert.Eventually(t, func() bool { return hits.Load() == 1 && d.queueLen() == 0 }, 2*time.Second, 10*time.Millisecond)
}

// The queue is capped; the oldest notifications go first.
func TestCloudDispatcher_QueueDropsOldestWhenFull(t *testing.T) {
	d := newTestCloudDispatcher("http://127.0.0.1:0")
	d.trip(time.Hour, "server error (503)", true)
	for i := range maxQueued + 5 {
		require.NoError(t, d.Send(context.Background(), newTestNotification(fmt.Sprint(i))))
	}

	d.mu.Lock()
	defer d.mu.Unlock()
	d.stopTimerLocked()
	require.Len(t, d.queue, maxQueued)
	assert.Equal(t, "5", d.queue[0].notification.Detail)
	assert.Equal(t, 5, d.dropped)
}

// Notifications older than maxQueueAge are dropped instead of resent.
func TestCloudDispatcher_ExpiredNotificationsDropped(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		rw.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	d := newTestCloudDispatcher(srv.URL)
	d.queue = []*queuedNotification{
		{notification: newTestNotification("stale"), queuedAt: time.Now().Add(-maxQueueAge - time.Second)},
		{notification: newTestNotification("fresh"), queuedAt: time.Now()},
	}
	d.flushing = true
	d.flush()

	assert.EqualValues(t, 1, hits.Load(), "only the fresh notification is resent")
	assert.Equal(t, 0, d.queueLen())
}

// A rejected API key empties the queue; resending cannot succeed.
func TestCloudDispatcher_AuthFailureDropsQueue(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		rw.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	d := newTestCloudDispatcher(srv.URL)
	now := time.Now()
	d.queue = []*queuedNotification{
		{notification: newTestNotification("a"), queuedAt: now},
		{notification: newTestNotification("b"), queuedAt: now},
	}
	d.flushing = true
	d.flush()

	assert.EqualValues(t, 1, hits.Load())
	assert.Equal(t, 0, d.queueLen())
}

// Once closed, nothing queued is resent and new failures are not queued.
func TestCloudDispatcher_CloseDiscardsQueue(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		rw.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	d := newTestCloudDispatcher(srv.URL)
	require.NoError(t, d.Send(context.Background(), newTestNotification("queued")))
	require.Equal(t, 1, d.queueLen())

	d.Close()
	assert.Equal(t, 0, d.queueLen())
	d.mu.Lock()
	assert.Nil(t, d.flushTimer)
	d.mu.Unlock()

	assert.Error(t, d.Send(context.Background(), newTestNotification("after-close")), "a closed dispatcher reports the failure instead of queuing")
	d.ResetBreaker()
	require.Error(t, d.Send(context.Background(), newTestNotification("after-reset")))
	assert.Equal(t, 0, d.queueLen())
}

// A same-key handoff keeps a transient pause but drops the 6h auth one, which
// agents only ever leave by getting a fresh dispatcher.
func TestCloudDispatcher_TakeOverCarriesOnlyRetryableBreaker(t *testing.T) {
	for _, tc := range []struct {
		name      string
		retryable bool
	}{
		{"server error carries over", true},
		{"auth failure resets", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			old := newTestCloudDispatcher("http://127.0.0.1:0")
			old.trip(time.Hour, "reason", tc.retryable)

			d := newTestCloudDispatcher("http://127.0.0.1:0")
			d.TakeOver(old)

			if tc.retryable {
				assert.NotNil(t, d.breaker.Load())
			} else {
				assert.Nil(t, d.breaker.Load())
			}
		})
	}
}

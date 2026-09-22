package dispatcher

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

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
		{"caps Retry-After", "3600", maxServerErrorRetryAfter},
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
			require.Error(t, d.Send(context.Background(), newTestNotification("first")))
			require.EqualValues(t, 1, hits.Load())

			b := d.breaker.Load()
			require.NotNil(t, b)
			assert.WithinDuration(t, start.Add(tc.want), b.until, 2*time.Second)

			err := d.Send(context.Background(), newTestNotification("second"))
			require.Error(t, err)
			assert.Contains(t, err.Error(), "server error (503)")
			assert.EqualValues(t, 1, hits.Load(), "breaker should block second send")
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

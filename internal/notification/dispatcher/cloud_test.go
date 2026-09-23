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

// Consecutive auth failures double the backoff from a minute up to the cap, and a
// success resets it.
func TestCloudDispatcher_AuthBackoffGrows(t *testing.T) {
	var status atomic.Int32
	status.Store(http.StatusUnauthorized)
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(int(status.Load()))
	}))
	defer srv.Close()

	d := newTestCloudDispatcher(srv.URL)
	blockedFor := func() time.Duration {
		return time.Until(time.Unix(0, d.blockedUntil.Load()))
	}

	for _, want := range []time.Duration{time.Minute, 2 * time.Minute, 4 * time.Minute} {
		d.blockedUntil.Store(0) // let the next send through without resetting the count
		require.Error(t, d.Send(context.Background(), newTestNotification("fail")))
		assert.InDelta(t, want.Seconds(), blockedFor().Seconds(), 5)
	}

	status.Store(http.StatusOK)
	d.blockedUntil.Store(0)
	require.NoError(t, d.Send(context.Background(), newTestNotification("ok")))

	status.Store(http.StatusUnauthorized)
	require.Error(t, d.Send(context.Background(), newTestNotification("fail again")))
	assert.InDelta(t, time.Minute.Seconds(), blockedFor().Seconds(), 5, "success should reset the backoff")
}

func TestUnauthorizedRetryAfter_Capped(t *testing.T) {
	assert.Equal(t, time.Minute, unauthorizedRetryAfter(1))
	assert.Equal(t, 256*time.Minute, unauthorizedRetryAfter(9))
	assert.Equal(t, maxUnauthorizedRetryAfter, unauthorizedRetryAfter(10))
	assert.Equal(t, maxUnauthorizedRetryAfter, unauthorizedRetryAfter(1000))
}

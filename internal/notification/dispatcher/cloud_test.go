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
		assert.Contains(t, err.Error(), "rate limited")
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

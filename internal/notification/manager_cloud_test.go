package notification

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/notification/dispatcher"
	"github.com/amir20/dozzle/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Replacing or clearing the cloud dispatcher must stop the old one resending
// its queue, or alerts go out with a key the user just removed.
func TestManager_SwappingCloudDispatcherDropsOldQueue(t *testing.T) {
	for _, tc := range []struct {
		name string
		swap func(m *Manager)
	}{
		{"cleared", func(m *Manager) { m.ClearCloudDispatcher() }},
		{"replaced", func(m *Manager) {
			d, err := dispatcher.NewCloudDispatcher("Dozzle Cloud", "new-key", "", nil)
			require.NoError(t, err)
			m.SetCloudDispatcher(d)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var hits atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				hits.Add(1)
				rw.WriteHeader(http.StatusBadGateway)
			}))
			defer srv.Close()
			t.Setenv("DOLIGENCE_URL", srv.URL)

			old, err := dispatcher.NewCloudDispatcher("Dozzle Cloud", "old-key", "", nil)
			require.NoError(t, err)
			m := &Manager{}
			m.SetCloudDispatcher(old)
			require.NoError(t, old.Send(context.Background(), types.Notification{Detail: "queued"}))
			require.EqualValues(t, 1, hits.Load())

			tc.swap(m)
			old.ResetBreaker() // would resend immediately if the queue survived

			time.Sleep(100 * time.Millisecond)
			assert.EqualValues(t, 1, hits.Load(), "old dispatcher must not resend after being swapped out")
		})
	}
}

// Setting the same dispatcher again must not close it.
func TestManager_SettingSameCloudDispatcherKeepsIt(t *testing.T) {
	d, err := dispatcher.NewCloudDispatcher("Dozzle Cloud", "key", "", nil)
	require.NoError(t, err)
	m := &Manager{}
	m.SetCloudDispatcher(d)
	m.SetCloudDispatcher(d)

	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()
	d.URL = srv.URL
	assert.NoError(t, d.Send(context.Background(), types.Notification{}), "still queues, so it was not closed")
	d.Close()
}

package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/auth"
	"github.com/stretchr/testify/assert"
)

func Test_feedAutoPrune(t *testing.T) {
	now := time.Now()
	assert.False(t, feedAutoPrune(nil, now))
	assert.True(t, feedAutoPrune(&autoPruneRun{At: now.Add(-time.Hour), SpaceReclaimed: 1}, now))
	assert.True(t, feedAutoPrune(&autoPruneRun{At: now.Add(-time.Hour), Failed: 1}, now), "a failed host is worth showing")
	assert.False(t, feedAutoPrune(&autoPruneRun{At: now.Add(-time.Hour)}, now), "a run that freed nothing is not news")
	assert.False(t, feedAutoPrune(&autoPruneRun{At: now.Add(-8 * 24 * time.Hour), SpaceReclaimed: 1}, now))
}

// The megaphone hears about the last clean up when the stream opens.
func Test_streamEvents_sends_last_auto_prune(t *testing.T) {
	autoPruneLast.Lock()
	prev := autoPruneLast.run
	autoPruneLast.run = &autoPruneRun{At: time.Now(), SpaceReclaimed: 3_200_000_000}
	autoPruneLast.Unlock()
	t.Cleanup(func() {
		autoPruneLast.Lock()
		autoPruneLast.run = prev
		autoPruneLast.Unlock()
	})

	h := restrictedHandler(t)
	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodGet, "/api/events/stream", nil).WithContext(auth.WithUser(ctx, auth.User{}))
	rr := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		defer close(done)
		h.streamEvents(rr, req)
	}()
	time.Sleep(300 * time.Millisecond)
	cancel()
	<-done

	assert.Contains(t, rr.Body.String(), "event: auto-prune")
	assert.Contains(t, rr.Body.String(), `"spaceReclaimed":3200000000`)
}

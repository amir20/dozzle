package docker_support

import (
	"context"
	"crypto/tls"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/stretchr/testify/assert"
)

// One caller may open several host subscriptions under a single context — the
// cloud client's log and stat streamers share one stream lifetime — so keying
// the subscriber set by context would let the second registration silently
// evict the first, and one of the two would never hear that an agent came up.
func TestRetriableClientManager_SubscribeSharedContext(t *testing.T) {
	m := NewRetriableClientManager(nil, time.Second, tls.Certificate{})

	ctx := t.Context()

	first := make(chan container.Host, 1)
	second := make(chan container.Host, 1)
	m.Subscribe(ctx, first)
	m.Subscribe(ctx, second)

	assert.Equal(t, 2, m.subscribers.Size())
}

func TestRetriableClientManager_SubscribeUnregistersOnDone(t *testing.T) {
	m := NewRetriableClientManager(nil, time.Second, tls.Certificate{})

	ctx, cancel := context.WithCancel(context.Background())
	m.Subscribe(ctx, make(chan container.Host, 1))
	assert.Equal(t, 1, m.subscribers.Size())

	cancel()
	assert.Eventually(t, func() bool {
		return m.subscribers.Size() == 0
	}, time.Second, 5*time.Millisecond)
}

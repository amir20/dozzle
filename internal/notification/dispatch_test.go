package notification

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/amir20/dozzle/types"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/semaphore"
)

type blockingDispatcher struct {
	release chan struct{}
	sent    atomic.Int32
}

func (d *blockingDispatcher) Send(ctx context.Context, _ types.Notification) error {
	select {
	case <-d.release:
	case <-ctx.Done():
	}
	d.sent.Add(1)
	return nil
}

// A slow destination fills every send slot. Matches past that are dropped right
// away instead of each parking a goroutine that waits for a slot.
func TestDispatch_DropsWhenEverySlotIsBusy(t *testing.T) {
	m := &Manager{ctx: t.Context(), sendSem: semaphore.NewWeighted(1)}
	d := &blockingDispatcher{release: make(chan struct{})}

	for range 10 {
		m.dispatch(d, types.Notification{}, 1)
	}
	close(d.release)

	assert.Eventually(t, func() bool { return d.sent.Load() == 1 }, time.Second, time.Millisecond)
	time.Sleep(50 * time.Millisecond)
	assert.EqualValues(t, 1, d.sent.Load(), "the matches that found no slot are dropped")
}

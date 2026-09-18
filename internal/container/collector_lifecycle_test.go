package container

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCollectorLifecycle_firstHolderRunsOthersJoin(t *testing.T) {
	var l CollectorLifecycle
	ctx, run := l.Acquire(t.Context(), time.Hour)
	assert.True(t, run)
	assert.NotNil(t, ctx)

	_, run = l.Acquire(t.Context(), time.Hour)
	assert.False(t, run, "a collector is already running")
	assert.Equal(t, 2, l.Holders())
}

func TestCollectorLifecycle_stopsIdleAfterLastRelease(t *testing.T) {
	var l CollectorLifecycle
	ctx, _ := l.Acquire(t.Context(), 10*time.Millisecond)
	l.Release(10 * time.Millisecond)

	select {
	case <-ctx.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("collector was not stopped after going idle")
	}
	assert.False(t, l.Running())
}

func TestCollectorLifecycle_acquireCancelsPendingStop(t *testing.T) {
	var l CollectorLifecycle
	ctx, _ := l.Acquire(t.Context(), 20*time.Millisecond)
	l.Release(20 * time.Millisecond)
	_, run := l.Acquire(t.Context(), 20*time.Millisecond)
	assert.False(t, run)

	time.Sleep(60 * time.Millisecond)
	assert.NoError(t, ctx.Err(), "a held collector keeps running")
}

// A release that lands before its acquire must not leave a collector nobody holds.
func TestCollectorLifecycle_releaseBeforeAcquire(t *testing.T) {
	var l CollectorLifecycle
	l.Release(time.Hour)
	_, run := l.Acquire(t.Context(), time.Hour)
	assert.False(t, run)
	assert.False(t, l.Running())
	assert.Equal(t, 0, l.Holders())
}

// The stop timer can fire and then wait on mu while Acquire takes a new reference.
func TestCollectorLifecycle_firedTimerSkipsHeldCollector(t *testing.T) {
	var l CollectorLifecycle
	ctx, _ := l.Acquire(t.Context(), time.Hour)
	l.stopIfIdle()
	assert.NoError(t, ctx.Err())
	assert.True(t, l.Running())
}

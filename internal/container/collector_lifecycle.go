package container

import (
	"context"
	"sync"
	"time"
)

// CollectorLifecycle is the reference count and idle timer a lazy stats collector
// runs on: it starts with its first holder and stops some time after its last one
// leaves. Docker and k8s each carried a copy of this, and every race fixed in one had
// to be fixed again in the other.
type CollectorLifecycle struct {
	mu      sync.Mutex
	stopper context.CancelFunc
	timer   *time.Timer
	holders int
}

// Acquire takes a reference. run is true when the caller should run the collector on
// ctx, and false when one is already running or this reference was already released.
//
// Callers acquire and release from separate goroutines, so a subscriber whose ctx is
// already done can release first. A count still <= 0 means that release already ran:
// starting here would leave a collector nobody holds and no timer to end it.
func (l *CollectorLifecycle) Acquire(parent context.Context, idle time.Duration) (ctx context.Context, run bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.timer != nil {
		l.timer.Stop()
		l.timer = nil
	}
	l.holders++
	if l.holders <= 0 {
		if l.stopper != nil {
			l.timer = time.AfterFunc(idle, l.stopIfIdle)
		}
		return nil, false
	}
	if l.stopper != nil {
		return nil, false
	}
	ctx, l.stopper = context.WithCancel(parent)
	return ctx, true
}

// Release drops a reference. The collector stops idle after the last one is gone.
func (l *CollectorLifecycle) Release(idle time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.holders--
	if l.holders == 0 {
		l.timer = time.AfterFunc(idle, l.stopIfIdle)
	}
}

func (l *CollectorLifecycle) stopIfIdle() {
	l.mu.Lock()
	defer l.mu.Unlock()
	// The timer can fire and then wait here on mu while Acquire takes a new reference.
	// Timer.Stop cannot recall a callback that already fired, so check the count.
	if l.holders > 0 {
		return
	}
	if l.stopper != nil {
		l.stopper()
		l.stopper = nil
	}
}

func (l *CollectorLifecycle) Running() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.stopper != nil
}

func (l *CollectorLifecycle) Holders() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.holders
}

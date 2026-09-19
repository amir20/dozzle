package container

import (
	"context"
	"time"
)

// NextBackoff doubles up to the ceiling.
func NextBackoff(d, ceiling time.Duration) time.Duration {
	return min(d*2, ceiling)
}

// SleepOrDone waits out the backoff, returning false if ctx ended first.
func SleepOrDone(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

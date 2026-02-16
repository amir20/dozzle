package notification

import (
	"context"
	"time"

	"github.com/puzpuzpuz/xsync/v4"
)

type ttlEntry[V any] struct {
	value     V
	expiresAt time.Time
}

// TTLCache is a concurrent-safe cache with per-entry TTL and periodic eviction.
type TTLCache[K comparable, V any] struct {
	entries *xsync.Map[K, ttlEntry[V]]
	ttl     time.Duration
}

// NewTTLCache creates a cache with the given TTL. It starts a background goroutine
// that sweeps expired entries every sweepInterval. The goroutine stops when ctx is cancelled.
func NewTTLCache[K comparable, V any](ctx context.Context, ttl time.Duration) *TTLCache[K, V] {
	c := &TTLCache[K, V]{
		entries: xsync.NewMap[K, ttlEntry[V]](),
		ttl:     ttl,
	}

	sweepInterval := max(ttl*2, time.Minute)

	go func() {
		ticker := time.NewTicker(sweepInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				c.evict()
			}
		}
	}()

	return c
}

// Load returns the cached value if present and not expired.
func (c *TTLCache[K, V]) Load(key K) (V, bool) {
	entry, ok := c.entries.Load(key)
	if !ok || time.Now().After(entry.expiresAt) {
		var zero V
		return zero, false
	}
	return entry.value, true
}

// Store adds or updates a value in the cache with the configured TTL.
func (c *TTLCache[K, V]) Store(key K, value V) {
	c.entries.Store(key, ttlEntry[V]{
		value:     value,
		expiresAt: time.Now().Add(c.ttl),
	})
}

// evict removes all expired entries.
func (c *TTLCache[K, V]) evict() {
	now := time.Now()
	c.entries.Range(func(key K, entry ttlEntry[V]) bool {
		if now.After(entry.expiresAt) {
			c.entries.Delete(key)
		}
		return true
	})
}

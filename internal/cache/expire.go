package cache

import (
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

type Cache[T any] struct {
	f         func() (T, error)
	Timestamp time.Time
	Duration  time.Duration
	Data      T
	mu        sync.Mutex
}

func New[T any](f func() (T, error), duration time.Duration) *Cache[T] {
	return &Cache[T]{
		f:        f,
		Duration: duration,
	}
}

func (c *Cache[T]) GetWithHit() (T, error, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	hit := true
	if c.Timestamp.IsZero() || time.Since(c.Timestamp) > c.Duration {
		hit = false

		var err error
		c.Data, err = c.f()
		if err != nil {
			return c.Data, err, hit
		}
		c.Timestamp = time.Now()
	}
	log.Debug().Bool("hit", hit).Type("data", c.Data).Msg("Cache hit")
	return c.Data, nil, hit
}

func (c *Cache[T]) Get() (T, error) {
	data, err, _ := c.GetWithHit()
	return data, err
}

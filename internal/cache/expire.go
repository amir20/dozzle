package cache

import (
	"time"

	log "github.com/sirupsen/logrus"
)

type Cache[T any] struct {
	f         func() (T, error)
	Timestamp time.Time
	Duration  time.Duration
	Data      T
}

func New[T any](f func() (T, error), duration time.Duration) *Cache[T] {
	return &Cache[T]{
		f:        f,
		Duration: duration,
	}
}

func (c *Cache[T]) GetWithHit() (T, error, bool) {
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
	if hit {
		log.Debugf("Cache hit for %T", c.Data)
	} else {
		log.Debugf("Cache miss for %T", c.Data)
	}
	return c.Data, nil, hit
}

func (c *Cache[T]) Get() (T, error) {
	data, err, _ := c.GetWithHit()
	return data, err
}

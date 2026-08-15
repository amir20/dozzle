package docker

import (
	"context"
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/puzpuzpuz/xsync/v4"
	"github.com/rs/zerolog/log"
)

type DockerStatsCollector struct {
	stream       chan container.ContainerStat
	subscribers  *xsync.Map[context.Context, chan<- container.ContainerStat]
	client       container.Client
	cancelers    *xsync.Map[string, context.CancelFunc]
	stopper      context.CancelFunc
	timer        *time.Timer
	mu           sync.Mutex
	totalStarted atomic.Int32
	labels       container.ContainerLabels
}

var timeToStop = 6 * time.Hour

// Retry bounds for the two Docker streams this collector depends on: a
// container's stats stream and the host's event stream.
//
// Both used to be one-shot. That was survivable when the only subscribers were
// UI tabs — a user looking at a dead chart reloads the page, which resubscribes
// and rebuilds everything. It is not survivable now that Cloud metrics keep a
// permanent subscription: nothing ever resubscribes, so a single transient
// error meant that container (or the whole host) stopped reporting until the
// process restarted, with no signal anywhere that it had.
var (
	streamRetryMin = 1 * time.Second
	streamRetryMax = 30 * time.Second
)

// nextBackoff doubles up to the ceiling.
func nextBackoff(d time.Duration) time.Duration {
	return min(d*2, streamRetryMax)
}

// sleepOrDone waits out the backoff, returning false if ctx ended first.
func sleepOrDone(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

func NewDockerStatsCollector(client container.Client, labels container.ContainerLabels) *DockerStatsCollector {
	return &DockerStatsCollector{
		stream:      make(chan container.ContainerStat),
		subscribers: xsync.NewMap[context.Context, chan<- container.ContainerStat](),
		client:      client,
		cancelers:   xsync.NewMap[string, context.CancelFunc](),
		labels:      labels,
	}
}

func (c *DockerStatsCollector) Subscribe(ctx context.Context, stats chan<- container.ContainerStat) {
	c.subscribers.Store(ctx, stats)
	go func() {
		<-ctx.Done()
		c.subscribers.Delete(ctx)
	}()
}

func (c *DockerStatsCollector) forceStop() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.stopper != nil {
		c.stopper()
		c.stopper = nil
		log.Debug().Str("host", c.client.Host().ID).Msg("stopped container stats collector")
	}
}

func (c *DockerStatsCollector) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.totalStarted.Add(-1) == 0 {
		c.timer = time.AfterFunc(timeToStop, func() {
			c.forceStop()
		})
	}
}

func (c *DockerStatsCollector) reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.timer != nil {
		c.timer.Stop()
	}
	c.timer = nil
}

// streamStats keeps one container's stats flowing for as long as its context
// lives. ContainerStats returning — with an error OR cleanly — means the stream
// broke, not that the container is gone: a container that actually stops
// arrives as a `die` event, which cancels this context. So anything else is
// retried with backoff. Without that, one hiccup left a single container
// silently absent from every subsequent stats window while its neighbours
// carried on, which is indistinguishable from a container that is simply idle.
func streamStats(parent context.Context, sc *DockerStatsCollector, id string) {
	ctx, cancel := context.WithCancel(parent)
	sc.cancelers.Store(id, cancel)

	backoff := streamRetryMin
	for {
		log.Debug().Str("container", id).Str("host", sc.client.Host().Name).Msg("starting to stream stats")
		err := sc.client.ContainerStats(ctx, id, sc.stream)

		// Cancelled = the container died or the collector shut down. Expected,
		// and the only way out of this loop.
		if ctx.Err() != nil {
			log.Debug().Str("container", id).Str("host", sc.client.Host().Name).Msg("stopping to stream stats")
			return
		}

		// Warn, not Debug: this is the failure that used to be permanent, and
		// the whole point of the retry is that someone can see it happening.
		ev := log.Warn()
		if err == nil || errors.Is(err, io.EOF) {
			// A clean end is ordinary (Docker closes idle stats streams), so it
			// does not deserve a warning until it starts repeating.
			ev = log.Debug()
		}
		ev.Str("container", id).
			Str("host", sc.client.Host().Name).
			Err(err).
			Dur("retry_in", backoff).
			Msg("container stats stream ended, retrying")

		if !sleepOrDone(ctx, backoff) {
			return
		}
		backoff = nextBackoff(backoff)
	}
}

// Start starts the stats collector and blocks until it's stopped. It returns true if the collector was stopped, false if it was already running
func (sc *DockerStatsCollector) Start(parentCtx context.Context) bool {
	sc.reset()
	sc.totalStarted.Add(1)

	sc.mu.Lock()
	if sc.stopper != nil {
		sc.mu.Unlock()
		return false
	}
	var ctx context.Context
	ctx, sc.stopper = context.WithCancel(parentCtx)
	sc.mu.Unlock()

	timeoutCtx, cancel := context.WithTimeout(parentCtx, 3*time.Second) // 3 seconds to list containers is hard limit
	if containers, err := sc.client.ListContainers(timeoutCtx, sc.labels); err == nil {
		for _, c := range containers {
			if c.State == "running" {
				go streamStats(ctx, sc, c.ID)
			}
		}
	} else {
		log.Error().Str("host", sc.client.Host().Name).Err(err).Msg("failed to list containers")
	}
	cancel()

	events := make(chan container.ContainerEvent)

	// The event stream is how new containers get a stats stream and dead ones
	// lose theirs, so losing it degrades the collector even while the existing
	// per-container streams keep running. It used to forceStop() the whole
	// collector on any error, which bypasses the reference count entirely:
	// every subscriber — including a Cloud connection that is never coming back
	// to resubscribe — went silent at once and was never told. Now it
	// reconnects, and only a cancelled context ends it.
	go func() {
		defer close(events)
		backoff := streamRetryMin
		for {
			log.Debug().Str("host", sc.client.Host().Name).Msg("starting to listen to docker events")
			err := sc.client.ContainerEvents(ctx, events)
			if ctx.Err() != nil {
				return
			}
			log.Warn().
				Str("host", sc.client.Host().Name).
				Err(err).
				Dur("retry_in", backoff).
				Msg("docker event stream ended, retrying")
			if !sleepOrDone(ctx, backoff) {
				return
			}
			backoff = nextBackoff(backoff)
		}
	}()

	go func() {
		for event := range events {
			switch event.Name {
			case "start":
				// Replace any stream still retrying for this id. Without this,
				// a start that arrives without a matching die (a restart, a
				// duplicate event) would leave two loops feeding sc.stream for
				// one container, doubling its samples.
				if cancel, ok := sc.cancelers.LoadAndDelete(event.ActorID); ok {
					cancel()
				}
				go streamStats(ctx, sc, event.ActorID)

			case "die":
				if cancel, ok := sc.cancelers.LoadAndDelete(event.ActorID); ok {
					cancel()
				}
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			log.Info().Str("host", sc.client.Host().Name).Msg("stopped container stats collector")
			return true
		case stat := <-sc.stream:
			sc.subscribers.Range(func(c context.Context, stats chan<- container.ContainerStat) bool {
				select {
				case stats <- stat:
				case <-c.Done():
					sc.subscribers.Delete(c)
				}
				return true
			})
		}
	}
}

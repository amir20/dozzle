package docker

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/puzpuzpuz/xsync/v4"
	"github.com/rs/zerolog/log"
)

type StatsCollector struct {
	stream      chan container.ContainerStat
	subscribers *xsync.Map[context.Context, chan<- container.ContainerStat]
	client      container.Client
	cancelers   *xsync.Map[string, context.CancelFunc]
	lifecycle   container.CollectorLifecycle
	labels      container.ContainerLabels
	retryMin    time.Duration
	retryMax    time.Duration
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
//
// They live on the collector rather than as package globals so tests can shrink
// them for one instance without writing to state other running collectors read.
const (
	streamRetryMin = 1 * time.Second
	streamRetryMax = 30 * time.Second
)

func NewStatsCollector(client container.Client, labels container.ContainerLabels) *StatsCollector {
	return &StatsCollector{
		stream:      make(chan container.ContainerStat),
		subscribers: xsync.NewMap[context.Context, chan<- container.ContainerStat](),
		client:      client,
		cancelers:   xsync.NewMap[string, context.CancelFunc](),
		labels:      labels,
		retryMin:    streamRetryMin,
		retryMax:    streamRetryMax,
	}
}

func (c *StatsCollector) Subscribe(ctx context.Context, stats chan<- container.ContainerStat) {
	c.subscribers.Store(ctx, stats)
	go func() {
		<-ctx.Done()
		c.subscribers.Delete(ctx)
	}()
}

func (c *StatsCollector) Stop() {
	c.lifecycle.Release(timeToStop)
}

// startStream starts a stats stream for id, replacing any stream still retrying
// for it. Without the replace, a start that arrives without a matching die (a
// restart, a duplicate event) would leave two loops feeding sc.stream for one
// container, doubling its samples. The canceler is stored before the goroutine
// starts so a resync right after cannot miss it and start a second one.
func (sc *StatsCollector) startStream(parent context.Context, id string) {
	if cancel, ok := sc.cancelers.LoadAndDelete(id); ok {
		cancel()
	}
	ctx, cancel := context.WithCancel(parent)
	sc.cancelers.Store(id, cancel)
	go streamStats(ctx, sc, id)
}

// streamRunning lists the running containers and starts a stream for each one
// that has none. It runs at start and again whenever the event stream
// reconnects, since a container started while it was down sent its `start`
// event to nobody.
func (sc *StatsCollector) streamRunning(listCtx, streamCtx context.Context) {
	timeoutCtx, cancel := context.WithTimeout(listCtx, 3*time.Second) // 3 seconds to list containers is hard limit
	defer cancel()
	containers, err := sc.client.ListContainers(timeoutCtx, sc.labels)
	if err != nil {
		log.Error().Str("host", sc.client.Host().Name).Err(err).Msg("failed to list containers")
		return
	}
	for _, c := range containers {
		if c.State != "running" {
			continue
		}
		if _, ok := sc.cancelers.Load(c.ID); !ok {
			sc.startStream(streamCtx, c.ID)
		}
	}
}

// streamStats keeps one container's stats flowing for as long as its context
// lives. ContainerStats returning — with an error OR cleanly — means the stream
// broke, not that the container is gone: a container that actually stops
// arrives as a `die` event, which cancels this context. So anything else is
// retried with backoff. Without that, one hiccup left a single container
// silently absent from every subsequent stats window while its neighbours
// carried on, which is indistinguishable from a container that is simply idle.
func streamStats(ctx context.Context, sc *StatsCollector, id string) {

	backoff := sc.retryMin
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

		if !container.SleepOrDone(ctx, backoff) {
			return
		}
		backoff = container.NextBackoff(backoff, sc.retryMax)
	}
}

// Start starts the stats collector and blocks until it's stopped. It returns true if the collector was stopped, false if it was already running
func (sc *StatsCollector) Start(parentCtx context.Context) bool {
	ctx, run := sc.lifecycle.Acquire(parentCtx, timeToStop)
	if !run {
		return false
	}

	// Entries left from a previous run hold cancels for a context that already
	// ended. Kept, they would make streamRunning skip every container they name.
	sc.cancelers.Clear()
	sc.streamRunning(parentCtx, ctx)

	events := make(chan container.ContainerEvent)
	// resync is how the reconnect loop asks the event goroutine to re-list.
	// Doing the list there keeps it serialized with start and die events.
	resync := make(chan struct{}, 1)

	// The event stream is how new containers get a stats stream and dead ones
	// lose theirs, so losing it degrades the collector even while the existing
	// per-container streams keep running. It used to forceStop() the whole
	// collector on any error, which bypasses the reference count entirely:
	// every subscriber — including a Cloud connection that is never coming back
	// to resubscribe — went silent at once and was never told. Now it
	// reconnects, and only a cancelled context ends it.
	go func() {
		defer close(events)
		backoff := sc.retryMin
		for {
			log.Debug().Str("host", sc.client.Host().Name).Msg("starting to listen to docker events")
			err := sc.client.ContainerEvents(ctx, events)
			if ctx.Err() != nil {
				return
			}
			// A clean exit is ordinary, so it only gets a warning once it starts
			// repeating for a real reason.
			ev := log.Warn()
			if err == nil {
				ev = log.Debug()
			}
			ev.
				Str("host", sc.client.Host().Name).
				Err(err).
				Dur("retry_in", backoff).
				Msg("docker event stream ended, retrying")
			if !container.SleepOrDone(ctx, backoff) {
				return
			}
			backoff = container.NextBackoff(backoff, sc.retryMax)
			select {
			case resync <- struct{}{}:
			default:
			}
		}
	}()

	go func() {
		for {
			select {
			case <-resync:
				sc.streamRunning(ctx, ctx)
			case event, ok := <-events:
				if !ok {
					return
				}
				switch event.Name {
				case "start":
					sc.startStream(ctx, event.ActorID)

				case "die":
					if cancel, ok := sc.cancelers.LoadAndDelete(event.ActorID); ok {
						cancel()
					}
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

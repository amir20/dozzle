package container

import (
	"context"
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/puzpuzpuz/xsync/v3"
	"github.com/rs/zerolog/log"
)

type StatsCollector struct {
	stream       chan ContainerStat
	subscribers  *xsync.MapOf[context.Context, chan<- ContainerStat]
	client       Client
	cancelers    *xsync.MapOf[string, context.CancelFunc]
	stopper      context.CancelFunc
	timer        *time.Timer
	mu           sync.Mutex
	totalStarted atomic.Int32
	labels       ContainerLabels
}

var timeToStop = 6 * time.Hour

func NewStatsCollector(client Client, labels ContainerLabels) *StatsCollector {
	return &StatsCollector{
		stream:      make(chan ContainerStat),
		subscribers: xsync.NewMapOf[context.Context, chan<- ContainerStat](),
		client:      client,
		cancelers:   xsync.NewMapOf[string, context.CancelFunc](),
		labels:      labels,
	}
}

func (c *StatsCollector) Subscribe(ctx context.Context, stats chan<- ContainerStat) {
	c.subscribers.Store(ctx, stats)
	go func() {
		<-ctx.Done()
		c.subscribers.Delete(ctx)
	}()
}

func (c *StatsCollector) forceStop() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.stopper != nil {
		c.stopper()
		c.stopper = nil
		log.Debug().Str("host", c.client.Host().ID).Msg("stopped container stats collector")
	}
}

func (c *StatsCollector) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.totalStarted.Add(-1) == 0 {
		c.timer = time.AfterFunc(timeToStop, func() {
			c.forceStop()
		})
	}
}

func (c *StatsCollector) reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.timer != nil {
		c.timer.Stop()
	}
	c.timer = nil
}

func streamStats(parent context.Context, sc *StatsCollector, id string) {
	ctx, cancel := context.WithCancel(parent)
	sc.cancelers.Store(id, cancel)
	log.Debug().Str("container", id).Str("host", sc.client.Host().Name).Msg("starting to stream stats")
	if err := sc.client.ContainerStats(ctx, id, sc.stream); err != nil {
		log.Debug().Str("container", id).Str("host", sc.client.Host().Name).Err(err).Msg("stopping to stream stats")
		if !errors.Is(err, context.Canceled) && !errors.Is(err, io.EOF) {
			log.Error().Str("container", id).Str("host", sc.client.Host().Name).Err(err).Msg("unexpected error while streaming stats")
		}
	}
}

// Start starts the stats collector and blocks until it's stopped. It returns true if the collector was stopped, false if it was already running
func (sc *StatsCollector) Start(parentCtx context.Context) bool {
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

	timeoutCtx, cancel := context.WithTimeout(parentCtx, defaultTimeout)
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

	events := make(chan ContainerEvent)

	go func() {
		log.Debug().Str("host", sc.client.Host().Name).Msg("starting to listen to docker events")
		err := sc.client.ContainerEvents(context.Background(), events)
		if !errors.Is(err, context.Canceled) {
			log.Error().Str("host", sc.client.Host().Name).Err(err).Msg("unexpected error while listening to docker events")
		}
		sc.forceStop()
	}()

	go func() {
		for event := range events {
			switch event.Name {
			case "start":
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
			sc.subscribers.Range(func(c context.Context, stats chan<- ContainerStat) bool {
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

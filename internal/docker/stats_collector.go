package docker

import (
	"context"
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/puzpuzpuz/xsync/v3"
	log "github.com/sirupsen/logrus"
)

type StatsCollector struct {
	stream       chan ContainerStat
	subscribers  *xsync.MapOf[context.Context, chan ContainerStat]
	client       Client
	cancelers    *xsync.MapOf[string, context.CancelFunc]
	stopper      context.CancelFunc
	timer        *time.Timer
	mu           sync.Mutex
	totalStarted atomic.Int32
}

var timeToStop = 6 * time.Hour

func NewStatsCollector(client Client) *StatsCollector {
	return &StatsCollector{
		stream:      make(chan ContainerStat),
		subscribers: xsync.NewMapOf[context.Context, chan ContainerStat](),
		client:      client,
		cancelers:   xsync.NewMapOf[string, context.CancelFunc](),
	}
}

func (c *StatsCollector) Subscribe(ctx context.Context, stats chan ContainerStat) {
	c.subscribers.Store(ctx, stats)
}

func (c *StatsCollector) forceStop() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.stopper != nil {
		c.stopper()
		c.stopper = nil
		log.Debug("stopping container stats collector due to inactivity")
	}
}

func (c *StatsCollector) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.totalStarted.Add(-1) == 0 {
		log.Debug("scheduled to stop container stats collector")
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

// Start starts the stats collector and blocks until it's stopped. It returns true if the collector was stopped, false if it was already running
func (sc *StatsCollector) Start(ctx context.Context) bool {
	sc.reset()
	if sc.totalStarted.Add(1) > 1 {
		return false
	}
	sc.mu.Lock()
	ctx, sc.stopper = context.WithCancel(ctx)
	sc.mu.Unlock()

	if containers, err := sc.client.ListContainers(); err == nil {
		for _, c := range containers {
			if c.State == "running" {
				go func(client Client, id string) {
					ctx, cancel := context.WithCancel(ctx)
					sc.cancelers.Store(id, cancel)
					if err := client.ContainerStats(ctx, id, sc.stream); err != nil {
						if !errors.Is(err, context.Canceled) && !errors.Is(err, io.EOF) {
							log.Errorf("unexpected error when streaming container stats: %v", err)
						}
					}
				}(sc.client, c.ID)
			}
		}
	} else {
		log.Errorf("error while listing containers: %v", err)
	}

	go func() {
		events := make(chan ContainerEvent)
		sc.client.Events(ctx, events)
		for event := range events {
			switch event.Name {
			case "start":
				go func(client Client, id string) {
					ctx, cancel := context.WithCancel(ctx)
					sc.cancelers.Store(id, cancel)
					if err := client.ContainerStats(ctx, id, sc.stream); err != nil {
						if !errors.Is(err, context.Canceled) && !errors.Is(err, io.EOF) {
							log.Errorf("unexpected error when streaming container stats: %v", err)
						}
					}
				}(sc.client, event.ActorID)

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
			log.Info("stopped collecting container stats")
			return true
		case stat := <-sc.stream:
			sc.subscribers.Range(func(c context.Context, stats chan ContainerStat) bool {
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

package docker

import (
	"context"
	"errors"
	"io"

	"github.com/puzpuzpuz/xsync/v3"
	log "github.com/sirupsen/logrus"
)

type StatsCollector struct {
	stream      chan ContainerStat
	subscribers *xsync.MapOf[context.Context, chan ContainerStat]
	client      Client
	cancelers   *xsync.MapOf[string, context.CancelFunc]
}

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

func (sc *StatsCollector) StartCollecting(ctx context.Context) {
	events := make(chan ContainerEvent)
	sc.client.Events(ctx, events)

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

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
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
	}()
}

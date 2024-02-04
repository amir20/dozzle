package docker

import (
	"context"
	"errors"
	"io"
	"sync"

	log "github.com/sirupsen/logrus"
)

type StatsCollector struct {
	stream      chan ContainerStat
	subscribers sync.Map
	client      Client
	cancelers   sync.Map
}

func NewStatsCollector(client Client) *StatsCollector {
	return &StatsCollector{
		stream:      make(chan ContainerStat),
		subscribers: sync.Map{},
		client:      client,
		cancelers:   sync.Map{},
	}
}

func (c *StatsCollector) Subscribe(ctx context.Context, stats chan ContainerStat) {
	c.subscribers.Store(ctx, stats)
}

func (sc *StatsCollector) StartCollecting(ctx context.Context) {
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
					cancel.(context.CancelFunc)()
				}
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case stat := <-sc.stream:
			sc.subscribers.Range(func(key, value interface{}) bool {
				select {
				case value.(chan ContainerStat) <- stat:
				case <-key.(context.Context).Done():
					sc.subscribers.Delete(key)
				}
				return true
			})
		}
	}
}

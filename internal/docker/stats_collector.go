package docker

import (
	"context"
	"errors"
	"io"

	log "github.com/sirupsen/logrus"
)

type StatsCollector struct {
	stream      chan ContainerStat
	subscribers map[context.Context]chan ContainerStat
	client      Client
	cancelers   map[string]context.CancelFunc
}

func NewStatsCollector(client Client) *StatsCollector {
	return &StatsCollector{
		stream:      make(chan ContainerStat),
		subscribers: make(map[context.Context]chan ContainerStat),
		client:      client,
		cancelers:   make(map[string]context.CancelFunc),
	}
}

func (c *StatsCollector) Subscribe(ctx context.Context, stats chan ContainerStat) {
	c.subscribers[ctx] = stats
}

func (sc *StatsCollector) StartCollecting(ctx context.Context) {
	if containers, err := sc.client.ListContainers(); err == nil {
		for _, c := range containers {
			if c.State == "running" {
				go func(client Client, id string) {
					ctx, cancel := context.WithCancel(ctx)
					sc.cancelers[id] = cancel
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
					if err := client.ContainerStats(ctx, id, sc.stream); err != nil {
						if !errors.Is(err, context.Canceled) && !errors.Is(err, io.EOF) {
							log.Errorf("unexpected error when streaming container stats: %v", err)
						}
					}
				}(sc.client, event.ActorID)

			case "die":
				if cancel, ok := sc.cancelers[event.ActorID]; ok {
					cancel()
					delete(sc.cancelers, event.ActorID)
				}
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case stat := <-sc.stream:
			for c, sub := range sc.subscribers {
				select {
				case sub <- stat:
				case <-c.Done():
					delete(sc.subscribers, c)
				}
			}
		}
	}
}

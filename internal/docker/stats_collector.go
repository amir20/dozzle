package docker

import (
	"context"
	"errors"
	"io"

	log "github.com/sirupsen/logrus"
)

type StatsCollector struct {
	stream      chan ContainerStat
	subscribers []chan ContainerStat
	client      Client
}

func NewStatsCollector(client Client) *StatsCollector {
	return &StatsCollector{
		stream:      make(chan ContainerStat),
		subscribers: []chan ContainerStat{},
		client:      client,
	}
}

func (c *StatsCollector) Subscribe(stats chan ContainerStat) {
	c.subscribers = append(c.subscribers, stats)
}

func (c *StatsCollector) Unsubscribe(subscriber chan ContainerStat) {
	for i, s := range c.subscribers {
		if s == subscriber {
			c.subscribers = append(c.subscribers[:i], c.subscribers[i+1:]...)
			close(s)
			break
		}
	}
}

func (sc *StatsCollector) StartCollecting(ctx context.Context) {
	if containers, err := sc.client.ListContainers(); err == nil {
		for _, c := range containers {
			if c.State == "running" {
				go func(client Client, id string) {
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
			if event.Name == "start" {
				go func(client Client, id string) {
					if err := client.ContainerStats(ctx, id, sc.stream); err != nil {
						if !errors.Is(err, context.Canceled) && !errors.Is(err, io.EOF) {
							log.Errorf("unexpected error when streaming container stats: %v", err)
						}
					}
				}(sc.client, event.ActorID)
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case stat := <-sc.stream:
			for _, subscriber := range sc.subscribers {
				subscriber <- stat
			}
		}
	}
}

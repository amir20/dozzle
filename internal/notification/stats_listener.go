package notification

import (
	"context"
	"fmt"

	"github.com/amir20/dozzle/internal/container"
	container_support "github.com/amir20/dozzle/internal/support/container"
	"github.com/rs/zerolog/log"
)

// ContainerStatsListener subscribes to container stats from all clients and forwards them to a channel
type ContainerStatsListener struct {
	clients     []container_support.ClientService
	statChannel chan container.ContainerStat
	ctx         context.Context
}

// NewContainerStatsListener creates a new listener for container stats
func NewContainerStatsListener(ctx context.Context, clients []container_support.ClientService) *ContainerStatsListener {
	return &ContainerStatsListener{
		clients:     clients,
		statChannel: make(chan container.ContainerStat, 1000),
		ctx:         ctx,
	}
}

// Start subscribes to events (to trigger stats collector start) and stats from all clients
func (l *ContainerStatsListener) Start() {
	for _, client := range l.clients {
		// Subscribe to events to ensure the stats collector is started
		// (stats collector only runs when there are event subscribers)
		dummyEvents := make(chan container.ContainerEvent, 100)
		client.SubscribeEvents(l.ctx, dummyEvents)
		go func() {
			for {
				select {
				case <-l.ctx.Done():
					return
				case _, ok := <-dummyEvents:
					if !ok {
						return
					}
				}
			}
		}()

		client.SubscribeStats(l.ctx, l.statChannel)
		log.Debug().Msg("Subscribed to container stats for metric alerts")
	}
}

// StatChannel returns the channel for stat events
func (l *ContainerStatsListener) StatChannel() <-chan container.ContainerStat {
	return l.statChannel
}

// FindContainerWithHost finds a container and its host by container ID
func (l *ContainerStatsListener) FindContainerWithHost(ctx context.Context, containerID string) (container.Container, container.Host, error) {
	for _, client := range l.clients {
		c, err := client.FindContainer(ctx, containerID, nil)
		if err != nil {
			continue
		}
		host, err := client.Host(ctx)
		if err != nil {
			continue
		}
		return c, host, nil
	}
	return container.Container{}, container.Host{}, fmt.Errorf("container %s not found in any client", containerID)
}

package notification

import (
	"context"
	"fmt"

	"github.com/amir20/dozzle/internal/container"
	container_support "github.com/amir20/dozzle/internal/support/container"
	"github.com/rs/zerolog/log"
)

// ContainerStatsListener subscribes to container stats from all clients and forwards them to a channel.
// It handles subscribing to both events (to start the stats collector) and stats internally.
type ContainerStatsListener struct {
	clients     []container_support.ClientService
	statChannel chan container.ContainerStat
	ctx         context.Context
}

// NewContainerStatsListener creates a new listener that subscribes to stats from the given clients.
func NewContainerStatsListener(ctx context.Context, clients []container_support.ClientService) *ContainerStatsListener {
	l := &ContainerStatsListener{
		clients:     clients,
		statChannel: make(chan container.ContainerStat, 1000),
		ctx:         ctx,
	}

	for _, client := range clients {
		client.SubscribeStats(ctx, l.statChannel)
	}

	log.Debug().Msg("Subscribed to container stats for metric alerts")

	return l
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

package notification

import (
	"context"
	"time"

	"github.com/amir20/dozzle/internal/container"
	container_support "github.com/amir20/dozzle/internal/support/container"
	"github.com/rs/zerolog/log"
)

// ContainerStatEvent pairs a stat with its resolved container and host metadata.
type ContainerStatEvent struct {
	Stat      container.ContainerStat
	Container container.Container
	Host      container.Host
}

// ContainerStatsListener subscribes to container stats from all clients,
// enriches each stat with container and host metadata, and forwards them to a channel.
type ContainerStatsListener struct {
	clients []container_support.ClientService
	channel chan ContainerStatEvent
	ctx     context.Context
}

// NewContainerStatsListener creates a new listener that subscribes to stats from the given clients.
func NewContainerStatsListener(ctx context.Context, clients []container_support.ClientService) *ContainerStatsListener {
	l := &ContainerStatsListener{
		clients: clients,
		channel: make(chan ContainerStatEvent, 1000),
		ctx:     ctx,
	}

	rawStats := make(chan container.ContainerStat, 1000)
	for _, client := range clients {
		client.SubscribeStats(ctx, rawStats)
	}

	go l.enrich(rawStats)

	log.Debug().Msg("Subscribed to container stats for metric alerts")

	return l
}

// enrich reads raw stats, resolves container+host, and sends enriched events.
func (l *ContainerStatsListener) enrich(rawStats <-chan container.ContainerStat) {
	for {
		select {
		case <-l.ctx.Done():
			return
		case stat := <-rawStats:
			ctx, cancel := context.WithTimeout(l.ctx, 5*time.Second)
			c, host, err := l.findContainerWithHost(ctx, stat.ID)
			cancel()
			if err != nil {
				continue
			}

			// Skip stats from Dozzle's own containers
			if isDozzleContainer(c) {
				continue
			}

			select {
			case l.channel <- ContainerStatEvent{Stat: stat, Container: c, Host: host}:
			case <-l.ctx.Done():
				return
			}
		}
	}
}

// Channel returns the channel for enriched stat events.
func (l *ContainerStatsListener) Channel() <-chan ContainerStatEvent {
	return l.channel
}

func (l *ContainerStatsListener) findContainerWithHost(ctx context.Context, containerID string) (container.Container, container.Host, error) {
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
	return container.Container{}, container.Host{}, container.ErrContainerNotFound
}

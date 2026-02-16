package notification

import (
	"context"
	"sync"
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

type containerInfo struct {
	container container.Container
	host      container.Host
}

// ContainerStatsListener subscribes to container stats from all clients,
// enriches each stat with container and host metadata, and forwards them to a channel.
type ContainerStatsListener struct {
	clients    []container_support.ClientService
	channel    chan *ContainerStatEvent
	parentCtx  context.Context
	cache      *TTLCache[string, containerInfo]
	mu         sync.Mutex
	cancelFunc context.CancelFunc
}

// NewContainerStatsListener creates a new listener that can subscribe to stats from the given clients.
// Call Start() to begin receiving stats.
func NewContainerStatsListener(ctx context.Context, clients []container_support.ClientService) *ContainerStatsListener {
	return &ContainerStatsListener{
		clients:   clients,
		channel:   make(chan *ContainerStatEvent, 1000),
		parentCtx: ctx,
		cache:     NewTTLCache[string, containerInfo](ctx, 30*time.Second),
	}
}

// Start subscribes to stats from all clients and begins enriching events.
// No-op if already running.
func (l *ContainerStatsListener) Start() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.cancelFunc != nil {
		return
	}

	ctx, cancel := context.WithCancel(l.parentCtx)
	l.cancelFunc = cancel

	rawStats := make(chan container.ContainerStat, 1000)
	for _, client := range l.clients {
		client.SubscribeStats(ctx, rawStats)
	}

	go l.enrich(ctx, rawStats)

	log.Debug().Msg("Started container stats listener for metric alerts")
}

// Stop unsubscribes from all clients' stats by cancelling the subscription context.
// No-op if not running.
func (l *ContainerStatsListener) Stop() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.cancelFunc == nil {
		return
	}
	l.cancelFunc()
	l.cancelFunc = nil
	log.Debug().Msg("Stopped container stats listener for metric alerts")
}

// IsRunning returns whether the listener is currently subscribed to stats.
func (l *ContainerStatsListener) IsRunning() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.cancelFunc != nil
}

// enrich reads raw stats, resolves container+host, and sends enriched events.
func (l *ContainerStatsListener) enrich(ctx context.Context, rawStats <-chan container.ContainerStat) {
	for {
		select {
		case <-ctx.Done():
			return
		case stat, ok := <-rawStats:
			if !ok {
				return
			}
			c, host, err := l.resolveContainer(stat.ID)
			if err != nil {
				continue
			}

			// Skip stats from Dozzle's own containers
			if isDozzleContainer(c) {
				continue
			}

			select {
			case l.channel <- &ContainerStatEvent{Stat: stat, Container: c, Host: host}:
			case <-ctx.Done():
				return
			default:
				log.Warn().Str("containerID", stat.ID).Msg("Metric stats channel full, dropping stat event")
			}
		}
	}
}

// resolveContainer looks up container+host, using a TTL cache to avoid repeated API calls.
func (l *ContainerStatsListener) resolveContainer(containerID string) (container.Container, container.Host, error) {
	if cached, ok := l.cache.Load(containerID); ok {
		return cached.container, cached.host, nil
	}

	ctx, cancel := context.WithTimeout(l.parentCtx, 5*time.Second)
	defer cancel()

	c, host, err := l.findContainerWithHost(ctx, containerID)
	if err != nil {
		return c, host, err
	}

	l.cache.Store(containerID, containerInfo{
		container: c,
		host:      host,
	})

	return c, host, nil
}

// Channel returns the channel for enriched stat events.
func (l *ContainerStatsListener) Channel() <-chan *ContainerStatEvent {
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

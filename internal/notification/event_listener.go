package notification

import (
	"context"
	"sync"
	"time"

	"github.com/amir20/dozzle/internal/container"
	container_support "github.com/amir20/dozzle/internal/support/container"
	"github.com/rs/zerolog/log"
)

var allowedEventNames = map[string]bool{
	"start":         true,
	"stop":          true,
	"die":           true,
	"restart":       true,
	"health_status": true,
}

type ContainerEventEntry struct {
	Event     container.ContainerEvent
	Container container.Container
	Host      container.Host
}

type ContainerEventListener struct {
	clients    []container_support.ClientService
	channel    chan *ContainerEventEntry
	parentCtx  context.Context
	cache      *TTLCache[string, containerInfo]
	mu         sync.Mutex
	cancelFunc context.CancelFunc
}

func NewContainerEventListener(ctx context.Context, clients []container_support.ClientService) *ContainerEventListener {
	return &ContainerEventListener{
		clients:   clients,
		channel:   make(chan *ContainerEventEntry, 1000),
		parentCtx: ctx,
		cache:     NewTTLCache[string, containerInfo](ctx, 30*time.Second),
	}
}

func (l *ContainerEventListener) Start() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.cancelFunc != nil {
		return
	}

	ctx, cancel := context.WithCancel(l.parentCtx)
	l.cancelFunc = cancel

	rawEvents := make(chan container.ContainerEvent, 1000)
	for _, client := range l.clients {
		client.SubscribeEvents(ctx, rawEvents)
	}

	go l.enrich(ctx, rawEvents)

	log.Debug().Msg("Started container event listener for event alerts")
}

func (l *ContainerEventListener) Stop() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.cancelFunc == nil {
		return
	}
	l.cancelFunc()
	l.cancelFunc = nil
	log.Debug().Msg("Stopped container event listener for event alerts")
}

func (l *ContainerEventListener) IsRunning() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.cancelFunc != nil
}

func (l *ContainerEventListener) enrich(ctx context.Context, rawEvents <-chan container.ContainerEvent) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-rawEvents:
			if !ok {
				return
			}

			if !allowedEventNames[event.Name] {
				continue
			}

			c, host, err := l.resolveContainer(event.ActorID)
			if err != nil {
				continue
			}

			if isDozzleContainer(c) {
				continue
			}

			select {
			case l.channel <- &ContainerEventEntry{Event: event, Container: c, Host: host}:
			case <-ctx.Done():
				return
			default:
				log.Warn().Str("containerID", event.ActorID).Str("event", event.Name).Msg("Event channel full, dropping event")
			}
		}
	}
}

func (l *ContainerEventListener) resolveContainer(containerID string) (container.Container, container.Host, error) {
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

func (l *ContainerEventListener) Channel() <-chan *ContainerEventEntry {
	return l.channel
}

func (l *ContainerEventListener) findContainerWithHost(ctx context.Context, containerID string) (container.Container, container.Host, error) {
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

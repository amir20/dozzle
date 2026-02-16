package notification

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/amir20/dozzle/internal/container"
	container_support "github.com/amir20/dozzle/internal/support/container"
	"github.com/puzpuzpuz/xsync/v4"
	"github.com/rs/zerolog/log"
)

// ContainerMatcher is an interface for checking if a container should be listened to
type ContainerMatcher interface {
	ShouldListenToContainer(c container.Container) bool
}

// ContainerLogListener manages active log streams for containers across multiple clients
type ContainerLogListener struct {
	clients          []container_support.ClientService
	containerClients *xsync.Map[string, container_support.ClientService] // containerID -> owning client
	activeStreams    *xsync.Map[string, context.CancelFunc]              // containerID -> cancel function
	matcher          ContainerMatcher
	logChannel       chan *container.LogEvent
	ctx              context.Context
	cache            *xsync.Map[string, cachedContainerInfo]
}

// NewContainerLogListener creates a new listener for multiple clients
func NewContainerLogListener(ctx context.Context, clients []container_support.ClientService) *ContainerLogListener {
	return &ContainerLogListener{
		clients:          clients,
		containerClients: xsync.NewMap[string, container_support.ClientService](),
		activeStreams:    xsync.NewMap[string, context.CancelFunc](),
		logChannel:       make(chan *container.LogEvent, 1000),
		ctx:              ctx,
		cache:            xsync.NewMap[string, cachedContainerInfo](),
	}
}

// Start begins listening for container events and processes log streams
func (l *ContainerLogListener) Start(matcher ContainerMatcher) error {
	l.matcher = matcher

	// Subscribe to new containers from all clients
	containerChan := make(chan container.Container, 10)

	// Get all current containers from all clients
	for _, client := range l.clients {
		containers, err := client.ListContainers(l.ctx, nil)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to list containers from client")
			continue
		}

		// Start listening to containers that match
		for _, c := range containers {
			if l.matcher.ShouldListenToContainer(c) {
				l.startListening(c, client)
			}
		}

		// Subscribe to new containers from this client
		client.SubscribeContainersStarted(l.ctx, containerChan)
	}

	go func() {
		for {
			select {
			case <-l.ctx.Done():
				return
			case c := <-containerChan:
				if l.matcher.ShouldListenToContainer(c) {
					l.startListeningByID(c)
				}
			}
		}
	}()

	return nil
}

// UpdateStreams updates which containers to listen to based on current matcher rules
func (l *ContainerLogListener) UpdateStreams() {
	// Get all current containers from all clients
	for _, client := range l.clients {
		containers, err := client.ListContainers(l.ctx, nil)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to list containers from client")
			continue
		}

		// Check each container against matcher
		for _, c := range containers {
			shouldListen := l.matcher.ShouldListenToContainer(c)
			isListening := l.isListening(c.ID)

			if shouldListen && !isListening {
				l.startListening(c, client)
			} else if !shouldListen && isListening {
				l.stopListening(c.ID)
			}
		}
	}
}

// startListening starts listening to a container's logs with a known client
func (l *ContainerLogListener) startListening(c container.Container, client container_support.ClientService) {
	streamCtx, cancel := context.WithCancel(l.ctx)

	// Only store if not already present
	_, loaded := l.activeStreams.LoadOrStore(c.ID, cancel)
	if loaded {
		cancel() // Already listening, cancel the new context
		return
	}

	l.containerClients.Store(c.ID, client)

	go func() {
		log.Debug().Str("containerID", c.ID).Str("name", c.Name).Msg("Started listening to container")
		if err := client.StreamLogs(streamCtx, c, time.Now(), container.STDALL, l.logChannel); err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, context.Canceled) {
			log.Error().Err(err).Str("containerID", c.ID).Msg("Error streaming logs")
		}
	}()
}

// startListeningByID finds the client for a container and starts listening
func (l *ContainerLogListener) startListeningByID(c container.Container) {
	for _, client := range l.clients {
		if found, err := client.FindContainer(l.ctx, c.ID, nil); err == nil {
			l.startListening(found, client)
			return
		}
	}
	log.Warn().Str("containerID", c.ID).Msg("Could not find client for container")
}

// stopListening stops listening to a container's logs
func (l *ContainerLogListener) stopListening(containerID string) {
	if cancel, exists := l.activeStreams.LoadAndDelete(containerID); exists {
		cancel()
		l.containerClients.Delete(containerID)
		log.Debug().Str("containerID", containerID).Msg("Stopped listening to container")
	}
}

// isListening returns true if listening to a container
func (l *ContainerLogListener) isListening(containerID string) bool {
	_, exists := l.activeStreams.Load(containerID)
	return exists
}

// FindContainer finds a container by ID using the client that owns it
func (l *ContainerLogListener) FindContainer(ctx context.Context, id string, labels container.ContainerLabels) (container.Container, error) {
	client, exists := l.containerClients.Load(id)
	if !exists {
		return container.Container{}, fmt.Errorf("container %s not found in any client", id)
	}

	return client.FindContainer(ctx, id, labels)
}

// FindContainerWithHost finds a container and its host by container ID, using a TTL cache.
func (l *ContainerLogListener) FindContainerWithHost(ctx context.Context, id string, labels container.ContainerLabels) (container.Container, container.Host, error) {
	if cached, ok := l.cache.Load(id); ok && time.Now().Before(cached.expiresAt) {
		return cached.container, cached.host, nil
	}

	client, exists := l.containerClients.Load(id)
	if !exists {
		return container.Container{}, container.Host{}, fmt.Errorf("container %s not found in any client", id)
	}

	c, err := client.FindContainer(ctx, id, labels)
	if err != nil {
		return container.Container{}, container.Host{}, err
	}

	host, err := client.Host(ctx)
	if err != nil {
		return container.Container{}, container.Host{}, fmt.Errorf("failed to get host for container %s: %w", id, err)
	}

	l.cache.Store(id, cachedContainerInfo{
		container: c,
		host:      host,
		expiresAt: time.Now().Add(30 * time.Second),
	})

	return c, host, nil
}

// LogChannel returns the channel for log events
func (l *ContainerLogListener) LogChannel() <-chan *container.LogEvent {
	return l.logChannel
}

// ListContainers returns all containers from all clients
func (l *ContainerLogListener) ListContainers() []container.Container {
	var result []container.Container
	for _, client := range l.clients {
		containers, err := client.ListContainers(l.ctx, nil)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to list containers from client")
			continue
		}
		result = append(result, containers...)
	}
	return result
}

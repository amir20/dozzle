package notification

import (
	"context"
	"sync"
	"time"

	"github.com/amir20/dozzle/internal/container"
	container_support "github.com/amir20/dozzle/internal/support/container"
	"github.com/rs/zerolog/log"
)

// ContainerMatcher is an interface for checking if a container should be listened to
type ContainerMatcher interface {
	ShouldListenToContainer(c container.Container) bool
}

// ContainerLogListener manages active log streams for containers
type ContainerLogListener struct {
	clientService container_support.ClientService
	matcher       ContainerMatcher
	activeStreams map[string]context.CancelFunc // containerID -> cancel function
	logChannel    chan *container.LogEvent
	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
}

// NewContainerLogListener creates a new listener
func NewContainerLogListener(clientService container_support.ClientService, matcher ContainerMatcher) *ContainerLogListener {
	ctx, cancel := context.WithCancel(context.Background())
	return &ContainerLogListener{
		clientService: clientService,
		matcher:       matcher,
		activeStreams: make(map[string]context.CancelFunc),
		logChannel:    make(chan *container.LogEvent, 1000),
		ctx:           ctx,
		cancel:        cancel,
	}
}

// Start begins listening for container events and processes log streams
func (l *ContainerLogListener) Start() error {
	// Get all current containers
	containers, err := l.clientService.ListContainers(l.ctx, nil)
	if err != nil {
		return err
	}

	// Start listening to containers that match
	for _, c := range containers {
		if l.matcher.ShouldListenToContainer(c) {
			l.startListening(c)
		}
	}

	// Subscribe to new containers
	containerChan := make(chan container.Container, 10)
	l.clientService.SubscribeContainersStarted(l.ctx, containerChan)

	go func() {
		for {
			select {
			case <-l.ctx.Done():
				return
			case c := <-containerChan:
				if l.matcher.ShouldListenToContainer(c) {
					l.startListening(c)
				}
			}
		}
	}()

	return nil
}

// UpdateStreams updates which containers to listen to based on current matcher rules
func (l *ContainerLogListener) UpdateStreams() error {
	// Get all current containers
	containers, err := l.clientService.ListContainers(l.ctx, nil)
	if err != nil {
		return err
	}

	// Check each container against matcher
	for _, c := range containers {
		shouldListen := l.matcher.ShouldListenToContainer(c)
		isListening := l.isListening(c.ID)

		if shouldListen && !isListening {
			l.startListening(c)
		} else if !shouldListen && isListening {
			l.stopListening(c.ID)
		}
	}

	return nil
}

// startListening starts listening to a container's logs
func (l *ContainerLogListener) startListening(c container.Container) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Already listening
	if _, exists := l.activeStreams[c.ID]; exists {
		return
	}

	streamCtx, cancel := context.WithCancel(l.ctx)
	l.activeStreams[c.ID] = cancel

	go func() {
		log.Info().Str("containerID", c.ID).Str("name", c.Name).Msg("Started listening to container")
		if err := l.clientService.StreamLogs(streamCtx, c, time.Now(), container.STDALL, l.logChannel); err != nil {
			log.Error().Err(err).Str("containerID", c.ID).Msg("Error streaming logs")
		}
	}()
}

// stopListening stops listening to a container's logs
func (l *ContainerLogListener) stopListening(containerID string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if cancel, exists := l.activeStreams[containerID]; exists {
		cancel()
		delete(l.activeStreams, containerID)
		log.Info().Str("containerID", containerID).Msg("Stopped listening to container")
	}
}

// isListening returns true if listening to a container
func (l *ContainerLogListener) isListening(containerID string) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	_, exists := l.activeStreams[containerID]
	return exists
}

// LogChannel returns the channel for log events
func (l *ContainerLogListener) LogChannel() <-chan *container.LogEvent {
	return l.logChannel
}

// Close stops all active streams
func (l *ContainerLogListener) Close() {
	l.cancel()

	l.mu.Lock()
	defer l.mu.Unlock()

	for id, cancel := range l.activeStreams {
		cancel()
		delete(l.activeStreams, id)
	}

	close(l.logChannel)
}

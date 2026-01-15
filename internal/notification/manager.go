package notification

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/amir20/dozzle/internal/container"
	container_support "github.com/amir20/dozzle/internal/support/container"
	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
	"github.com/rs/zerolog/log"
)

// Manager handles notification subscriptions and matching
type Manager struct {
	subscriptions    []*compiledSubscription
	mu               sync.RWMutex
	activeStreams    map[string]*activeStream // containerID -> stream info
	containerCache   map[string]Container     // containerID -> notification.Container
	containerService ContainerService
	dispatcher       Dispatcher
}

// activeStream tracks a single log stream and all subscriptions watching it
type activeStream struct {
	container     Container               // notification.Container
	subscriptions []*compiledSubscription // All subscriptions watching this container
	cancel        context.CancelFunc      // Cancel function for the stream
}

// compiledSubscription wraps a Subscription with compiled expr programs
type compiledSubscription struct {
	subscription     *Subscription
	containerProgram *vm.Program // Always non-nil
	logProgram       *vm.Program // Always non-nil
}

// ContainerService provides access to containers and log streaming
type ContainerService interface {
	ListAllContainers(labels container.ContainerLabels) ([]container.Container, []error)
	FindContainer(host, id string, labels container.ContainerLabels) (*container_support.ContainerService, error)
	SubscribeContainersStarted(ctx context.Context, containers chan<- container.Container, filter container_support.ContainerFilter)
}

// NewManager creates a new notification manager
func NewManager(containerService ContainerService, dispatcher Dispatcher) *Manager {
	return &Manager{
		subscriptions:    make([]*compiledSubscription, 0),
		activeStreams:    make(map[string]*activeStream),
		containerCache:   make(map[string]Container),
		containerService: containerService,
		dispatcher:       dispatcher,
	}
}

// LoadSubscriptions loads and compiles multiple subscriptions (replaces existing)
func (m *Manager) LoadSubscriptions(subs []*Subscription) error {
	m.mu.Lock()
	m.subscriptions = make([]*compiledSubscription, 0, len(subs))
	m.mu.Unlock()

	for _, sub := range subs {
		if err := m.AddSubscription(sub); err != nil {
			return err
		}
	}

	log.Info().Int("count", len(subs)).Msg("loaded notification subscriptions")
	return nil
}

// AddSubscription adds a single subscription
func (m *Manager) AddSubscription(sub *Subscription) error {
	cs, err := m.compileSubscription(sub)
	if err != nil {
		return err
	}

	m.mu.Lock()
	m.subscriptions = append(m.subscriptions, cs)
	m.mu.Unlock()

	log.Info().Str("subscription", sub.Name).Msg("added notification subscription")
	return nil
}

// compileSubscription compiles expr programs for a subscription
func (m *Manager) compileSubscription(sub *Subscription) (*compiledSubscription, error) {
	if sub.ContainerFilter == "" {
		return nil, fmt.Errorf("container_filter is required for subscription %q", sub.Name)
	}
	if sub.LogFilter == "" {
		return nil, fmt.Errorf("log_filter is required for subscription %q", sub.Name)
	}

	cs := &compiledSubscription{
		subscription: sub,
	}

	// Compile container filter (required)
	containerProgram, err := expr.Compile(sub.ContainerFilter, expr.Env(Container{}))
	if err != nil {
		log.Error().Err(err).Str("subscription", sub.Name).Msg("failed to compile container filter")
		return nil, fmt.Errorf("failed to compile container_filter for %q: %w", sub.Name, err)
	}
	cs.containerProgram = containerProgram

	// Compile log filter (required)
	logProgram, err := expr.Compile(sub.LogFilter, expr.Env(Log{}))
	if err != nil {
		log.Error().Err(err).Str("subscription", sub.Name).Msg("failed to compile log filter")
		return nil, fmt.Errorf("failed to compile log_filter for %q: %w", sub.Name, err)
	}
	cs.logProgram = logProgram

	return cs, nil
}

// Start begins monitoring containers and streaming logs for matching subscriptions
// Only starts if there are subscriptions configured
func (m *Manager) Start(ctx context.Context) error {
	m.mu.RLock()
	hasSubscriptions := len(m.subscriptions) > 0
	m.mu.RUnlock()

	if !hasSubscriptions {
		log.Debug().Msg("no subscriptions configured, skipping notification manager start")
		return nil
	}

	// Subscribe to new containers that match our filters
	newContainers := make(chan container.Container)
	m.containerService.SubscribeContainersStarted(ctx, newContainers, func(c *container.Container) bool {
		matchingSubs := m.getMatchingSubscriptions(c)
		return len(matchingSubs) > 0
	})

	// Handle new containers in background
	go func() {
		for c := range newContainers {
			m.startContainerStream(ctx, &c)
		}
	}()

	// Get all existing containers and start streaming matching ones
	containers, errs := m.containerService.ListAllContainers(nil)
	for _, err := range errs {
		log.Warn().Err(err).Msg("error listing containers for notifications")
	}

	// Check each container and start streaming only if it matches any subscription
	for _, c := range containers {
		if c.State == "running" {
			matchingSubs := m.getMatchingSubscriptions(&c)
			if len(matchingSubs) > 0 {
				m.startContainerStream(ctx, &c)
			}
		}
	}

	return nil
}

// startContainerStream starts a single log stream for a container
// All matching subscriptions will receive logs from this one stream
func (m *Manager) startContainerStream(ctx context.Context, c *container.Container) {
	// Check if already streaming
	m.mu.RLock()
	_, exists := m.activeStreams[c.ID]
	m.mu.RUnlock()

	if exists {
		return
	}

	// Find which subscriptions match this container
	matchingSubs := m.getMatchingSubscriptions(c)
	if len(matchingSubs) == 0 {
		return
	}

	log.Debug().
		Str("container", c.Name).
		Int("subscriptions", len(matchingSubs)).
		Msg("starting log stream for notification subscriptions")

	// Convert to notification.Container and cache it
	notifContainer := NewContainer(c)
	m.mu.Lock()
	m.containerCache[c.ID] = notifContainer
	m.mu.Unlock()

	streamCtx, cancel := context.WithCancel(ctx)

	stream := &activeStream{
		container:     notifContainer,
		subscriptions: matchingSubs,
		cancel:        cancel,
	}

	m.mu.Lock()
	m.activeStreams[c.ID] = stream
	m.mu.Unlock()

	go func() {
		defer func() {
			m.mu.Lock()
			delete(m.activeStreams, c.ID)
			m.mu.Unlock()
		}()

		logs := make(chan *container.LogEvent)

		// Stream logs in separate goroutine
		go func() {
			containerService, err := m.containerService.FindContainer(c.Host, c.ID, nil)
			if err != nil {
				log.Error().Err(err).Msg("error finding container for notification streaming")
				return
			}

			err = containerService.StreamLogs(streamCtx, time.Now(), container.STDOUT|container.STDERR, logs)
			if err != nil && err != context.Canceled {
				log.Error().Err(err).Msg("error streaming logs for notification")
			}
			close(logs)
		}()

		// Process logs - check against all matching subscriptions
		for logEvent := range logs {
			for _, cs := range matchingSubs {
				m.processLogEvent(logEvent, c.ID, cs)
			}
		}
	}()
}

// getMatchingSubscriptions returns subscriptions that match a container
func (m *Manager) getMatchingSubscriptions(c *container.Container) []*compiledSubscription {
	m.mu.RLock()
	defer m.mu.RUnlock()

	containerCtx := NewContainer(c)
	matching := make([]*compiledSubscription, 0)

	for _, cs := range m.subscriptions {
		if m.matchesContainer(cs.containerProgram, containerCtx) {
			matching = append(matching, cs)
		}
	}

	return matching
}

// matchesContainer evaluates the container filter expression
func (m *Manager) matchesContainer(program *vm.Program, containerCtx Container) bool {
	result, err := expr.Run(program, containerCtx)
	if err != nil {
		log.Error().Err(err).Msg("error evaluating container filter")
		return false
	}

	match, ok := result.(bool)
	if !ok {
		log.Error().Msg("container filter did not return boolean")
		return false
	}

	return match
}

// processLogEvent evaluates a log event against the subscription's log filter
func (m *Manager) processLogEvent(logEvent *container.LogEvent, containerID string, cs *compiledSubscription) {
	// Lookup notification.Container from cache
	m.mu.RLock()
	notifContainer, ok := m.containerCache[containerID]
	m.mu.RUnlock()

	if !ok {
		log.Warn().Str("container", containerID).Msg("container not in cache")
		return
	}

	logCtx := NewLog(logEvent, notifContainer)

	// Evaluate log filter (program is never nil)
	if m.matchesLog(cs.logProgram, logCtx) {
		m.triggerWebhook(cs.subscription, logCtx)
	}
}

// matchesLog evaluates the log filter expression
func (m *Manager) matchesLog(program *vm.Program, logCtx Log) bool {
	result, err := expr.Run(program, logCtx)
	if err != nil {
		log.Error().Err(err).Msg("error evaluating log filter")
		return false
	}

	match, ok := result.(bool)
	if !ok {
		log.Error().Msg("log filter did not return boolean")
		return false
	}

	return match
}

// triggerWebhook sends the notification
func (m *Manager) triggerWebhook(sub *Subscription, logCtx Log) {
	payload := WebhookPayload{
		SubscriptionName: sub.Name,
		Timestamp:        time.Now(),
		Container:        logCtx.Container,
		Log: LogEvent{
			Message:   logCtx.Message,
			Level:     logCtx.Level,
			Timestamp: logCtx.Timestamp,
		},
	}

	m.dispatcher.Dispatch(sub, payload)
}

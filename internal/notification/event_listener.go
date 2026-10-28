package notification

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/rs/zerolog/log"
)

var allowedEventNames = map[string]bool{
	"start":         true,
	"stop":          true,
	"die":           true,
	"restart":       true,
	"health_status": true,
	"oom":           true,
	"kill":          true,
}

// UpdateEventName is the event an update or rollback Dozzle made raises. Docker
// has no such event: an update is a stop, a create and a start, so it is built
// from the update record instead.
const UpdateEventName = "update"

type ContainerEventEntry struct {
	Event     container.ContainerEvent
	Container container.Container
	Host      container.Host
}

type ContainerEventListener struct {
	clients    []container.ClientService
	channel    chan *ContainerEventEntry
	parentCtx  context.Context
	cache      *TTLCache[string, containerInfo]
	mu         sync.Mutex
	cancelFunc context.CancelFunc

	// updates, when set, turns every record it gets into an update event.
	// updateClients finds the container an update left running, on any host:
	// an update on an agent host is recorded by the server, not the agent.
	updates       *container.UpdateRecords
	updateClients func() []container.ClientService

	// checkUpdates, when set, runs daily at checkAt past midnight. alerted
	// maps host/name to the newest digest already raised as update_available.
	checkUpdates   UpdateChecker
	checkAt        time.Duration
	alertedMu      sync.Mutex
	alertedUpdates map[string]string
}

func NewContainerEventListener(ctx context.Context, clients []container.ClientService) *ContainerEventListener {
	return &ContainerEventListener{
		clients:   clients,
		channel:   make(chan *ContainerEventEntry, 1000),
		parentCtx: ctx,
		cache:     NewTTLCache[string, containerInfo](ctx, 30*time.Second),

		alertedUpdates: map[string]string{},
	}
}

// ListenToUpdates raises an update event for every record added to records
// from the next Start on. Only the process that records updates calls it, so
// an update is never raised twice.
func (l *ContainerEventListener) ListenToUpdates(records *container.UpdateRecords, clients func() []container.ClientService) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.updates = records
	l.updateClients = clients
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
	named := container.WithSubscriberName(ctx, "notifications")
	for _, client := range l.clients {
		client.SubscribeEvents(named, rawEvents)
	}

	go l.enrich(ctx, rawEvents)

	if l.updates != nil {
		records := make(chan container.UpdateRecord, 64)
		l.updates.Subscribe(ctx, records)
		go l.forwardUpdates(ctx, records)
	}

	if l.checkUpdates != nil && l.updateClients != nil {
		go l.runUpdateChecks(ctx, l.checkUpdates, l.checkAt)
	}

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

// normalizeEvent rewrites Docker's health event so notification expressions can match it.
// Docker emits health events as "health_status: healthy" / "health_status: unhealthy".
// We collapse them to a bare "health_status" name and expose the status as a "healthStatus"
// attribute, so expressions like
// name == "health_status" && attributes["healthStatus"] == "unhealthy" work.
func normalizeEvent(event *container.ContainerEvent) {
	if name, status, found := strings.Cut(event.Name, ": "); found && name == "health_status" {
		event.Name = name
		if event.ActorAttributes == nil {
			event.ActorAttributes = map[string]string{}
		}
		event.ActorAttributes["healthStatus"] = status
	}
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

			normalizeEvent(&event)

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

func (l *ContainerEventListener) forwardUpdates(ctx context.Context, records <-chan container.UpdateRecord) {
	for {
		select {
		case <-ctx.Done():
			return
		case r := <-records:
			entry, ok := l.updateEntry(ctx, r)
			if !ok {
				continue
			}
			select {
			case l.channel <- entry:
			case <-ctx.Done():
				return
			default:
				log.Warn().Str("containerID", r.NewID).Msg("Event channel full, dropping update event")
			}
		}
	}
}

// updateEntry is the update event for r, on the container r left running.
func (l *ContainerEventListener) updateEntry(ctx context.Context, r container.UpdateRecord) (*ContainerEventEntry, bool) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	for _, client := range l.updateClients() {
		host, err := client.Host(ctx)
		if err != nil || host.ID != r.Host {
			continue
		}
		c, err := client.FindContainer(ctx, r.NewID, nil)
		if err != nil {
			log.Debug().Err(err).Str("containerID", r.NewID).Msg("update event: container not found")
			return nil, false
		}
		if isDozzleContainer(c) {
			return nil, false
		}
		return &ContainerEventEntry{Event: updateEvent(r), Container: c, Host: host}, true
	}
	return nil, false
}

func updateEvent(r container.UpdateRecord) container.ContainerEvent {
	return container.ContainerEvent{
		Name:    UpdateEventName,
		Host:    r.Host,
		ActorID: r.NewID,
		ActorAttributes: map[string]string{
			"source":     r.Source,
			"rolledBack": strconv.FormatBool(r.RolledBack),
			"image":      r.ImageRef,
			"fromDigest": r.FromDigest,
			"toDigest":   r.ToDigest,
		},
		Time: r.At,
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

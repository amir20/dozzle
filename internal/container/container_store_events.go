package container

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/puzpuzpuz/xsync/v4"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
)

// matchesLabels is a cheap pre-check against the store's filter, so an update for a
// container outside it does not cost a full list every time it changes.
func matchesLabels(labels map[string]string, filter ContainerLabels) bool {
	for key, values := range filter {
		value, ok := labels[key]
		if !ok || !slices.Contains(values, value) {
			return false
		}
	}
	return true
}

// addContainer records a newly created or started container and returns what it
// stored. It does not notify: create and start both land here for the same container,
// so the caller decides which of them counts as the container starting.
//
// With no filter configured the inspect alone is enough. With one, the list is the
// authoritative membership check: filter keys can be docker filters that are not
// labels at all, so matchesLabels cannot stand in for it. It runs first, so a
// container outside the filter never costs an inspect.
//
// FindContainer can fail transiently (the daemon is busy right after a compose recreate,
// or the inspect times out). Nothing re-adds the container afterwards, so it would stay
// missing from the store until the next reconnect and the UI would never update it again.
// Fall back to the list entry in that case: it is not FullyLoaded, so the next
// FindContainer fills in the rest.
func (s *ContainerStore) addContainer(id string, timeout time.Duration) (Container, bool) {
	ctx, cancel := context.WithTimeout(s.ctx, timeout)
	defer cancel()

	var listed *Container
	if s.labels.Exists() {
		if listed = s.listedEntry(ctx, id); listed == nil {
			return Container{}, false
		}
	}

	found, err := s.client.FindContainer(ctx, id)
	if err == nil {
		return *s.storeKeepingStats(found), true
	}

	log.Warn().Err(err).Str("id", id).Msg("failed to inspect container, falling back to list entry")
	if listed == nil {
		if listed = s.listedEntry(ctx, id); listed == nil {
			return Container{}, false
		}
	}
	return *s.storeKeepingStats(*listed), true
}

// listedEntry is the store's filtered list entry for id, or nil if the list does not
// have it.
func (s *ContainerStore) listedEntry(ctx context.Context, id string) *Container {
	list, err := s.client.ListContainers(ctx, s.labels)
	if err != nil {
		log.Warn().Err(err).Str("id", id).Msg("failed to list containers while adding container")
		return nil
	}
	if listed, ok := lo.Find(list, func(item Container) bool { return item.ID == id }); ok {
		return &listed
	}
	return nil
}

// patch applies change to a copy of a container and stores the copy. Entries are
// never mutated in place: readers hold the old pointer, and a refresh compares
// against it. change returns false to leave the entry alone. ok is false when the
// container is unknown or was left alone.
func (s *ContainerStore) patch(id string, change func(c *Container) bool) (patched *Container, ok bool) {
	patched, _ = s.containers.Compute(id, func(c *Container, loaded bool) (*Container, xsync.ComputeOp) {
		if !loaded {
			return c, xsync.CancelOp
		}
		copy := *c
		if !change(&copy) {
			return c, xsync.CancelOp
		}
		ok = true
		return &copy, xsync.UpdateOp
	})
	return patched, ok
}

// run is the store's one goroutine: it boots the store, then applies events and stats
// for as long as the store lives. Everything that changes a container because of an
// event happens here, in order.
func (s *ContainerStore) run() {
	stats := make(chan ContainerStat)
	s.statsCollector.Subscribe(s.ctx, stats)

	s.boot()

	for {
		select {
		case event := <-s.events:
			log.Trace().Str("event", event.Name).Str("id", event.ActorID).Msg("received container event")
			s.handleEvent(event)
			s.broadcast(event)

		case stat := <-stats:
			if container, ok := s.containers.Load(stat.ID); ok && container.Stats != nil {
				s.volumeMonitor.observe(container, stat)
				stat.ID = ""
				container.Stats.Push(stat)
			}
		case <-s.ctx.Done():
			return
		}
	}
}

// boot connects the event stream, fills the map and opens the store to callers.
func (s *ContainerStore) boot() {
	go s.refresher()

	// the stream starts before the first list, so what happens during the list is not
	// missed once the subscription is live
	s.staleGen.Add(1)
	go s.streamEvents()
	if err := s.ensureFresh(s.ctx); err != nil {
		// ready is not held back on it: callers get the error from their own
		// ListContainers instead of hanging while the daemon is down
		log.Error().Err(err).Str("host", s.client.Host().Name).Msg("failed to list containers while initializing container store, retrying")
		s.wakeRefresher()
	}

	// The subscription goes live a moment after streamEvents starts, and a container
	// that starts in between is in neither the list nor the stream. Holding the first
	// list back would add that wait to every boot, so look again afterwards instead:
	// one list, no inspects.
	go func() {
		if !SleepOrDone(s.ctx, s.timing.subscribeGrace) || s.staleGen.Load() != s.freshGen.Load() {
			return
		}
		if missed, err := s.refresh(false); err == nil {
			s.replay(missed)
		}
	}()

	close(s.ready)
}

func (s *ContainerStore) handleEvent(event ContainerEvent) {
	id := event.ActorID
	switch event.Name {
	case "create":
		// docker follows a create with a start, which is what notifies. Only a
		// container that is already running here (a k8s pod created running)
		// would otherwise never be announced.
		if added, ok := s.addContainer(id, 3*time.Second); ok && added.State == "running" {
			s.notifyNewContainer(added)
		}

	case "start":
		if added, ok := s.addContainer(id, defaultTimeout); ok {
			s.notifyNewContainer(added)
		}

	case "destroy":
		log.Debug().Str("id", id).Msg("container destroyed")
		s.containers.Delete(id)
		delete(s.announced, id)

	case "update":
		s.handleUpdate(event)

	case "die":
		log.Debug().Str("id", id).Msg("container died")
		s.patch(id, func(c *Container) bool {
			c.State = "exited"
			c.FinishedAt = time.Now()
			return true
		})

	case "pause", "unpause":
		state := "paused"
		if event.Name == "unpause" {
			state = "running"
		}
		log.Debug().Str("id", id).Str("state", state).Msg("container state changed")
		s.patch(id, func(c *Container) bool {
			c.State = state
			return true
		})

	case "health_status: healthy", "health_status: unhealthy":
		_, health, _ := strings.Cut(event.Name, ": ")
		log.Debug().Str("id", id).Str("health", health).Msg("container health status changed")
		s.patch(id, func(c *Container) bool {
			c.Health = health
			return true
		})

	case "rename":
		s.patch(id, func(c *Container) bool {
			// A dev.dozzle.name or coolify.serviceName label pins a custom
			// display name (see newContainer). That name must survive a
			// docker-level rename, so only follow the rename when the name
			// actually comes from Docker.
			if c.Labels["dev.dozzle.name"] != "" || c.Labels["coolify.serviceName"] != "" {
				log.Debug().Str("id", id).Msg("ignoring rename: container has a custom name label")
				return false
			}
			log.Debug().Str("id", id).Str("name", event.ActorAttributes["name"]).Msg("container renamed")
			c.Name = event.ActorAttributes["name"]
			return true
		})
	}
}

// handleUpdate applies an update that carries the container's new state. Only
// Kubernetes sends these.
func (s *ContainerStore) handleUpdate(event ContainerEvent) {
	update := event.Container
	if update == nil {
		return
	}

	started := false
	updated, known := s.patch(event.ActorID, func(c *Container) bool {
		// A short-lived k8s pod (a Job) can go from Pending straight to
		// Succeeded without ever reporting Running. It still ran, so it
		// counts as a start, or the UI never learns it exists.
		started = c.State != "running" && (update.State == "running" || (c.State == "created" && update.State == "exited"))
		c.Name = update.Name
		c.State = update.State
		c.Labels = update.Labels
		c.StartedAt = update.StartedAt
		c.FinishedAt = update.FinishedAt
		c.Created = update.Created
		c.Host = update.Host
		return true
	})

	if !known {
		// The store never loaded this container: its create landed before the store's
		// first list, or adding it failed. A k8s watch no longer ends and forces a
		// fresh list, so pick it up here or it stays missing for the life of the process.
		if !matchesLabels(update.Labels, s.labels) {
			return
		}
		if added, ok := s.addContainer(event.ActorID, 3*time.Second); ok {
			updated, started = &added, true
		}
	}

	if started {
		// k8s sends create for a pending pod and only this update once it runs, so
		// this is where its start is announced
		s.broadcast(ContainerEvent{Name: "start", ActorID: updated.ID, Host: updated.Host})
		s.notifyNewContainer(*updated)
	}
}

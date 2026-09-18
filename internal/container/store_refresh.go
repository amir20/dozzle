package container

import (
	"context"
	"time"

	"github.com/puzpuzpuz/xsync/v4"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/semaphore"
)

// storeTiming is the backoff for reconnecting the event stream, and how long a new
// subscription is given to go live before the map is listed again. The docker SDK
// subscribes to /events on its own goroutine and never says when it is live, and a
// list taken before that misses whatever changes in between while still marking the
// map fresh. Waiting a moment puts the list after the subscription.
//
// It lives on the store rather than in package globals so tests can shrink it for one
// instance without writing to state other running stores read.
type storeTiming struct {
	retryMin       time.Duration
	retryMax       time.Duration
	subscribeGrace time.Duration
}

var defaultStoreTiming = storeTiming{retryMin: time.Second, retryMax: 30 * time.Second, subscribeGrace: time.Second}

// streamEvents keeps the event stream connected for the life of the store. It runs
// from boot whether or not anyone is watching, so reconnecting costs nothing extra.
// It used to reconnect only when something next called ListContainers, which left a
// headless instance deaf after a daemon restart: new containers never appeared and
// event and log alerts went silent with nothing saying so.
//
// Docker does not replay what it sent while nobody was listening, so every reconnect
// marks the map stale once the new subscription is live. The first connect's list is
// init's.
func (s *Store) streamEvents() {
	backoff := s.timing.retryMin
	for attempt := 0; ; attempt++ {
		if attempt > 0 {
			go func() {
				// marked stale only after the grace, or a ListContainers during it
				// would list early and mark the map fresh
				if SleepOrDone(s.ctx, s.timing.subscribeGrace) {
					s.markStale()
				}
			}()
		}

		connectedAt := time.Now()
		log.Debug().Str("host", s.client.Host().Name).Msg("docker store subscribing docker events")
		err := s.client.ContainerEvents(s.ctx, s.events)
		if s.ctx.Err() != nil {
			return
		}

		// a stream that stayed up for a while was healthy, so start the backoff over
		if time.Since(connectedAt) > s.timing.retryMax {
			backoff = s.timing.retryMin
		}
		log.Warn().Err(err).Str("host", s.client.Host().Name).Dur("retry_in", backoff).Msg("docker store disconnected from docker events, reconnecting")
		if !SleepOrDone(s.ctx, backoff) {
			return
		}
		backoff = NextBackoff(backoff, s.timing.retryMax)
	}
}

// markStale says the map may have missed something and wakes the refresher.
func (s *Store) markStale() {
	s.staleGen.Add(1)
	s.wakeRefresher()
}

func (s *Store) wakeRefresher() {
	select {
	case s.refreshWake <- struct{}{}:
	default:
	}
}

// refresher refreshes the map whenever it is stale, without waiting for someone to
// call ListContainers, and retries until the map is actually fresh. A nil error is
// not enough to stop on: ensureFresh may have joined a refresh that started before
// the latest markStale and so does not cover it.
func (s *Store) refresher() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-s.refreshWake:
		}

		backoff := s.timing.retryMin
		for s.staleGen.Load() != s.freshGen.Load() {
			err := s.ensureFresh(s.ctx)
			if s.ctx.Err() != nil {
				return
			}
			if err == nil {
				continue
			}
			log.Warn().Err(err).Str("host", s.client.Host().Name).Dur("retry_in", backoff).Msg("failed to refresh containers, retrying")
			if !SleepOrDone(s.ctx, backoff) {
				return
			}
			backoff = NextBackoff(backoff, s.timing.retryMax)
		}
	}
}

// ensureFresh lists containers when the map may be out of date. A failed list leaves
// it stale, so the next caller retries it. ctx only bounds how long this caller
// waits: the refresh runs on its own goroutine and the store's context, because
// callers share it and one cancelled request must not fail it for the others.
func (s *Store) ensureFresh(ctx context.Context) error {
	if s.staleGen.Load() == s.freshGen.Load() {
		return nil
	}

	result := s.refreshes.DoChan("refresh", func() (any, error) {
		gen := s.staleGen.Load()
		if gen == s.freshGen.Load() {
			// a refresh that finished just before this one started covered it
			return nil, nil
		}
		// The first list has nothing to compare against: everything in it is new to
		// the map but not newly started, and announcing it would restart every log
		// stream from the container's StartedAt.
		first := s.freshGen.Load() == 0
		missed, err := s.refresh(true)
		if err != nil {
			return nil, err
		}
		s.freshGen.Store(gen)
		if !first {
			s.replay(missed)
		}
		return nil, nil
	})

	select {
	case r := <-result:
		return r.Err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// replay feeds the event loop what a refresh found out the stream never said: the
// refresh fixes the map, but subscribers only hear about events. Without this a
// container that came back during a daemon restart never reached the log listener,
// so its log alerts stayed silent, and open tabs kept showing containers that were gone.
//
// They go through the loop like real events, so start dedupe, alerts and SSE all
// behave as if the stream had delivered them.
func (s *Store) replay(missed []ContainerEvent) {
	for _, event := range missed {
		log.Debug().Str("event", event.Name).Str("id", event.ActorID).Msg("replaying event missed while the stream was down")
		select {
		case s.events <- event:
		case <-s.ctx.Done():
			return
		}
	}
}

// refresh reconciles the map with a fresh list. It never clears the map first:
// concurrent readers would see it empty or half built, and a container the event loop
// added while the list was in flight would be wiped. Only IDs that were already in the
// map before the list, and that the list no longer reports, are removed.
//
// A full refresh re-inspects every running container, which is what catches a restart
// during an outage. A light one trusts entries that are already fully loaded and only
// picks up what the list shows changed.
//
// It returns the events the changes it made amount to, for replay.
func (s *Store) refresh(full bool) ([]ContainerEvent, error) {
	previous := make(map[string]*Container)
	s.containers.Range(func(id string, c *Container) bool {
		previous[id] = c
		return true
	})

	ctx, cancel := context.WithTimeout(s.ctx, defaultTimeout)
	defer cancel()
	containers, err := s.client.ListContainers(ctx, s.labels)
	if err != nil {
		return nil, err
	}

	diff := s.applyList(previous, containers, full)
	s.inspectPartial(containers)

	log.Debug().Int("containers", len(containers)).Bool("full", full).Msg("finished refreshing container store")
	return s.missedEvents(previous, containers, diff), nil
}

// listDiff is what applying a list changed that the event loop had not already
// handled itself. Anything the loop touched is left out: it was broadcast when it
// happened, and replaying it would fire its alerts twice.
type listDiff struct {
	changed map[string]struct{} // new to the map, or in a different state
	died    []*Container
	removed []*Container
}

// applyList stores every listed container and removes the ones the list no longer has.
func (s *Store) applyList(previous map[string]*Container, containers []Container, full bool) listDiff {
	diff := listDiff{changed: make(map[string]struct{})}

	listed := make(map[string]struct{}, len(containers))
	for _, c := range containers {
		listed[c.ID] = struct{}{}
		before := previous[c.ID]
		if stored, touched := s.storeListed(before, c, full); !stored || touched {
			continue
		}
		if before == nil || before.State != c.State {
			diff.changed[c.ID] = struct{}{}
		}
		if before != nil && before.State == "running" && c.State == "exited" {
			diff.died = append(diff.died, before)
		}
	}

	for id, before := range previous {
		if _, ok := listed[id]; ok {
			continue
		}
		// Only drop the entry the snapshot saw. If the loop changed it during the list,
		// the container came back after the list was taken (a k8s StatefulSet pod keeps
		// its ID across a recreate) and deleting it would lose it for good.
		_, kept := s.containers.Compute(id, func(c *Container, loaded bool) (*Container, xsync.ComputeOp) {
			if loaded && !touchedByLoop(before, c) {
				return nil, xsync.DeleteOp
			}
			return c, xsync.CancelOp
		})
		if !kept {
			diff.removed = append(diff.removed, before)
		}
	}
	return diff
}

// inspectPartial inspects the running containers that are only known from the list,
// in parallel, and waits for all of them.
func (s *Store) inspectPartial(containers []Container) {
	sem := semaphore.NewWeighted(maxFetchParallelism)
	for _, c := range containers {
		if c.State == "exited" {
			continue
		}
		prev, ok := s.containers.Load(c.ID)
		if !ok || prev.FullyLoaded {
			continue
		}
		if err := sem.Acquire(s.ctx, 1); err != nil {
			break
		}
		go func() {
			defer sem.Release(1)
			ctx, cancel := context.WithTimeout(s.ctx, defaultTimeout)
			defer cancel()
			if fetched, err := s.client.FindContainer(ctx, prev.ID); err == nil {
				s.mergeFetched(prev, fetched)
			}
		}()
	}
	// all of them, which is to say every inspect has finished
	_ = sem.Acquire(s.ctx, maxFetchParallelism)
}

// missedEvents turns a refresh into the events the stream would have sent. Starts are
// worked out after the inspects, because only an inspect has StartedAt and a restart
// during the outage shows up as nothing else. A start the loop also saw is announced
// once: notifyNewContainer dedupes on StartedAt.
func (s *Store) missedEvents(previous map[string]*Container, containers []Container, diff listDiff) []ContainerEvent {
	var missed []ContainerEvent
	add := func(name string, c *Container) {
		missed = append(missed, ContainerEvent{Name: name, ActorID: c.ID, Host: c.Host, Time: time.Now()})
	}

	for _, c := range diff.died {
		add("die", c)
	}
	for _, c := range diff.removed {
		add("destroy", c)
	}
	for _, c := range containers {
		cur, ok := s.containers.Load(c.ID)
		if !ok || cur.State != "running" {
			continue
		}
		before := previous[c.ID]
		_, stateChanged := diff.changed[c.ID]
		restarted := before != nil && !before.StartedAt.IsZero() && !cur.StartedAt.IsZero() && !before.StartedAt.Equal(cur.StartedAt)
		if stateChanged || restarted {
			add("start", cur)
		}
	}
	return missed
}

// storeKeepingStats stores c, carrying over the stats history and mount stats of the
// entry it replaces. A list or inspect result always comes with an empty ring buffer,
// and swapping that in would reset every chart on each reconnect or refetch.
func (s *Store) storeKeepingStats(c Container) *Container {
	stored, _ := s.containers.Compute(c.ID, func(existing *Container, loaded bool) (*Container, xsync.ComputeOp) {
		if loaded {
			carryOverStats(existing, &c)
		}
		return &c, xsync.UpdateOp
	})
	return stored
}

func carryOverStats(from *Container, to *Container) {
	if from.Stats != nil {
		to.Stats = from.Stats
	}
	if from.MountStats != nil {
		to.MountStats = from.MountStats
	}
}

// storeListed stores a list entry taken during a refresh. It reports whether it did,
// and whether the event loop had touched the entry, in which case the loop already
// told subscribers about it.
// before is what the map held when the refresh started. If the entry changed while
// the list was in flight, the event loop got there first (a die, a pause, a health
// change) and its state is newer than the list's, so the list must not write the
// older state back.
func (s *Store) storeListed(before *Container, c Container, full bool) (stored bool, touched bool) {
	s.containers.Compute(c.ID, func(existing *Container, loaded bool) (*Container, xsync.ComputeOp) {
		if !loaded {
			if before != nil {
				// the loop handled its destroy during the list; storing it again would
				// leave a ghost until the next reconnect
				return existing, xsync.CancelOp
			}
			stored = true
			return &c, xsync.UpdateOp
		}
		if touchedByLoop(before, existing) {
			touched = true
			if existing.FullyLoaded {
				// stored by the event loop during the list from an inspect, which is
				// newer than the list and complete
				return existing, xsync.CancelOp
			}
			keepLoopFields(existing, &c)
		} else if !full && existing.FullyLoaded {
			if existing.State == c.State {
				return existing, xsync.CancelOp
			}
			copy := *existing
			copy.State = c.State
			stored = true
			return &copy, xsync.UpdateOp
		}
		carryOverStats(existing, &c)
		stored = true
		return &c, xsync.UpdateOp
	})
	return stored, touched
}

// touchedByLoop reports whether the event loop changed an entry since before was
// read. Pointer identity is not enough: applyMountStats swaps the pointer every
// minute without changing anything a list or an inspect knows better.
func touchedByLoop(before *Container, current *Container) bool {
	if before == current {
		return false
	}
	if before == nil || current == nil {
		return true
	}
	return before.State != current.State ||
		before.Health != current.Health ||
		before.Name != current.Name ||
		before.FullyLoaded != current.FullyLoaded ||
		!before.StartedAt.Equal(current.StartedAt) ||
		!before.FinishedAt.Equal(current.FinishedAt) ||
		!before.Created.Equal(current.Created)
}

// keepLoopFields copies the fields the event loop maintains from events onto a
// snapshot that was taken before those events were applied. State is always known.
// The rest only when set: from can be a list entry, which has no StartedAt or Health,
// and copying its zero values would wipe what an inspect just fetched for good, since
// the result is FullyLoaded and never fetched again.
func keepLoopFields(from *Container, to *Container) {
	to.State = from.State
	if from.Health != "" {
		to.Health = from.Health
	}
	if !from.StartedAt.IsZero() {
		to.StartedAt = from.StartedAt
	}
	if !from.FinishedAt.IsZero() {
		to.FinishedAt = from.FinishedAt
	}
	if from.Name != "" {
		to.Name = from.Name
	}
}

// mergeFetched stores a container fetched from the client over prev, the entry the
// fetch started from. The fetch runs outside any lock, so by the time it returns the
// event loop may have moved on: the container can be gone, or its entry replaced.
// found reports whether the container is still in the map, updated whether the
// fetched value was stored.
func (s *Store) mergeFetched(prev *Container, fetched Container) (current *Container, found bool, updated bool) {
	current, found = s.containers.Compute(prev.ID, func(c *Container, loaded bool) (*Container, xsync.ComputeOp) {
		if !loaded {
			return c, xsync.CancelOp
		}
		if touchedByLoop(prev, c) {
			if c.FullyLoaded {
				// someone else stored a complete entry during the fetch, and it is newer
				return c, xsync.CancelOp
			}
			// the event loop changed these while the fetch was in flight, so they are
			// newer than what the fetch saw
			keepLoopFields(c, &fetched)
		}
		carryOverStats(c, &fetched)
		updated = true
		return &fetched, xsync.UpdateOp
	})
	return current, found, updated
}

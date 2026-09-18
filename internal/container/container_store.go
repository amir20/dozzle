package container

import (
	"context"
	"errors"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/puzpuzpuz/xsync/v4"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
	"golang.org/x/sync/semaphore"
	"golang.org/x/sync/singleflight"
)

type StatsCollector interface {
	Start(parentCtx context.Context) bool
	Subscribe(ctx context.Context, stats chan<- ContainerStat)
	Stop()
}

type ContainerStore struct {
	containers              *xsync.Map[string, *Container]
	subscribers             *xsync.Map[context.Context, *eventSubscriber]
	newContainerSubscribers *xsync.Map[context.Context, chan<- Container]
	client                  Client
	statsCollector          StatsCollector
	volumeMonitor           *volumeMonitor
	// ready is closed once init's first refresh has run, so callers can stop waiting
	// on it when their own context ends.
	ready chan struct{}
	// staleGen is bumped every time the event stream (re)connects, and freshGen records
	// the staleGen the last successful list covered. They differ while the map may be
	// missing whatever happened while no stream was running, including when the very
	// first list failed. A generation rather than a flag, so a reconnect that lands
	// while a refresh is in flight is not cleared by that refresh finishing.
	staleGen atomic.Uint64
	freshGen atomic.Uint64
	// refreshes runs one refresh at a time on its own goroutine and lets every caller
	// that needs it wait on the same one. A caller whose context ends stops waiting,
	// and the refresh carries on for the others.
	refreshes singleflight.Group
	// refreshWake wakes the one refresher goroutine. Every reconnect used to start its
	// own retry loop, and they piled up for as long as the daemon stayed down.
	refreshWake chan struct{}
	// inspects merges concurrent inspects of one container into one.
	inspects singleflight.Group
	timing   storeTiming
	// announced is the StartedAt each container was last announced to new-container
	// subscribers with. Only the event loop touches it.
	announced map[string]time.Time
	events    chan ContainerEvent
	ctx       context.Context
	labels    ContainerLabels
}

const defaultTimeout = 10 * time.Second

func NewContainerStore(ctx context.Context, client Client, statsCollect StatsCollector, labels ContainerLabels) *ContainerStore {
	return newContainerStore(ctx, client, statsCollect, labels, defaultStoreTiming)
}

func newContainerStore(ctx context.Context, client Client, statsCollect StatsCollector, labels ContainerLabels, timing storeTiming) *ContainerStore {
	log.Debug().Str("host", client.Host().Name).Interface("labels", labels).Msg("initializing container store")

	s := &ContainerStore{
		containers:              xsync.NewMap[string, *Container](),
		client:                  client,
		subscribers:             xsync.NewMap[context.Context, *eventSubscriber](),
		newContainerSubscribers: xsync.NewMap[context.Context, chan<- Container](),
		statsCollector:          statsCollect,
		ready:                   make(chan struct{}),
		refreshWake:             make(chan struct{}, 1),
		timing:                  timing,
		events:                  make(chan ContainerEvent),
		ctx:                     ctx,
		labels:                  labels,
	}
	s.volumeMonitor = newVolumeMonitor(s)
	s.volumeMonitor.start(ctx)

	go s.init()

	return s
}

// applyMountStats updates a container's MountStats and broadcasts an "update"
// event so subscribers (SSE) can propagate the new data to clients.
func (s *ContainerStore) applyMountStats(id string, stats map[string]MountStat) {
	updated, ok := s.containers.Compute(id, func(c *Container, loaded bool) (*Container, xsync.ComputeOp) {
		if !loaded {
			return c, xsync.CancelOp
		}
		copy := *c
		copy.MountStats = stats
		return &copy, xsync.UpdateOp
	})
	if !ok || updated == nil {
		return
	}

	s.broadcast(ContainerEvent{
		Name:      "update",
		Host:      updated.Host,
		ActorID:   updated.ID,
		Time:      time.Now(),
		Container: updated,
	})
}

var (
	ErrContainerNotFound = errors.New("container not found")
	maxFetchParallelism  = int64(30)
)

type subscriberNameKey struct{}

// WithSubscriberName labels a subscription so a dropped-event warning can say which
// consumer stalled. Subscribers are keyed by their context, so the name rides along on
// that context instead of being threaded through every ClientService implementation.
func WithSubscriberName(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, subscriberNameKey{}, name)
}

func subscriberNameFrom(ctx context.Context) string {
	if name, ok := ctx.Value(subscriberNameKey{}).(string); ok && name != "" {
		return name
	}
	return "unnamed"
}

// dropLogInterval bounds how often one stalled subscriber may warn. A container with a
// 5s healthcheck emits three exec events per cycle, so an unthrottled warning buries
// every other line in the log while repeating the same fact.
const dropLogInterval = 10 * time.Second

// eventSubscriber is one registered consumer plus the bookkeeping needed to report a
// stall as a single span rather than one line per lost event.
type eventSubscriber struct {
	ch   chan<- ContainerEvent
	name string

	mu      sync.Mutex
	dropped int       // events lost since the last warning
	total   int       // events lost since the stall began
	since   time.Time // when the current stall began
	lastLog time.Time
}

// recordDrop counts a lost event and reports whether this one should be logged.
func (s *eventSubscriber) recordDrop(now time.Time) (dropped int, stalledFor time.Duration, shouldLog bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.since.IsZero() {
		s.since = now
	}
	s.dropped++
	s.total++

	if !s.lastLog.IsZero() && now.Sub(s.lastLog) < dropLogInterval {
		return 0, 0, false
	}

	s.lastLog = now
	dropped, s.dropped = s.dropped, 0
	return dropped, now.Sub(s.since), true
}

// recordDelivered closes out a stall. It reports the total lost only once, on the first
// event that lands after the subscriber starts reading again.
func (s *eventSubscriber) recordDelivered(now time.Time) (dropped int, stalledFor time.Duration, recovered bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.since.IsZero() {
		return 0, 0, false
	}

	dropped, stalledFor = s.total, now.Sub(s.since)
	s.dropped, s.total, s.since, s.lastLog = 0, 0, time.Time{}, time.Time{}
	return dropped, stalledFor, true
}

// broadcastTimeout bounds how long a fan-out waits on a subscriber that is not
// draining its channel.
//
// Everything this store does runs on one goroutine: the Docker event stream,
// stats, and the create/start handling that is the only path a container has
// into the map. A blocking send to a subscriber puts all of that behind the
// slowest reader, and a subscriber that stops reading entirely (a handler that
// deadlocked, a client whose socket wedged) stops the store for as long as its
// context stays alive. Docker does not hold events for a consumer that has
// stopped reading, so what is lost during a stall is lost permanently — a
// missed `start` means the container never appears until the process restarts.
//
// One wedged subscriber must cost its own messages, never the loop.
const broadcastTimeout = 250 * time.Millisecond

type sendResult int

const (
	sendOK sendResult = iota
	// sendDropped: the subscriber did not read in time. Its message is gone.
	sendDropped
	// sendCancelled: the subscriber is finished and should be unregistered.
	sendCancelled
)

// fanoutBudget is the wait shared by one fan-out. The timeout is per fan-out
// rather than per subscriber so that N wedged subscribers cost the loop one
// broadcastTimeout between them, not N of them on every event. It starts on
// first use, so a fan-out where everyone is reading allocates nothing.
type fanoutBudget struct {
	expired chan struct{}
	timer   *time.Timer
}

func (b *fanoutBudget) start() <-chan struct{} {
	if b.expired == nil {
		b.expired = make(chan struct{})
		b.timer = time.AfterFunc(broadcastTimeout, func() { close(b.expired) })
	}
	return b.expired
}

func (b *fanoutBudget) stop() {
	if b.timer != nil {
		b.timer.Stop()
	}
}

// sendBounded delivers v, waiting no longer than what is left of the fan-out's
// shared budget for a subscriber that is not reading.
func sendBounded[T any](ctx context.Context, ch chan<- T, v T, budget *fanoutBudget) sendResult {
	select {
	case ch <- v:
		return sendOK
	case <-ctx.Done():
		return sendCancelled
	default:
	}

	select {
	case ch <- v:
		return sendOK
	case <-ctx.Done():
		return sendCancelled
	case <-budget.start():
		return sendDropped
	}
}

// broadcast fans an event out to every subscriber without letting any of them
// stall the caller.
func (s *ContainerStore) broadcast(event ContainerEvent) {
	budget := &fanoutBudget{}
	defer budget.stop()

	s.subscribers.Range(func(ctx context.Context, sub *eventSubscriber) bool {
		switch sendBounded(ctx, sub.ch, event, budget) {
		case sendCancelled:
			s.subscribers.Delete(ctx)
		case sendOK:
			if dropped, stalled, recovered := sub.recordDelivered(time.Now()); recovered {
				// info, not warn: this is the line that closes out the warning above, and
				// the default level shows it
				log.Info().
					Str("host", s.client.Host().Name).
					Str("subscriber", sub.name).
					Int("dropped", dropped).
					Dur("stalledFor", stalled).
					Msg("subscriber is reading container events again")
			}
		case sendDropped:
			log.Trace().
				Str("host", s.client.Host().Name).
				Str("subscriber", sub.name).
				Str("event", event.Name).
				Str("id", event.ActorID).
				Msg("dropped container event")
			if dropped, stalled, shouldLog := sub.recordDrop(time.Now()); shouldLog {
				log.Warn().
					Str("host", s.client.Host().Name).
					Str("subscriber", sub.name).
					Int("dropped", dropped).
					Dur("stalledFor", stalled).
					Msg("subscriber is not reading container events, dropping events")
			}
		}
		return true
	})
}

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
func (s *ContainerStore) streamEvents() {
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
func (s *ContainerStore) markStale() {
	s.staleGen.Add(1)
	s.wakeRefresher()
}

func (s *ContainerStore) wakeRefresher() {
	select {
	case s.refreshWake <- struct{}{}:
	default:
	}
}

// refresher refreshes the map whenever it is stale, without waiting for someone to
// call ListContainers, and retries until the map is actually fresh. A nil error is
// not enough to stop on: ensureFresh may have joined a refresh that started before
// the latest markStale and so does not cover it.
func (s *ContainerStore) refresher() {
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
func (s *ContainerStore) ensureFresh(ctx context.Context) error {
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
func (s *ContainerStore) replay(missed []ContainerEvent) {
	for _, event := range missed {
		log.Debug().Str("event", event.Name).Str("id", event.ActorID).Msg("replaying event missed while the stream was down")
		select {
		case s.events <- event:
		case <-s.ctx.Done():
			return
		}
	}
}

// waitReady blocks until init's first refresh has run, or ctx ends.
func (s *ContainerStore) waitReady(ctx context.Context) error {
	select {
	case <-s.ready:
		return nil
	case <-ctx.Done():
		return ctx.Err()
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
func (s *ContainerStore) refresh(full bool) ([]ContainerEvent, error) {
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

	var missed []ContainerEvent
	event := func(name string, c *Container) {
		missed = append(missed, ContainerEvent{Name: name, ActorID: c.ID, Host: c.Host, Time: time.Now()})
	}

	listed := make(map[string]struct{}, len(containers))
	// containers whose state only the list changed; the loop saw no event for them
	changed := make(map[string]struct{})
	for _, c := range containers {
		listed[c.ID] = struct{}{}
		before := previous[c.ID]
		if stored, touched := s.storeListed(before, c, full); !stored || touched {
			continue
		}
		switch {
		case before == nil:
			changed[c.ID] = struct{}{}
		case before.State != c.State:
			changed[c.ID] = struct{}{}
			if before.State == "running" && c.State == "exited" {
				event("die", &c)
			}
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
			event("destroy", before)
		}
	}

	running := lo.Filter(containers, func(item Container, index int) bool {
		return item.State != "exited" && !item.FullyLoaded
	})

	sem := semaphore.NewWeighted(maxFetchParallelism)

	for _, c := range running {
		if err := sem.Acquire(s.ctx, 1); err != nil {
			log.Error().Err(err).Msg("failed to acquire semaphore")
			break
		}
		go func(id string) {
			defer sem.Release(1)
			prev, ok := s.containers.Load(id)
			if !ok || prev.FullyLoaded {
				return
			}
			ctx, cancel := context.WithTimeout(s.ctx, defaultTimeout)
			defer cancel()
			if fetched, err := s.client.FindContainer(ctx, id); err == nil {
				s.mergeFetched(prev, fetched)
			}
		}(c.ID)
	}

	if err := sem.Acquire(s.ctx, maxFetchParallelism); err != nil {
		log.Error().Err(err).Msg("failed to acquire semaphore")
	}

	// Starts are worked out after the inspects, because only an inspect has StartedAt
	// and a restart during the outage shows up as nothing else. A start the loop also
	// saw is announced once: notifyNewContainer dedupes on StartedAt.
	for _, c := range containers {
		cur, ok := s.containers.Load(c.ID)
		if !ok || cur.State != "running" {
			continue
		}
		before := previous[c.ID]
		_, stateChanged := changed[c.ID]
		restarted := before != nil && !before.StartedAt.IsZero() && !cur.StartedAt.IsZero() && !before.StartedAt.Equal(cur.StartedAt)
		if stateChanged || restarted {
			event("start", cur)
		}
	}

	log.Debug().Int("containers", len(containers)).Bool("full", full).Msg("finished refreshing container store")
	return missed, nil
}

// storeKeepingStats stores c, carrying over the stats history and mount stats of the
// entry it replaces. A list or inspect result always comes with an empty ring buffer,
// and swapping that in would reset every chart on each reconnect or refetch.
func (s *ContainerStore) storeKeepingStats(c Container) *Container {
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
func (s *ContainerStore) storeListed(before *Container, c Container, full bool) (stored bool, touched bool) {
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
func (s *ContainerStore) mergeFetched(prev *Container, fetched Container) (current *Container, found bool, updated bool) {
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

// userFilterIDs lists the containers a user's labels allow. It is bounded by
// defaultTimeout: on bare s.ctx a hung daemon hung the request forever.
func (s *ContainerStore) userFilterIDs(ctx context.Context, labels ContainerLabels) (map[string]Container, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()
	validContainers, err := s.client.ListContainers(ctx, labels)
	if err != nil {
		return nil, err
	}
	if len(validContainers) == 0 {
		log.Warn().Interface("userLabels", labels).Msg("no containers found with user labels")
	}
	return lo.KeyBy(validContainers, func(item Container) string {
		return item.ID
	}), nil
}

func (s *ContainerStore) ListContainers(ctx context.Context, labels ContainerLabels) ([]Container, error) {
	if err := s.waitReady(ctx); err != nil {
		return nil, err
	}

	if err := s.ensureFresh(ctx); err != nil {
		return nil, err
	}

	containers := make([]Container, 0)
	if labels.Exists() {
		validIDMap, err := s.userFilterIDs(ctx, labels)
		if err != nil {
			return nil, err
		}

		s.containers.Range(func(_ string, c *Container) bool {
			if _, ok := validIDMap[c.ID]; ok {
				containers = append(containers, *c)
			}
			return true
		})
	} else {
		s.containers.Range(func(_ string, c *Container) bool {
			containers = append(containers, *c)
			return true
		})
	}

	return containers, nil
}

func (s *ContainerStore) FindContainer(ctx context.Context, id string, labels ContainerLabels) (Container, error) {
	if err := s.waitReady(ctx); err != nil {
		return Container{}, err
	}
	if labels.Exists() {
		validIDMap, err := s.userFilterIDs(ctx, labels)
		if err != nil {
			return Container{}, err
		}

		if _, ok := validIDMap[id]; !ok {
			log.Warn().Str("id", id).Msg("user doesn't have access to container")
			return Container{}, ErrContainerNotFound
		}
	}

	if c, ok := s.containers.Load(id); !ok {
		log.Warn().Str("id", id).Msg("container not found")
		return Container{}, ErrContainerNotFound
	} else if c.FullyLoaded {
		return *c, nil
	}

	// One inspect per container however many requests want it, on the store's context
	// so a request that goes away neither fails it for the others nor logs an error.
	result := s.inspects.DoChan(id, func() (any, error) {
		return s.loadFully(id)
	})
	select {
	case r := <-result:
		if r.Err != nil {
			return Container{}, r.Err
		}
		return r.Val.(Container), nil
	case <-ctx.Done():
		return Container{}, ctx.Err()
	}
}

// loadFully inspects a container that is only known from a list and stores the result.
//
// The inspect can take up to defaultTimeout, so it runs outside any lock. Inside
// Compute it held the bucket lock for that long and stalled the event loop behind it
// whenever the loop touched a container in the same bucket.
func (s *ContainerStore) loadFully(id string) (Container, error) {
	prev, ok := s.containers.Load(id)
	if !ok {
		return Container{}, ErrContainerNotFound
	}
	if prev.FullyLoaded {
		return *prev, nil
	}

	log.Debug().Str("id", id).Msg("container is not fully loaded, fetching it")
	ctx, cancel := context.WithTimeout(s.ctx, defaultTimeout)
	defer cancel()
	fetched, err := s.client.FindContainer(ctx, id)
	if err != nil {
		log.Error().Err(err).Str("id", id).Msg("failed to fetch container")
		return *prev, nil
	}

	container, found, updated := s.mergeFetched(prev, fetched)
	if !found {
		log.Warn().Str("id", id).Msg("container was removed while fetching it")
		return Container{}, ErrContainerNotFound
	}

	if updated {
		go func() {
			// read back at send time: the event loop may have changed the entry since
			latest, ok := s.containers.Load(id)
			if !ok {
				return
			}
			s.broadcast(ContainerEvent{
				Name:      "update",
				Host:      latest.Host,
				ActorID:   id,
				Time:      time.Now(),
				Container: latest,
			})
		}()
	}

	return *container, nil
}

func (s *ContainerStore) Client() Client {
	return s.client
}

// startStats starts the stats collector in the background. Start blocks for the
// collector's whole life and returns true once the collector this call started has
// stopped; the history then has a gap in it, so it is cleared.
func (s *ContainerStore) startStats() {
	go func() {
		if s.statsCollector.Start(s.ctx) {
			s.containers.Range(func(_ string, c *Container) bool {
				if c.Stats != nil {
					c.Stats.Clear()
				}
				return true
			})
		}
	}()
}

func (s *ContainerStore) SubscribeEvents(ctx context.Context, events chan<- ContainerEvent) {
	s.startStats()

	s.subscribers.Store(ctx, &eventSubscriber{ch: events, name: subscriberNameFrom(ctx)})
	go func() {
		<-ctx.Done()
		s.subscribers.Delete(ctx)
		s.statsCollector.Stop()
	}()
}

func (s *ContainerStore) SubscribeStats(ctx context.Context, stats chan<- ContainerStat) {
	s.startStats()

	s.statsCollector.Subscribe(ctx, stats)
	go func() {
		<-ctx.Done()
		s.statsCollector.Stop()
	}()
}

func (s *ContainerStore) SubscribeNewContainers(ctx context.Context, containers chan<- Container) {
	s.newContainerSubscribers.Store(ctx, containers)
	go func() {
		<-ctx.Done()
		s.newContainerSubscribers.Delete(ctx)
	}()
}

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
// labels at all, so matchesLabels cannot stand in for it.
//
// FindContainer can fail transiently (the daemon is busy right after a compose recreate,
// or the inspect times out). Nothing re-adds the container afterwards, so it would stay
// missing from the store until the next reconnect and the UI would never update it again.
// Fall back to the list entry in that case: it is not FullyLoaded, so the next
// FindContainer fills in the rest.
func (s *ContainerStore) addContainer(id string, timeout time.Duration) (Container, bool) {
	ctx, cancel := context.WithTimeout(s.ctx, timeout)
	defer cancel()

	filtered := s.labels.Exists()
	if !filtered {
		if found, err := s.client.FindContainer(ctx, id); err == nil {
			return *s.storeKeepingStats(found), true
		} else {
			log.Warn().Err(err).Str("id", id).Msg("failed to inspect container, falling back to list entry")
		}
	}

	list, err := s.client.ListContainers(ctx, s.labels)
	if err != nil {
		log.Warn().Err(err).Str("id", id).Msg("failed to list containers while adding container")
		return Container{}, false
	}

	// make sure the container is in the list of containers when using filter
	listed, valid := lo.Find(list, func(item Container) bool {
		return item.ID == id
	})
	if !valid {
		return Container{}, false
	}

	if filtered {
		// inspected only now, so a container outside the filter never costs one
		if found, err := s.client.FindContainer(ctx, id); err == nil {
			return *s.storeKeepingStats(found), true
		} else {
			log.Warn().Err(err).Str("id", id).Msg("failed to inspect container, falling back to list entry")
		}
	}

	return *s.storeKeepingStats(listed), true
}

// notifyNewContainer tells new-container subscribers that c started, so they can
// begin streaming its logs.
//
// One start can reach here twice: docker often starts a container before the loop
// handles its create, so the create's inspect already sees it running and the start
// that follows announces it again. Each announcement starts a log stream, so it is
// deduplicated on StartedAt; a restart has a new StartedAt and is announced again.
func (s *ContainerStore) notifyNewContainer(found Container) {
	if last, ok := s.announced[found.ID]; ok && !found.StartedAt.IsZero() && last.Equal(found.StartedAt) {
		return
	}

	budget := &fanoutBudget{}
	defer budget.stop()

	delivered, dropped := 0, 0
	s.newContainerSubscribers.Range(func(c context.Context, containers chan<- Container) bool {
		switch sendBounded(c, containers, found, budget) {
		case sendOK:
			delivered++
		case sendCancelled:
			s.newContainerSubscribers.Delete(c)
		case sendDropped:
			dropped++
			log.Warn().
				Str("host", s.client.Host().Name).
				Str("id", found.ID).
				Msg("subscriber is not reading new containers, dropping container")
		}
		return true
	})

	// Recorded only once everyone has it. If the create's announcement was dropped, or
	// nobody was subscribed yet, the start that follows is the retry.
	if delivered > 0 && dropped == 0 && !found.StartedAt.IsZero() {
		if s.announced == nil {
			s.announced = make(map[string]time.Time)
		}
		s.announced[found.ID] = found.StartedAt
	}
}

func (s *ContainerStore) init() {
	stats := make(chan ContainerStat)
	s.statsCollector.Subscribe(s.ctx, stats)

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

	for {
		select {
		case event := <-s.events:
			log.Trace().Str("event", event.Name).Str("id", event.ActorID).Msg("received container event")
			switch event.Name {
			case "create":
				// docker follows a create with a start, which is what notifies. Only a
				// container that is already running here (a k8s pod created running)
				// would otherwise never be announced.
				if added, ok := s.addContainer(event.ActorID, 3*time.Second); ok && added.State == "running" {
					s.notifyNewContainer(added)
				}

			case "start":
				if added, ok := s.addContainer(event.ActorID, defaultTimeout); ok {
					s.notifyNewContainer(added)
				}
			case "destroy":
				log.Debug().Str("id", event.ActorID).Msg("container destroyed")
				s.containers.Delete(event.ActorID)
				delete(s.announced, event.ActorID)

			case "update":
				started, known := false, false
				updatedContainer, _ := s.containers.Compute(event.ActorID, func(c *Container, loaded bool) (*Container, xsync.ComputeOp) {
					if loaded && event.Container != nil {
						known = true
						newContainer := event.Container
						// A short-lived k8s pod (a Job) can go from Pending straight to
						// Succeeded without ever reporting Running. It still ran, so it
						// counts as a start, or the UI never learns it exists.
						if c.State != "running" && (newContainer.State == "running" || (c.State == "created" && newContainer.State == "exited")) {
							started = true
						}
						copy := *c
						copy.Name = newContainer.Name
						copy.State = newContainer.State
						copy.Labels = newContainer.Labels
						copy.StartedAt = newContainer.StartedAt
						copy.FinishedAt = newContainer.FinishedAt
						copy.Created = newContainer.Created
						copy.Host = newContainer.Host
						return &copy, xsync.UpdateOp
					} else {
						return c, xsync.CancelOp
					}
				})

				if started {
					s.broadcast(ContainerEvent{
						Name:    "start",
						ActorID: updatedContainer.ID,
						Host:    updatedContainer.Host,
					})
					// k8s sends create for a pending pod and only this update once it
					// runs, so this is where its start is announced
					s.notifyNewContainer(*updatedContainer)
				}

				// Only Kubernetes sends updates carrying a container the store never loaded:
				// its create landed before the store's first list, or adding it failed. A
				// k8s watch no longer ends and forces a fresh list, so pick it up here or
				// it stays missing for the life of the process.
				if !known && event.Container != nil && matchesLabels(event.Container.Labels, s.labels) {
					if added, ok := s.addContainer(event.ActorID, 3*time.Second); ok {
						s.broadcast(ContainerEvent{Name: "start", ActorID: added.ID, Host: added.Host})
						s.notifyNewContainer(added)
					}
				}

			case "die":
				s.containers.Compute(event.ActorID, func(c *Container, loaded bool) (*Container, xsync.ComputeOp) {
					if loaded {
						log.Debug().Str("id", c.ID).Msg("container died")
						copy := *c
						copy.State = "exited"
						copy.FinishedAt = time.Now()
						return &copy, xsync.UpdateOp
					} else {
						return c, xsync.CancelOp
					}
				})
			case "pause", "unpause":
				newState := "paused"
				if event.Name == "unpause" {
					newState = "running"
				}
				s.containers.Compute(event.ActorID, func(c *Container, loaded bool) (*Container, xsync.ComputeOp) {
					if loaded {
						log.Debug().Str("id", c.ID).Str("state", newState).Msg("container state changed")
						copy := *c
						copy.State = newState
						return &copy, xsync.UpdateOp
					} else {
						return c, xsync.CancelOp
					}
				})
			case "health_status: healthy", "health_status: unhealthy":
				healthy := "unhealthy"
				if event.Name == "health_status: healthy" {
					healthy = "healthy"
				}

				s.containers.Compute(event.ActorID, func(c *Container, loaded bool) (*Container, xsync.ComputeOp) {
					if loaded {
						log.Debug().Str("id", c.ID).Str("health", healthy).Msg("container health status changed")
						copy := *c
						copy.Health = healthy
						return &copy, xsync.UpdateOp
					} else {
						return c, xsync.CancelOp
					}
				})

			case "rename":
				s.containers.Compute(event.ActorID, func(c *Container, loaded bool) (*Container, xsync.ComputeOp) {
					if !loaded {
						return c, xsync.CancelOp
					}
					// A dev.dozzle.name or coolify.serviceName label pins a custom
					// display name (see newContainer). That name must survive a
					// docker-level rename, so only follow the rename when the name
					// actually comes from Docker.
					if c.Labels["dev.dozzle.name"] != "" || c.Labels["coolify.serviceName"] != "" {
						log.Debug().Str("id", event.ActorID).Msg("ignoring rename: container has a custom name label")
						return c, xsync.CancelOp
					}
					log.Debug().Str("id", event.ActorID).Str("name", event.ActorAttributes["name"]).Msg("container renamed")
					copy := *c
					copy.Name = event.ActorAttributes["name"]
					return &copy, xsync.UpdateOp
				})
			}
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

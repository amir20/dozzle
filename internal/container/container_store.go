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
	wg                      sync.WaitGroup
	// connected guards the event stream goroutine only: it is true while one is running.
	connected atomic.Bool
	// staleGen is bumped every time the event stream (re)connects, and freshGen records
	// the staleGen the last successful list covered. They differ while the map may be
	// missing whatever happened while no stream was running, including when the very
	// first list failed. A generation rather than a flag, so a reconnect that lands
	// while a refresh is in flight is not cleared by that refresh finishing.
	staleGen atomic.Uint64
	freshGen atomic.Uint64
	// refreshMu serialises refreshes and makes concurrent readers wait for one in
	// flight instead of reading a map that is still being rebuilt.
	refreshMu sync.Mutex
	// announced is the StartedAt each container was last announced to new-container
	// subscribers with. Only the event loop touches it.
	announced map[string]time.Time
	events    chan ContainerEvent
	ctx       context.Context
	labels    ContainerLabels
}

const defaultTimeout = 10 * time.Second

func NewContainerStore(ctx context.Context, client Client, statsCollect StatsCollector, labels ContainerLabels) *ContainerStore {
	log.Debug().Str("host", client.Host().Name).Interface("labels", labels).Msg("initializing container store")

	s := &ContainerStore{
		containers:              xsync.NewMap[string, *Container](),
		client:                  client,
		subscribers:             xsync.NewMap[context.Context, *eventSubscriber](),
		newContainerSubscribers: xsync.NewMap[context.Context, chan<- Container](),
		statsCollector:          statsCollect,
		wg:                      sync.WaitGroup{},
		events:                  make(chan ContainerEvent),
		ctx:                     ctx,
		labels:                  labels,
	}
	s.volumeMonitor = newVolumeMonitor(s)
	s.volumeMonitor.start(ctx)

	s.wg.Add(1)

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

// checkConnectivity starts the event stream if none is running and lists containers
// when the map may be out of date. A failed list leaves the map stale, so the next
// caller retries it instead of serving an empty store until the stream drops.
func (s *ContainerStore) checkConnectivity() error {
	if s.connected.CompareAndSwap(false, true) {
		// marked before the stream starts: whatever happened while no stream was
		// running is only recovered by a list
		s.staleGen.Add(1)
		go func() {
			log.Debug().Str("host", s.client.Host().Name).Msg("docker store subscribing docker events")
			err := s.client.ContainerEvents(s.ctx, s.events)
			if err != nil && !errors.Is(err, context.Canceled) {
				log.Error().Err(err).Str("host", s.client.Host().Name).Msg("docker store unexpectedly disconnected from docker events")
			}
			s.connected.Store(false)
		}()
	}

	if s.staleGen.Load() == s.freshGen.Load() {
		return nil
	}

	s.refreshMu.Lock()
	defer s.refreshMu.Unlock()

	gen := s.staleGen.Load()
	if gen == s.freshGen.Load() {
		// another caller refreshed while this one waited for the lock
		return nil
	}

	if err := s.refresh(); err != nil {
		return err
	}
	s.freshGen.Store(gen)
	return nil
}

// refresh reconciles the map with a fresh list. It never clears the map first:
// concurrent readers would see it empty or half built, and a container the event loop
// added while the list was in flight would be wiped. Only IDs that were already in the
// map before the list, and that the list no longer reports, are removed.
func (s *ContainerStore) refresh() error {
	previous := make(map[string]*Container)
	s.containers.Range(func(id string, c *Container) bool {
		previous[id] = c
		return true
	})

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	containers, err := s.client.ListContainers(ctx, s.labels)
	if err != nil {
		return err
	}

	listed := make(map[string]struct{}, len(containers))
	for _, c := range containers {
		listed[c.ID] = struct{}{}
		s.storeListed(previous[c.ID], c)
	}
	for id, before := range previous {
		if _, ok := listed[id]; ok {
			continue
		}
		// Only drop the entry the snapshot saw. If the loop replaced it during the list,
		// the container came back after the list was taken (a k8s StatefulSet pod keeps
		// its ID across a recreate) and deleting it would lose it for good.
		s.containers.Compute(id, func(c *Container, loaded bool) (*Container, xsync.ComputeOp) {
			if loaded && c == before {
				return nil, xsync.DeleteOp
			}
			return c, xsync.CancelOp
		})
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
			ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
			defer cancel()
			if fetched, err := s.client.FindContainer(ctx, id); err == nil {
				s.mergeFetched(prev, fetched)
			}
		}(c.ID)
	}

	if err := sem.Acquire(s.ctx, maxFetchParallelism); err != nil {
		log.Error().Err(err).Msg("failed to acquire semaphore")
	}

	log.Debug().Int("containers", len(containers)).Msg("finished initializing container store")
	return nil
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

// storeListed stores a list entry taken during a refresh. before is what the map
// held when the refresh started. If the entry changed while the list was in flight,
// the event loop got there first (a die, a pause, a health change) and its state is
// newer than the list's, so the list must not write the older state back.
func (s *ContainerStore) storeListed(before *Container, c Container) {
	s.containers.Compute(c.ID, func(existing *Container, loaded bool) (*Container, xsync.ComputeOp) {
		if !loaded {
			return &c, xsync.UpdateOp
		}
		if existing != before {
			if before == nil || existing.FullyLoaded {
				// stored by the event loop during the list from an inspect, which is
				// newer than the list and complete
				return existing, xsync.CancelOp
			}
			keepLoopFields(existing, &c)
		}
		carryOverStats(existing, &c)
		return &c, xsync.UpdateOp
	})
}

// keepLoopFields copies the fields the event loop maintains from events onto a
// snapshot that was taken before those events were applied.
func keepLoopFields(from *Container, to *Container) {
	to.State = from.State
	to.Health = from.Health
	to.StartedAt = from.StartedAt
	to.FinishedAt = from.FinishedAt
	to.Name = from.Name
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
		if c != prev {
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

func (s *ContainerStore) ListContainers(labels ContainerLabels) ([]Container, error) {
	s.wg.Wait()

	if err := s.checkConnectivity(); err != nil {
		return nil, err
	}

	containers := make([]Container, 0)
	if labels.Exists() {
		validContainers, err := s.client.ListContainers(s.ctx, labels)
		if err != nil {
			return nil, err
		}

		if len(validContainers) == 0 {
			log.Warn().Interface("userLabels", labels).Msg("no containers found with user labels")
		}

		validIDMap := lo.KeyBy(validContainers, func(item Container) string {
			return item.ID
		})

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

func (s *ContainerStore) FindContainer(id string, labels ContainerLabels) (Container, error) {
	s.wg.Wait()
	if labels.Exists() {
		validContainers, err := s.client.ListContainers(s.ctx, labels)
		if err != nil {
			return Container{}, err
		}

		validIDMap := lo.KeyBy(validContainers, func(item Container) string {
			return item.ID
		})

		if _, ok := validIDMap[id]; !ok {
			log.Warn().Str("id", id).Msg("user doesn't have access to container")
			return Container{}, ErrContainerNotFound
		}
	}

	prev, ok := s.containers.Load(id)
	if !ok {
		log.Warn().Str("id", id).Msg("container not found")
		return Container{}, ErrContainerNotFound
	}
	if prev.FullyLoaded {
		return *prev, nil
	}

	// The inspect can take up to defaultTimeout, so it runs outside any lock. Inside
	// Compute it held the bucket lock for that long and stalled the event loop behind
	// it whenever the loop touched a container in the same bucket.
	log.Debug().Str("id", id).Msg("container is not fully loaded, fetching it")
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
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
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
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
	if !found.StartedAt.IsZero() {
		if s.announced == nil {
			s.announced = make(map[string]time.Time)
		}
		if last, ok := s.announced[found.ID]; ok && last.Equal(found.StartedAt) {
			return
		}
		s.announced[found.ID] = found.StartedAt
	}

	budget := &fanoutBudget{}
	defer budget.stop()

	s.newContainerSubscribers.Range(func(c context.Context, containers chan<- Container) bool {
		switch sendBounded(c, containers, found, budget) {
		case sendCancelled:
			s.newContainerSubscribers.Delete(c)
		case sendDropped:
			log.Warn().
				Str("host", s.client.Host().Name).
				Str("id", found.ID).
				Msg("subscriber is not reading new containers, dropping container")
		}
		return true
	})
}

func (s *ContainerStore) init() {
	stats := make(chan ContainerStat)
	s.statsCollector.Subscribe(s.ctx, stats)

	if err := s.checkConnectivity(); err != nil {
		// the store stays stale, so the next ListContainers retries the list
		log.Error().Err(err).Str("host", s.client.Host().Name).Msg("failed to list containers while initializing container store")
	}

	s.wg.Done()

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

package container

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"github.com/puzpuzpuz/xsync/v4"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
	"golang.org/x/sync/singleflight"
)

type StatsCollector interface {
	Start(parentCtx context.Context) bool
	Subscribe(ctx context.Context, stats chan<- ContainerStat)
	Stop()
}

// Store is one host's containers, kept current from the engine's event
// stream, plus the fan-out of those events to subscribers.
//
// Three things write to the map, and the rules between them are the whole design:
//
//   - the event loop (store_events.go) applies events in order on one
//     goroutine. What it writes is the newest truth there is.
//   - a refresh (store_refresh.go) lists everything after the stream
//     (re)connects, because the engine does not replay what was sent while nobody
//     listened. It yields to anything the loop changed while the list was in flight,
//     and replays to the loop whatever it found that the stream never said.
//   - FindContainer inspects a container that is only known from a list, outside any
//     lock, and merges the result the same way.
//
// Entries are immutable: every change stores a new *Container (see patch), which is
// what lets a refresh or an inspect tell whether the loop got there first.
//
// store_fanout.go is delivery: bounded sends, so one stalled subscriber
// costs its own messages and never the loop.
type Store struct {
	containers     *xsync.Map[string, *Container]
	client         Client
	labels         ContainerLabels
	statsCollector StatsCollector
	volumeMonitor  *volumeMonitor
	ctx            context.Context
	timing         storeTiming

	// the event loop
	events chan ContainerEvent
	// ready is closed once boot's first refresh has run, so callers can stop waiting
	// on it when their own context ends.
	ready chan struct{}
	// announced is the StartedAt each container was last announced to new-container
	// subscribers with. Only the event loop touches it.
	announced map[string]time.Time

	// subscribers
	subscribers             *xsync.Map[context.Context, *eventSubscriber]
	newContainerSubscribers *xsync.Map[context.Context, chan<- Container]

	// refreshing
	//
	// staleGen is bumped every time the event stream reconnects, and freshGen records
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
}

const defaultTimeout = 10 * time.Second

func NewStore(ctx context.Context, client Client, statsCollect StatsCollector, labels ContainerLabels) *Store {
	return newStore(ctx, client, statsCollect, labels, defaultStoreTiming)
}

func newStore(ctx context.Context, client Client, statsCollect StatsCollector, labels ContainerLabels, timing storeTiming) *Store {
	log.Debug().Str("host", client.Host().Name).Interface("labels", labels).Msg("initializing container store")

	s := &Store{
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

	go s.run()

	return s
}

// applyMountStats updates a container's MountStats and broadcasts an "update"
// event so subscribers (SSE) can propagate the new data to clients.
func (s *Store) applyMountStats(id string, stats map[string]MountStat) {
	updated, ok := s.patch(id, func(c *Container) bool {
		c.MountStats = stats
		return true
	})
	if !ok {
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

// waitReady blocks until init's first refresh has run, or ctx ends.
func (s *Store) waitReady(ctx context.Context) error {
	select {
	case <-s.ready:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// userFilterIDs lists the containers a user's labels allow. It is bounded by
// defaultTimeout: on bare s.ctx a hung daemon hung the request forever.
func (s *Store) userFilterIDs(ctx context.Context, labels ContainerLabels) (map[string]Container, error) {
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

func (s *Store) ListContainers(ctx context.Context, labels ContainerLabels) ([]Container, error) {
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

func (s *Store) FindContainer(ctx context.Context, id string, labels ContainerLabels) (Container, error) {
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
func (s *Store) loadFully(id string) (Container, error) {
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

func (s *Store) Client() Client {
	return s.client
}

// startStats starts the stats collector in the background. Start blocks for the
// collector's whole life and returns true once the collector this call started has
// stopped; the history then has a gap in it, so it is cleared.
func (s *Store) startStats() {
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

func (s *Store) SubscribeEvents(ctx context.Context, events chan<- ContainerEvent) {
	s.startStats()

	s.subscribers.Store(ctx, &eventSubscriber{ch: events, name: subscriberNameFrom(ctx)})
	go func() {
		<-ctx.Done()
		s.subscribers.Delete(ctx)
		s.statsCollector.Stop()
	}()
}

func (s *Store) SubscribeStats(ctx context.Context, stats chan<- ContainerStat) {
	s.startStats()

	s.statsCollector.Subscribe(ctx, stats)
	go func() {
		<-ctx.Done()
		s.statsCollector.Stop()
	}()
}

func (s *Store) SubscribeNewContainers(ctx context.Context, containers chan<- Container) {
	s.newContainerSubscribers.Store(ctx, containers)
	go func() {
		<-ctx.Done()
		s.newContainerSubscribers.Delete(ctx)
	}()
}

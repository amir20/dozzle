package container

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
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

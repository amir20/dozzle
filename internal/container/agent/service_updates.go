package agent

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/rs/zerolog/log"
)

const (
	// recentUpdatesTimeout bounds one read of the agent's update events.
	// RecentUpdates has no context of its own, like the Docker service's.
	recentUpdatesTimeout = 5 * time.Second
	// updateStreamRetry is the wait before the update stream is opened again
	// after it broke, typically because the agent restarted.
	updateStreamRetry = 5 * time.Second
	// noHistoryRetry is the wait for an agent without the RPC. It is tried
	// again in case the agent is upgraded behind the same address.
	noHistoryRetry = 5 * time.Minute
)

// An agent host keeps its own update events, since the container store that
// works them out runs on the agent. The service reads them over gRPC so the
// server can treat an agent host like a local one.
var _ container.UpdateHistory = (*service)(nil)

// RecentUpdates is the agent's kept update events, oldest first. It is empty
// when the agent cannot be reached or is too old to keep them.
func (a *service) RecentUpdates() []container.ContainerUpdateEvent {
	ctx, cancel := context.WithTimeout(context.Background(), recentUpdatesTimeout)
	defer cancel()
	events, err := a.recentUpdates(ctx)
	if err != nil {
		return nil
	}
	return events
}

func (a *service) recentUpdates(ctx context.Context) ([]container.ContainerUpdateEvent, error) {
	events, err := a.client.RecentUpdates(ctx)
	if err != nil {
		a.logUpdatesErr(err)
		return nil, err
	}
	return events, nil
}

func (a *service) logUpdatesErr(err error) {
	if errors.Is(err, ErrNoUpdateHistory) {
		if a.noHistoryLogged.CompareAndSwap(false, true) {
			log.Debug().Str("agent", a.client.endpoint).Msg("agent keeps no update history, it is older than this server")
		}
		return
	}
	if !errors.Is(err, context.Canceled) {
		log.Debug().Err(err).Str("agent", a.client.endpoint).Msg("could not read update events from agent")
	}
}

// SubscribeUpdates sends the agent's update events to ch until ctx ends. The
// stream is opened again whenever it breaks, and the events the agent kept
// meanwhile are sent then, so an agent restart or a network blip loses none
// the agent still has. Sends wait for ch, which only holds up the stream.
func (a *service) SubscribeUpdates(ctx context.Context, ch chan<- container.ContainerUpdateEvent) {
	a.updateStreams.Go(func() { a.streamUpdates(ctx, ch) })
}

func (a *service) streamUpdates(ctx context.Context, ch chan<- container.ContainerUpdateEvent) {
	seen := newSeenUpdates()
	baseline := false
	for {
		err := a.streamUpdatesOnce(ctx, ch, seen, &baseline)
		if ctx.Err() != nil {
			return
		}
		a.logUpdatesErr(err)
		wait := a.updateRetry
		if errors.Is(err, ErrNoUpdateHistory) {
			wait = a.noHistoryRetry
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}
	}
}

// streamUpdatesOnce opens the stream, catches up from the agent's kept events,
// then forwards live ones until the stream breaks. The first successful
// catch-up only marks what the agent already had, since the subscriber asked
// for new events.
func (a *service) streamUpdatesOnce(ctx context.Context, ch chan<- container.ContainerUpdateEvent, seen *seenUpdates, baseline *bool) error {
	streamCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Unbuffered, so nothing is left behind in it when the stream breaks.
	live := make(chan container.ContainerUpdateEvent)
	errCh := make(chan error, 1)
	go func() { errCh <- a.client.StreamUpdates(streamCtx, live) }()

	forward := func(e container.ContainerUpdateEvent) bool {
		if !seen.add(e) {
			return true
		}
		select {
		case ch <- e:
			return true
		case <-ctx.Done():
			return false
		}
	}

	// The stream is opened first, so an event between this read and the
	// stream starting is in one or the other, and seen drops the copy.
	recentCtx, cancelRecent := context.WithTimeout(streamCtx, recentUpdatesTimeout)
	kept, err := a.client.RecentUpdates(recentCtx)
	cancelRecent()
	if err == nil {
		for _, e := range kept {
			if !*baseline {
				seen.add(e)
				continue
			}
			if !forward(e) {
				return ctx.Err()
			}
		}
		*baseline = true
	}

	for {
		select {
		case e := <-live:
			if !forward(e) {
				return ctx.Err()
			}
		case err := <-errCh:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// seenUpdates remembers the last events forwarded, so a catch-up after a
// reconnect does not send one twice. The agent keeps 20, so twice that is
// always enough.
type seenUpdates struct {
	keys []updateKey
}

type updateKey struct {
	engineName string
	toImageID  string
	at         time.Time
	rolledBack bool
}

const seenUpdatesSize = 40

func newSeenUpdates() *seenUpdates { return &seenUpdates{} }

// add reports whether e is new, and remembers it.
func (s *seenUpdates) add(e container.ContainerUpdateEvent) bool {
	key := updateKey{engineName: e.EngineName, toImageID: e.ToImageID, at: e.At, rolledBack: e.RolledBack}
	if slices.ContainsFunc(s.keys, func(k updateKey) bool {
		return k.engineName == key.engineName && k.toImageID == key.toImageID && k.at.Equal(key.at) && k.rolledBack == key.rolledBack
	}) {
		return false
	}
	s.keys = append(s.keys, key)
	if over := len(s.keys) - seenUpdatesSize; over > 0 {
		s.keys = slices.Delete(s.keys, 0, over)
	}
	return true
}

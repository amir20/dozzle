package web

import (
	"cmp"
	"context"
	"slices"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/rs/zerolog/log"
)

const (
	// backfillMatches is how many matches the search collects before it stops.
	backfillMatches = 50
	// backfillFirstWindow is the newest slice read on its own. A common match
	// fills the cap here; anything sparser falls through to one read of the rest.
	backfillFirstWindow = 10 * time.Second
)

// searchProgressEvery throttles progress events during the long read.
var searchProgressEvery = 250 * time.Millisecond

// searchStatus reports progress of the filtered backfill to the frontend.
// scannedTo is how far back the results reach; reason is only set when done.
type searchStatus struct {
	ScannedTo time.Time `json:"scannedTo"`
	Matches   int       `json:"matches"`
	// Progress is the share of the long read done so far, from 0 to 1.
	Progress float64 `json:"progress,omitempty"`
	Done     bool    `json:"done"`
	Reason   string  `json:"reason,omitempty"`
}

// searchBackfill finds the newest backfillMatches matches older than `to`, down
// to floor (zero for none), across every container. It reads the newest
// backfillFirstWindow first and, if that falls short of the cap, everything
// older in one read per container. A read with `since` makes Docker decode the
// log from its start however small the window, so one long read beats a walk
// of growing windows that each pays that cost again. Matches go out oldest first
// on backfill and progress on status. Every send gives up once ctx is done, so
// the search never outlives the client.
func searchBackfill(
	ctx context.Context,
	services []*container.ContainerService,
	to time.Time,
	floor time.Time,
	stdTypes container.StdType,
	filter logFilter,
	backfill chan<- []*container.LogEvent,
	status chan<- searchStatus,
) {
	found := 0
	scannedTo := to
	send := func(s searchStatus) {
		select {
		case status <- s:
		case <-ctx.Done():
		}
	}
	// Always emit exactly one terminal status, whatever exit fires (ran out
	// of logs, hit the cap, or errored). Without this the frontend would keep
	// suppressing the empty state and spin forever. "exhausted" is the default
	// for running out of logs and for error/early returns; "capped" is set only
	// when the search stops at the match cap.
	reason := "exhausted"
	defer func() {
		send(searchStatus{ScannedTo: scannedTo, Matches: found, Done: true, Reason: reason})
	}()
	emit := func(events []*container.LogEvent) bool {
		found += len(events)
		if len(events) == 0 {
			return true
		}
		select {
		case backfill <- events:
			return true
		case <-ctx.Done():
			return false
		}
	}

	if !floor.IsZero() && !to.After(floor) {
		return
	}
	from := to.Add(-backfillFirstWindow)
	if !floor.IsZero() && from.Before(floor) {
		from = floor
	}
	events, seen, err := scanMatches(ctx, services, from, to, stdTypes, filter, backfillMatches, nil)
	if err != nil {
		log.Error().Err(err).Msg("error while fetching logs")
		return
	}
	if !emit(events) {
		return
	}
	scannedTo = from
	if seen > len(events) {
		// matches older than the oldest one kept were dropped
		scannedTo = time.UnixMilli(events[0].Timestamp)
	}
	if found >= backfillMatches {
		reason = "capped"
		return
	}
	send(searchStatus{ScannedTo: scannedTo, Matches: found})

	// Everything older, back to the floor or the oldest container's birth.
	oldest := from
	for _, s := range services {
		if s.Container.Created.Before(oldest) {
			oldest = s.Container.Created
		}
	}
	if !floor.IsZero() && oldest.Before(floor) {
		oldest = floor
	}
	if !oldest.Before(from) {
		return
	}

	remaining := backfillMatches - found
	lastSent := time.Now()
	events, seen, err = scanMatches(ctx, services, oldest, from, stdTypes, filter, remaining, func(progress float64, seen int) {
		if time.Since(lastSent) < searchProgressEvery {
			return
		}
		lastSent = time.Now()
		send(searchStatus{ScannedTo: scannedTo, Matches: found + min(seen, remaining), Progress: progress})
	})
	if err != nil {
		log.Error().Err(err).Msg("error while fetching logs")
		return
	}
	if !emit(events) {
		return
	}
	if seen > len(events) {
		// more matches exist before the oldest one kept
		reason = "capped"
		scannedTo = time.UnixMilli(events[0].Timestamp)
		return
	}
	scannedTo = oldest
}

// scanMatches reads [from, to) of every container, clamped to its creation, and
// returns the newest `limit` matches oldest first along with how many matched in
// all. progress, when set, is called on every line with the share of the whole
// read done so far, measured in time, and the running match count.
func scanMatches(
	ctx context.Context,
	services []*container.ContainerService,
	from time.Time,
	to time.Time,
	stdTypes container.StdType,
	filter logFilter,
	limit int,
	progress func(progress float64, seen int),
) ([]*container.LogEvent, int, error) {
	starts := make([]time.Time, len(services))
	var total time.Duration
	for i, s := range services {
		starts[i] = from
		if s.Container.Created.After(from) {
			starts[i] = s.Container.Created
		}
		if starts[i].Before(to) {
			total += to.Sub(starts[i])
		}
	}

	var matches []*container.LogEvent
	var done time.Duration
	seen := 0
	for i, s := range services {
		start := starts[i]
		if !start.Before(to) {
			continue
		}
		logs, err := s.LogsBetweenDates(ctx, start, to, stdTypes)
		if err != nil {
			return nil, 0, err
		}
		var kept []*container.LogEvent
		for event := range logs {
			// k8s ignores the end of the range and reads to now, which would
			// refill the newest `limit` with lines already shown.
			if !time.UnixMilli(event.Timestamp).Before(to) {
				continue
			}
			if filter.matches(event) {
				seen++
				kept = append(kept, event)
				// One container's lines come oldest first, so the newest `limit`
				// are the tail. Trimming at twice the limit keeps it amortized.
				if len(kept) >= 2*limit {
					kept = append(kept[:0], kept[len(kept)-limit:]...)
				}
			}
			if progress != nil {
				at := min(max(time.UnixMilli(event.Timestamp).Sub(start), 0), to.Sub(start))
				progress(float64(done+at)/float64(total), seen)
			}
		}
		matches = append(matches, kept...)
		done += to.Sub(start)
	}

	// Stable so that events sharing a timestamp keep the per-container order
	// they were collected in.
	slices.SortStableFunc(matches, func(a, b *container.LogEvent) int {
		return cmp.Compare(a.Timestamp, b.Timestamp)
	})
	if len(matches) > limit {
		matches = matches[len(matches)-limit:]
	}
	return matches, seen, nil
}

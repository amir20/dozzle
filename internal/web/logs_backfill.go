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
	// backfillMatches is how many matches the walk collects before it stops.
	backfillMatches = 50
	// backfillFirstWindow is the first window scanned; each pass doubles it.
	backfillFirstWindow = 10 * time.Second
)

// searchStatus reports progress of the filtered backfill walk to the frontend.
// scannedTo is the oldest boundary scanned so far; reason is only set when done.
type searchStatus struct {
	ScannedTo time.Time `json:"scannedTo"`
	Matches   int       `json:"matches"`
	Done      bool      `json:"done"`
	Reason    string    `json:"reason,omitempty"`
}

// searchBackfill walks back from `to` across every container in doubling windows,
// sending each pass's matches oldest first on backfill and its progress on status,
// until it has backfillMatches of them or scans past the oldest container's birth.
// Every send gives up once ctx is done, so the walk never outlives the client.
func searchBackfill(
	ctx context.Context,
	services []*container.ContainerService,
	to time.Time,
	stdTypes container.StdType,
	filter logFilter,
	backfill chan<- []*container.LogEvent,
	status chan<- searchStatus,
) {
	remaining := backfillMatches
	found := 0
	delta := -backfillFirstWindow
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
	// when the loop completes by reaching the match cap.
	reason := "exhausted"
	defer func() {
		send(searchStatus{ScannedTo: to, Matches: found, Done: true, Reason: reason})
	}()

	for remaining > 0 {
		events := make([]*container.LogEvent, 0)
		stillRunning := false
		for _, containerService := range services {
			if to.Before(containerService.Container.Created) {
				continue
			}

			logs, err := containerService.LogsBetweenDates(ctx, to.Add(delta), to, stdTypes)
			if err != nil {
				log.Error().Err(err).Msg("error while fetching logs")
				return
			}

			for event := range logs {
				if filter.matches(event) {
					events = append(events, event)
				}
			}

			stillRunning = true
		}

		if !stillRunning {
			// scanned past the oldest container's birth: nothing older exists
			return
		}

		to = to.Add(delta)
		delta *= 2
		remaining -= len(events)
		found += len(events)
		// Stable so that events sharing a timestamp keep the per-container
		// order they were collected in rather than shuffling between passes.
		slices.SortStableFunc(events, func(a, b *container.LogEvent) int {
			return cmp.Compare(a.Timestamp, b.Timestamp)
		})
		if len(events) > 0 {
			select {
			case backfill <- events:
			case <-ctx.Done():
				return
			}
		}
		send(searchStatus{ScannedTo: to, Matches: found, Done: false})
	}
	// accumulated enough matches; more may exist further back
	reason = "capped"
}

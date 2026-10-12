package web

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/utils"
	"github.com/amir20/dozzle/internal/web/search"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

// defaultFetchSize caps how many events a windowed fetch returns without `min`.
const defaultFetchSize = 500

// fetchWindow is the paging a historical load asks for: the time range, and the
// ids that pin where the returned page starts and stops.
type fetchWindow struct {
	from, to time.Time
	// size is the ring buffer's capacity, which is `min` when that is set.
	size int
	// minimum keeps widening the window back until this many matches; 0 fetches once.
	minimum    int
	maxStart   int
	lastSeenId uint32
	startId    uint32
	// windowed is set when the request names any of the paging params above.
	// Without them a fetch is the whole log.
	windowed bool
}

func parseFetchWindow(q url.Values) (fetchWindow, error) {
	from, _ := time.Parse(time.RFC3339Nano, q.Get("from"))
	to, _ := time.Parse(time.RFC3339Nano, q.Get("to"))
	win := fetchWindow{from: from, to: to, size: defaultFetchSize, maxStart: math.MaxInt}
	for _, key := range [...]string{"from", "to", "min", "maxStart", "lastSeenId", "startId"} {
		if q.Has(key) {
			win.windowed = true
			break
		}
	}

	if q.Has("min") {
		minimum, err := strconv.Atoi(q.Get("min"))
		if err != nil {
			return win, err
		}
		if minimum < 0 || minimum > win.size {
			return win, errors.New("minimum must be between 0 and buffer size")
		}
		win.minimum = minimum
		win.size = minimum
	}

	if q.Has("maxStart") {
		maxStart, err := strconv.Atoi(q.Get("maxStart"))
		if err != nil {
			return win, err
		}
		if maxStart < 1 || maxStart > win.size {
			return win, errors.New("invalid maxStart")
		}
		win.maxStart = maxStart
	}

	if q.Has("lastSeenId") {
		win.to = win.to.Add(50 * time.Millisecond)
		num, err := strconv.ParseUint(q.Get("lastSeenId"), 10, 32)
		if err != nil {
			return win, err
		}
		win.lastSeenId = uint32(num)
	}

	if q.Has("startId") {
		win.from = win.from.Add(-50 * time.Millisecond)
		num, err := strconv.ParseUint(q.Get("startId"), 10, 32)
		if err != nil {
			return win, err
		}
		win.startId = uint32(num)
	}

	return win, nil
}

// compressedWriter gzips the body when the client accepts it. Call it only once
// nothing can fail with an http.Error, which would go out under a gzip header.
func compressedWriter(w http.ResponseWriter, r *http.Request) (io.Writer, func()) {
	if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		return w, func() {}
	}
	w.Header().Set("Content-Encoding", "gzip")
	gzWriter := gzip.NewWriter(w)
	return gzWriter, func() { gzWriter.Close() }
}

// writeAllLogs streams a container's whole history, as plain text for a copy or as
// JSON lines for the SQL view, which only gets the complex events. Unlike the
// windowed fetch, an empty level set here means every level.
func writeAllLogs(writer io.Writer, events <-chan *container.LogEvent, filter logFilter, plainText bool) {
	encoder := json.NewEncoder(writer)
	for event := range events {
		// Grouped events carry a []string message, so filtering on "not a
		// string" still lets arrays through and breaks struct inference for
		// the SQL view's unnest(m).
		if !plainText && event.Type != container.LogTypeComplex {
			continue
		}
		if !filter.matchesText(event) {
			continue
		}
		if len(filter.levels) > 0 && !filter.matchesLevel(event) {
			continue
		}
		if plainText {
			// Expand grouped events into their fragment lines; grouped
			// events store their lines in Message and have an empty
			// RawMessage, so writing RawMessage alone drops every group.
			fmt.Fprintf(writer, "%s\n", event.PlainText())
		} else if err := encoder.Encode(event); err != nil {
			log.Error().Err(err).Msg("error encoding log event")
		}
	}
}

// fetchWindowedLogs returns the matching events in win, widening the window back
// in doubling steps until it holds win.minimum of them or reaches the container's
// creation.
func fetchWindowedLogs(ctx context.Context, containerService *container.ContainerService, win fetchWindow, filter logFilter, stdTypes container.StdType) ([]*container.LogEvent, error) {
	buffer := utils.NewRingBuffer[*container.LogEvent](win.size)
	delta := max(win.to.Sub(win.from), time.Second*3)
	from := win.from
	startIdFound := win.startId == 0

	for {
		if win.minimum > 0 && buffer.Len() >= win.minimum {
			break
		}

		buffer.Clear()

		// Each read gets its own context, cancelled once it is done: a break below
		// would otherwise leave the engine's log stream open until the request ends.
		readCtx, cancel := context.WithCancel(ctx)
		events, err := containerService.LogsBetweenDates(readCtx, from, win.to, stdTypes)
		if err != nil {
			cancel()
			return nil, err
		}

		for event := range events {
			if !filter.matches(event) {
				continue
			}

			if !startIdFound {
				if event.Id == win.startId {
					log.Debug().Uint32("startId", win.startId).Msg("found start id, will include subsequent events")
					startIdFound = true
				}
				continue
			}

			if win.lastSeenId != 0 && event.Id == win.lastSeenId {
				log.Debug().Uint32("lastSeenId", win.lastSeenId).Msg("found last seen id")
				break
			}

			if buffer.Len() >= win.maxStart {
				break
			}

			search.EscapeHTMLValues(event)
			buffer.Push(event)
		}
		cancel()

		if from.Before(containerService.Container.Created) || win.minimum == 0 {
			break
		}

		from = from.Add(-delta)
		delta = delta * 2
	}

	return buffer.Data(), nil
}

// windowedFetchConcurrency caps how many containers one windowed fetch reads at a
// time. Each read is bounded by the window's size, so a few at once is cheap, and
// a big stack still never opens a log stream per container all at once.
const windowedFetchConcurrency = 4

// fetchWindowedLogsMerged runs the windowed fetch for each container, `min`
// applying to each one, and merges the results by time.
func fetchWindowedLogsMerged(ctx context.Context, services []*container.ContainerService, win fetchWindow, filter logFilter, stdTypes container.StdType) ([]*container.LogEvent, error) {
	results := make([][]*container.LogEvent, len(services))
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(windowedFetchConcurrency)
	for i, service := range services {
		g.Go(func() error {
			events, err := fetchWindowedLogs(ctx, service, win, filter, stdTypes)
			results[i] = events
			return err
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return mergeByTime(results), nil
}

// mergeByTime merges lists that are each in their container's order into one,
// oldest first. Each list keeps its own order, and ties go to the earlier list.
func mergeByTime(lists [][]*container.LogEvent) []*container.LogEvent {
	if len(lists) == 1 {
		return lists[0]
	}
	total := 0
	for _, list := range lists {
		total += len(list)
	}
	merged := make([]*container.LogEvent, 0, total)
	heads := make([]int, len(lists))
	for len(merged) < total {
		next := -1
		for i, list := range lists {
			if heads[i] < len(list) && (next < 0 || list[heads[i]].Timestamp < lists[next][heads[next]].Timestamp) {
				next = i
			}
		}
		merged = append(merged, lists[next][heads[next]])
		heads[next]++
	}
	return merged
}

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
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
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
}

func parseFetchWindow(q url.Values) (fetchWindow, error) {
	from, _ := time.Parse(time.RFC3339Nano, q.Get("from"))
	to, _ := time.Parse(time.RFC3339Nano, q.Get("to"))
	win := fetchWindow{from: from, to: to, size: defaultFetchSize, maxStart: math.MaxInt}

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

func (h *handler) fetchLogsBetweenDates(w http.ResponseWriter, r *http.Request) {
	plainText := strings.Contains(r.Header.Get("Accept"), "text/plain")
	if plainText {
		w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
	} else {
		w.Header().Set("Content-Type", "application/x-jsonl; charset=UTF-8")
	}

	stdTypes := parseStdTypes(r)
	if stdTypes == 0 {
		http.Error(w, "stdout or stderr is required", http.StatusBadRequest)
		return
	}

	containerService, err := h.hostService.FindContainer(hostKey(r), chi.URLParam(r, "id"), h.resolveLabels(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	filter, err := parseLogFilter(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	q := r.URL.Query()
	win, err := parseFetchWindow(q)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if q.Has("everything") {
		events, err := containerService.LogsBetweenDates(r.Context(), time.Time{}, time.Now(), stdTypes)
		if err != nil {
			log.Error().Err(err).Msg("error fetching logs")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writer, done := compressedWriter(w, r)
		defer done()
		writeAllLogs(writer, events, filter, q.Has("jsonOnly"), plainText)
		return
	}

	events, err := fetchWindowedLogs(r.Context(), containerService, win, filter, stdTypes)
	if err != nil {
		log.Error().Err(err).Msg("error fetching logs")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Debug().Int("buffer_size", len(events)).Msg("sending logs to client")

	writer, done := compressedWriter(w, r)
	defer done()
	encoder := json.NewEncoder(writer)
	for _, event := range events {
		if err := encoder.Encode(event); err != nil {
			log.Error().Err(err).Msg("error encoding log event")
			return
		}
	}
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

// writeAllLogs streams a container's whole history for a download. Unlike the
// windowed fetch, an empty level set here means every level.
func writeAllLogs(writer io.Writer, events <-chan *container.LogEvent, filter logFilter, onlyComplex, plainText bool) {
	encoder := json.NewEncoder(writer)
	for event := range events {
		// Grouped events carry a []string message, so filtering on "not a
		// string" still lets arrays through and breaks struct inference for
		// consumers like the SQL analytics view.
		if onlyComplex && event.Type != container.LogTypeComplex {
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

		events, err := containerService.LogsBetweenDates(ctx, from, win.to, stdTypes)
		if err != nil {
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

		if from.Before(containerService.Container.Created) || win.minimum == 0 {
			break
		}

		from = from.Add(-delta)
		delta = delta * 2
	}

	return buffer.Data(), nil
}

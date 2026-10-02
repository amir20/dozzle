package web

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/amir20/dozzle/internal/analytics"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// histogramWidths are the bucket widths a histogram may use. Fixing them means
// buckets fall on round clock times and the same width comes back for windows of
// similar size, which is what lets the counter reuse what it cached.
var histogramWidths = []time.Duration{
	time.Second, 2 * time.Second, 5 * time.Second, 10 * time.Second, 15 * time.Second, 30 * time.Second,
	time.Minute, 2 * time.Minute, 5 * time.Minute, 10 * time.Minute, 15 * time.Minute, 30 * time.Minute,
	time.Hour, 2 * time.Hour, 3 * time.Hour, 6 * time.Hour, 12 * time.Hour, 24 * time.Hour,
}

const (
	defaultHistogramBuckets = 60
	minHistogramBuckets     = 10
	maxHistogramBuckets     = 240
)

// histogramWidth is the narrowest width that splits span into at most buckets.
func histogramWidth(span time.Duration, buckets int) time.Duration {
	for _, w := range histogramWidths {
		if span/w <= time.Duration(buckets) {
			return w
		}
	}
	return histogramWidths[len(histogramWidths)-1]
}

type histogramResponse struct {
	Start  time.Time `json:"start"`
	Width  int64     `json:"width"`
	Total  []uint32  `json:"total"`
	Errors []uint32  `json:"errors"`
	// Present when the count stopped short of start: buckets that end before it
	// are unknown rather than empty.
	ScannedFrom *time.Time `json:"scannedFrom,omitempty"`
	// Only for an empty window: the nearest lines on either side of it.
	Before *time.Time `json:"before,omitempty"`
	After  *time.Time `json:"after,omitempty"`
}

func (h *handler) fetchLogHistogram(w http.ResponseWriter, r *http.Request) {
	analytics.Count("logs.histogram")
	q := r.URL.Query()
	from, err := time.Parse(time.RFC3339Nano, q.Get("from"))
	if err != nil {
		http.Error(w, "from must be an RFC 3339 time", http.StatusBadRequest)
		return
	}
	to, err := time.Parse(time.RFC3339Nano, q.Get("to"))
	if err != nil || !to.After(from) {
		http.Error(w, "to must be an RFC 3339 time after from", http.StatusBadRequest)
		return
	}
	buckets := defaultHistogramBuckets
	if q.Has("buckets") {
		buckets, err = strconv.Atoi(q.Get("buckets"))
		if err != nil {
			http.Error(w, "buckets must be a number", http.StatusBadRequest)
			return
		}
		buckets = min(max(buckets, minHistogramBuckets), maxHistogramBuckets)
	}

	width := histogramWidth(to.Sub(from), buckets)
	from = from.Truncate(width)
	if aligned := to.Truncate(width); aligned.Before(to) {
		to = aligned.Add(width)
	}

	containerService, err := h.hostService.FindContainer(hostKey(r), chi.URLParam(r, "id"), h.resolveLabels(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	hist, err := containerService.LogHistogram(r.Context(), from, to, width)
	if err != nil {
		if status.Code(err) == codes.Unimplemented {
			// An agent older than this server: the UI leaves the ribbon out.
			http.Error(w, "the agent for this host does not support log histograms", http.StatusNotImplemented)
			return
		}
		if r.Context().Err() == nil {
			log.Error().Err(err).Msg("error counting logs")
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := histogramResponse{Start: hist.Start, Width: int64(hist.Width / time.Second), Total: hist.Total, Errors: hist.Errors}
	if !hist.ScannedFrom.IsZero() {
		resp.ScannedFrom = &hist.ScannedFrom
	}
	if !hist.Before.IsZero() {
		resp.Before = &hist.Before
	}
	if !hist.After.IsZero() {
		resp.After = &hist.After
	}
	w.Header().Set("Content-Type", "application/json")
	// Buckets that ended long ago never change, but the newest ones do, and the
	// counter's own cache already makes a repeat cheap.
	w.Header().Set("Cache-Control", "no-store")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Error().Err(err).Msg("error encoding log histogram")
	}
}

package web

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/amir20/dozzle/internal/cloud"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// cloudContainerMetrics reads back what Cloud kept for one container.
//
// The container is resolved through the caller's own scope first, so a filtered
// user cannot read the history of a container they cannot see. Cloud has never
// heard of a Dozzle user and would happily answer for any container on the
// instance.
//
//	200 — points returned (may be empty)
//	404 — no such container, or not one this caller may see
//	503 — cloud not configured
//	504 — cloud round-trip exceeded the timeout
//	502 — any other cloud-side error
func (h *handler) cloudContainerMetrics(w http.ResponseWriter, r *http.Request) {
	if h.config.Cloud.GetContainerMetrics == nil {
		writeError(w, http.StatusServiceUnavailable, "cloud not configured")
		return
	}

	// Resolved through the caller's own scope rather than passed through: cloud
	// has never heard of a Dozzle user and would answer for any container on the
	// instance. A container this caller cannot see reads as one that does not
	// exist. The lookup is also where the name comes from, which is what cloud
	// keys these samples by.
	service, err := h.hostService.FindContainer(hostKey(r), chi.URLParam(r, "id"), h.resolveLabels(r))
	if err != nil {
		writeError(w, http.StatusNotFound, "container not found")
		return
	}

	q := r.URL.Query()

	// A window rather than a start: "the last hour" is the question, and the
	// panel is the only caller.
	window := time.Hour
	if v := q.Get("window"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 && d <= 7*24*time.Hour {
			window = d
		}
	}

	buckets := int32(120)
	if v := q.Get("buckets"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 32); err == nil && n > 0 && n <= 1000 {
			buckets = int32(n)
		}
	}

	until := time.Now()
	ctx, cancel := context.WithTimeout(r.Context(), cloudAlertsTimeout)
	defer cancel()

	result, err := h.config.Cloud.GetContainerMetrics(ctx, service.Container.Name, service.Container.Host, until.Add(-window).UnixNano(), until.UnixNano(), buckets)
	if err != nil {
		if errors.Is(err, cloud.ErrNotConfigured) {
			writeError(w, http.StatusServiceUnavailable, "cloud not configured")
			return
		}
		if errors.Is(err, context.DeadlineExceeded) || status.Code(err) == codes.DeadlineExceeded {
			writeError(w, http.StatusGatewayTimeout, "cloud metrics timed out")
			return
		}
		// Cloud may not serve this RPC yet. That is an empty chart with a line
		// saying so, not an error banner.
		if status.Code(err) == codes.Unimplemented {
			writeMetrics(w, &cloud.MetricResult{Points: []cloud.MetricPoint{}})
			return
		}
		log.Warn().Err(err).Msg("cloud container metrics failed")
		writeError(w, http.StatusBadGateway, "cloud metrics failed")
		return
	}

	writeMetrics(w, result)
}

func writeMetrics(w http.ResponseWriter, result *cloud.MetricResult) {
	if result == nil {
		result = &cloud.MetricResult{Points: []cloud.MetricPoint{}}
	}
	if result.Points == nil {
		result.Points = []cloud.MetricPoint{}
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		log.Error().Err(err).Msg("failed to encode cloud metrics")
	}
}

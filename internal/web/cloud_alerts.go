package web

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/amir20/dozzle/internal/cloud"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// cloudAlertsTimeout caps the round-trip to Doligence Cloud. Shorter than the
// search timeout: this rides the scroll path, where the log lines themselves
// have already loaded and the alerts are a decoration on top. Better to render
// the logs without alerts than to hold the scroll waiting for them.
const cloudAlertsTimeout = 1500 * time.Millisecond

// maxAlertContainers mirrors Cloud's own cap. The viewer sends the containers
// currently on screen, which is a handful even in a merged multi-container
// view, so anything larger is a misbehaving client.
const maxAlertContainers = 100

// cloudAlerts proxies an alert-history lookup to Doligence Cloud over the
// existing authenticated gRPC connection, so the log viewer can merge alerts
// into the stream as the user scrolls back. Identity is derived server-side
// from the API key — this handler passes neither user nor instance ids.
//
// Deliberately NOT gated on streamLogs, unlike cloudSearchLogs: alerts live in
// Cloud's own database rather than the log store, so they exist for anyone who
// linked cloud and configured a subscription. Gating them behind the
// log-streaming opt-in would hide the feature from most users who'd benefit.
//
// Status mapping:
//
//	200 — hits returned (may be empty)
//	400 — malformed window or an oversized container list
//	503 — cloud not configured (no API key) or no GetAlerts func wired
//	504 — cloud round-trip exceeded the timeout
//	502 — any other cloud-side error
func (h *handler) cloudAlerts(w http.ResponseWriter, r *http.Request) {
	if h.config.Cloud.GetAlerts == nil {
		writeError(w, http.StatusServiceUnavailable, "cloud not configured")
		return
	}

	q := r.URL.Query()

	var containerIDs []string
	for id := range strings.SplitSeq(q.Get("containerIds"), ",") {
		if id = strings.TrimSpace(id); id != "" {
			containerIDs = append(containerIDs, id)
		}
	}
	if len(containerIDs) == 0 {
		// Nothing to look up. Answer with an empty result rather than an error
		// so the caller's merge path stays uniform.
		writeAlerts(w, &cloud.AlertResult{Hits: []cloud.AlertHit{}})
		return
	}
	// The client picks which containers to ask about, so run the list through
	// the caller's label scope before handing it to Cloud.
	if h.restrictedUser(r) {
		visible := h.visibleContainerIDs(r)
		allowed := make([]string, 0, len(containerIDs))
		for _, id := range containerIDs {
			if _, ok := visible[id]; ok {
				allowed = append(allowed, id)
			}
		}
		if len(allowed) == 0 {
			writeAlerts(w, &cloud.AlertResult{Hits: []cloud.AlertHit{}})
			return
		}
		containerIDs = allowed
	}

	if len(containerIDs) > maxAlertContainers {
		writeError(w, http.StatusBadRequest, "too many containerIds")
		return
	}

	from, err := strconv.ParseInt(q.Get("from"), 10, 64)
	if err != nil || from <= 0 {
		writeError(w, http.StatusBadRequest, "missing or invalid from")
		return
	}
	to, err := strconv.ParseInt(q.Get("to"), 10, 64)
	if err != nil || to <= from {
		writeError(w, http.StatusBadRequest, "missing or invalid to")
		return
	}

	// Cloud caps server-side too; mirroring it here keeps a misbehaving client
	// from tying up the scroll path with an oversized request.
	limit := int32(50)
	if v := q.Get("limit"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 32); err == nil && n > 0 {
			if n > 200 {
				n = 200
			}
			limit = int32(n)
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), cloudAlertsTimeout)
	defer cancel()

	result, err := h.config.Cloud.GetAlerts(ctx, containerIDs, q.Get("hostId"), from, to, limit, q.Get("followUps") == "1", q.Get("events") == "1")
	if err != nil {
		if errors.Is(err, cloud.ErrNotConfigured) {
			writeError(w, http.StatusServiceUnavailable, "cloud not configured")
			return
		}
		// A blown deadline surfaces two ways: context.DeadlineExceeded when our
		// own ctx fires first, or a gRPC status when the server-side deadline
		// trips. status.Code walks the %w wrap chain, so both map to a 504.
		if errors.Is(err, context.DeadlineExceeded) || status.Code(err) == codes.DeadlineExceeded {
			writeError(w, http.StatusGatewayTimeout, "cloud alerts timed out")
			return
		}
		log.Warn().Err(err).Msg("cloud alerts failed")
		writeError(w, http.StatusBadGateway, "cloud alerts failed")
		return
	}

	writeAlerts(w, result)
}

func writeAlerts(w http.ResponseWriter, result *cloud.AlertResult) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

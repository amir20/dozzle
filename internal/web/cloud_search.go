package web

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/amir20/dozzle/internal/cloud"
	"github.com/rs/zerolog/log"
)

// cloudSearchTimeout caps the round-trip to Doligence Cloud. Search is on
// the keystroke path; we'd rather show "no results" than block typing.
const cloudSearchTimeout = 500 * time.Millisecond

// cloudSearchLogs proxies a search query to Doligence Cloud over the existing
// authenticated gRPC connection. Identity is derived server-side from the
// API key — this handler passes neither user nor instance ids.
//
// Status mapping:
//   200 — hits returned (may be empty)
//   204 — streamLogs is disabled; nothing to search
//   503 — cloud not configured (no API key) or no SearchLogs func wired
//   504 — cloud round-trip exceeded the search timeout
//   502 — any other cloud-side error
func (h *handler) cloudSearchLogs(w http.ResponseWriter, r *http.Request) {
	if h.config.CloudSearchLogs == nil {
		writeError(w, http.StatusServiceUnavailable, "cloud not configured")
		return
	}

	cc := h.hostService.CloudConfig()
	if cc == nil || !cc.StreamLogsEnabled() {
		// Defense in depth — the UI already gates on streamLogs, but a stale
		// flag client-side mustn't trigger spurious cloud queries.
		w.WriteHeader(http.StatusNoContent)
		return
	}

	q := r.URL.Query().Get("q")
	limit := int32(20)
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = int32(n)
		}
	}
	hostID := r.URL.Query().Get("hostId")
	containerID := r.URL.Query().Get("containerId")

	ctx, cancel := context.WithTimeout(r.Context(), cloudSearchTimeout)
	defer cancel()

	result, err := h.config.CloudSearchLogs(ctx, q, limit, hostID, containerID)
	if err != nil {
		if errors.Is(err, cloud.ErrNotConfigured) {
			writeError(w, http.StatusServiceUnavailable, "cloud not configured")
			return
		}
		if errors.Is(err, context.DeadlineExceeded) {
			writeError(w, http.StatusGatewayTimeout, "cloud search timed out")
			return
		}
		log.Warn().Err(err).Msg("cloud search failed")
		writeError(w, http.StatusBadGateway, "cloud search failed")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(result)
}

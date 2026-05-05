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
	if h.config.Cloud.SearchLogs == nil {
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
	if q == "" {
		writeError(w, http.StatusBadRequest, "missing q")
		return
	}
	// Defense in depth: the UI input is short (debounced typing) but a
	// malicious client could POST any size. Reject anything past 512
	// chars rather than fan it out to Cloud's gRPC backend.
	if len(q) > 512 {
		writeError(w, http.StatusBadRequest, "q too long")
		return
	}
	// Cloud caps server-side at 50; mirror it here so a misbehaving client
	// can't tie up the keystroke path with an oversized request. ParseInt
	// with bitSize=32 guarantees the value fits in int32, so the cast is
	// provably safe (out-of-range parses return an error and fall through).
	limit := int32(20)
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 32); err == nil && n > 0 {
			if n > 50 {
				n = 50
			}
			limit = int32(n)
		}
	}
	hostID := r.URL.Query().Get("hostId")
	containerID := r.URL.Query().Get("containerId")
	// Pagination cursor — pass-through to Cloud. 0 (the default) means
	// "newest"; subsequent pages send back the prior response's nextBefore.
	var before int64
	if v := r.URL.Query().Get("before"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			before = n
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), cloudSearchTimeout)
	defer cancel()

	result, err := h.config.Cloud.SearchLogs(ctx, q, limit, hostID, containerID, before)
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
	_ = json.NewEncoder(w).Encode(result)
}

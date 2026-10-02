package web

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/amir20/dozzle/internal/cloud"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// cloudPatternsTimeout matches the alerts lookup: the chips decorate lines
// that have already rendered, so they give up quickly rather than hold anything.
const cloudPatternsTimeout = 1500 * time.Millisecond

// maxPatternLines mirrors Cloud's own cap on one request.
const maxPatternLines = 500

type cloudPatternsRequest struct {
	Lines []cloud.PatternLine `json:"lines"`
	From  int64               `json:"from"`
	To    int64               `json:"to"`
}

// cloudPatterns asks Cloud's error memory about the error and warn lines on
// screen, so the viewer can mark a line whose pattern is new for its
// container. POST because the line list is too long for a query string.
//
// Gated on streamLogs, unlike alerts: Cloud finds the lines in the logs it was
// streamed, so without the opt-in there is nothing to find.
//
// Status mapping:
//
//	200 — patterns returned (may be empty)
//	204 — streamLogs is off, or this Cloud doesn't serve the RPC yet
//	400 — malformed body or window, or too many lines
//	503 — cloud not configured (no API key) or no hook wired
//	504 — cloud round-trip exceeded the timeout
//	502 — any other cloud-side error
func (h *handler) cloudPatterns(w http.ResponseWriter, r *http.Request) {
	if h.config.Cloud.GetPatternContext == nil {
		writeError(w, http.StatusServiceUnavailable, "cloud not configured")
		return
	}
	cc := h.hostService.CloudConfig()
	if cc == nil || !cc.StreamLogsEnabled() {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	var req cloudPatternsRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if req.From <= 0 || req.To <= req.From {
		writeError(w, http.StatusBadRequest, "missing or invalid window")
		return
	}
	if len(req.Lines) > maxPatternLines {
		writeError(w, http.StatusBadRequest, "too many lines")
		return
	}
	lines := req.Lines
	// The client picks which containers to ask about, so run them through the
	// caller's label scope before anything reaches Cloud.
	if h.restrictedUser(r) {
		visible := h.visibleContainerIDs(r)
		allowed := make([]cloud.PatternLine, 0, len(lines))
		for _, l := range lines {
			if _, ok := visible[l.ContainerID]; ok {
				allowed = append(allowed, l)
			}
		}
		lines = allowed
	}
	if len(lines) == 0 {
		writePatterns(w, []cloud.PatternContext{})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), cloudPatternsTimeout)
	defer cancel()

	result, err := h.config.Cloud.GetPatternContext(ctx, lines, req.From, req.To)
	if err != nil {
		switch {
		case errors.Is(err, cloud.ErrNotConfigured):
			writeError(w, http.StatusServiceUnavailable, "cloud not configured")
		case status.Code(err) == codes.Unimplemented:
			// A Cloud that predates the RPC. Nothing to show, and nothing wrong.
			w.WriteHeader(http.StatusNoContent)
		case errors.Is(err, context.DeadlineExceeded) || status.Code(err) == codes.DeadlineExceeded:
			writeError(w, http.StatusGatewayTimeout, "cloud patterns timed out")
		default:
			log.Warn().Err(err).Msg("cloud patterns failed")
			writeError(w, http.StatusBadGateway, "cloud patterns failed")
		}
		return
	}
	writePatterns(w, result)
}

func writePatterns(w http.ResponseWriter, patterns []cloud.PatternContext) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"patterns": patterns})
}

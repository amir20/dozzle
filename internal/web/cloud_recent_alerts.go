package web

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/amir20/dozzle/internal/cloud"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// cloudRecentAlerts answers "what fired lately, anywhere on this instance" for
// the notifications page and the container dot.
//
// The scoping is the mirror image of cloudAlerts. There the client names the
// containers, so the request list is narrowed before it leaves. Here nobody
// names anything, so Cloud answers for the whole instance and the response is
// narrowed on the way back — Cloud scopes to the instance and has never heard
// of a Dozzle user.
//
//	200 — hits returned (may be empty)
//	503 — cloud not configured, or no GetRecentAlerts func wired
//	504 — cloud round-trip exceeded the timeout
//	502 — any other cloud-side error
func (h *handler) cloudRecentAlerts(w http.ResponseWriter, r *http.Request) {
	if h.config.Cloud.GetRecentAlerts == nil {
		writeError(w, http.StatusServiceUnavailable, "cloud not configured")
		return
	}

	q := r.URL.Query()

	var since int64
	if v := q.Get("since"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			since = n
		}
	}

	// Cloud caps server-side as well; mirroring it keeps a misbehaving client
	// from asking for the whole history.
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

	result, err := h.config.Cloud.GetRecentAlerts(ctx, since, limit, q.Get("subscriptionId"), q.Get("followUps") == "1")
	if err != nil {
		if errors.Is(err, cloud.ErrNotConfigured) {
			writeError(w, http.StatusServiceUnavailable, "cloud not configured")
			return
		}
		if errors.Is(err, context.DeadlineExceeded) || status.Code(err) == codes.DeadlineExceeded {
			writeError(w, http.StatusGatewayTimeout, "cloud alerts timed out")
			return
		}
		// Cloud may not serve this RPC yet. That is an empty page, not an
		// error banner: the section says what it would hold and moves on.
		if status.Code(err) == codes.Unimplemented {
			writeAlerts(w, &cloud.AlertResult{Hits: []cloud.AlertHit{}})
			return
		}
		log.Warn().Err(err).Msg("cloud recent alerts failed")
		writeError(w, http.StatusBadGateway, "cloud alerts failed")
		return
	}

	writeAlerts(w, scopeAlertsToViewer(h, r, result))
}

// scopeAlertsToViewer drops anything about a container the caller may not see.
// A filtered user must not learn that a container exists from an alert about it.
func scopeAlertsToViewer(h *handler, r *http.Request, result *cloud.AlertResult) *cloud.AlertResult {
	if result == nil {
		return &cloud.AlertResult{Hits: []cloud.AlertHit{}}
	}
	if !h.restrictedUser(r) {
		return result
	}

	visible := h.visibleContainerIDs(r)
	hits := make([]cloud.AlertHit, 0, len(result.Hits))
	for _, hit := range result.Hits {
		if _, ok := visible[hit.ContainerID]; ok {
			hits = append(hits, hit)
		}
	}
	events := make([]cloud.EventHit, 0, len(result.Events))
	for _, e := range result.Events {
		if _, ok := visible[e.ContainerID]; ok {
			events = append(events, e)
		}
	}

	return &cloud.AlertResult{Hits: hits, Events: events, Truncated: result.Truncated}
}

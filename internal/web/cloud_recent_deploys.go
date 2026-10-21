package web

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/amir20/dozzle/internal/cloud"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// cloudRecentDeploys answers "what did Cloud record being updated lately" for
// the megaphone, which merges it with the updates this process still holds.
// Same window and cap as the in-memory feed, so the two lists read as one.
//
// Like cloudRecentAlerts, Cloud answers for the whole instance and the result
// is narrowed on the way back. A filtered user sees an update only while the
// container it created is visible: a gone container's labels are unknown, the
// same rule the events stream applies to Dozzle's own records.
//
//	200 — deploys returned (may be empty, and is when Cloud does not serve the RPC)
//	503 — cloud not configured, or no GetRecentDeploys func wired
//	504 — cloud round-trip exceeded the timeout
//	502 — any other cloud-side error
func (h *handler) cloudRecentDeploys(w http.ResponseWriter, r *http.Request) {
	if h.config.Cloud.GetRecentDeploys == nil {
		writeError(w, http.StatusServiceUnavailable, "cloud not configured")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), cloudAlertsTimeout)
	defer cancel()

	since := time.Now().Add(-updateFeedWindow).UnixNano()
	deploys, err := h.config.Cloud.GetRecentDeploys(ctx, since, updateFeedLimit)
	if err != nil {
		switch {
		case errors.Is(err, cloud.ErrNotConfigured):
			writeError(w, http.StatusServiceUnavailable, "cloud not configured")
		case errors.Is(err, context.DeadlineExceeded) || status.Code(err) == codes.DeadlineExceeded:
			writeError(w, http.StatusGatewayTimeout, "cloud deploys timed out")
		case status.Code(err) == codes.Unimplemented:
			// An older Cloud: the megaphone shows what Dozzle remembers, as before.
			writeJSON(w, http.StatusOK, []cloud.RecentDeploy{})
		default:
			log.Warn().Err(err).Msg("cloud recent deploys failed")
			writeError(w, http.StatusBadGateway, "cloud deploys failed")
		}
		return
	}

	writeJSON(w, http.StatusOK, scopeDeploysToViewer(h, r, deploys))
}

// scopeDeploysToViewer drops any update to a container the caller may not see.
func scopeDeploysToViewer(h *handler, r *http.Request, deploys []cloud.RecentDeploy) []cloud.RecentDeploy {
	if deploys == nil {
		return []cloud.RecentDeploy{}
	}
	if !h.restrictedUser(r) {
		return deploys
	}
	visible := h.visibleContainerIDs(r)
	out := make([]cloud.RecentDeploy, 0, len(deploys))
	for _, d := range deploys {
		if _, ok := visible[d.NewID]; ok {
			out = append(out, d)
		}
	}
	return out
}

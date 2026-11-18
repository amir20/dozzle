package web

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/amir20/dozzle/internal/cloud"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// findingsFilters maps the three views Dozzle offers to Cloud's list query.
// "new" is open and ongoing, not ignored: Cloud's "needs attention".
var findingsFilters = map[string]struct {
	status  string
	ignored bool
}{
	"new":     {status: "open"},
	"all":     {status: ""},
	"ignored": {status: "", ignored: true},
}

// cloudFindings answers what Cloud's scheduled review found about the
// containers on screen.
//
// A finding records container names only, so each requested host/id is
// resolved through the caller's own scope first and only the names of the ones
// they may see go to Cloud. One they cannot see reads as one that does not
// exist, and is dropped without a word. Only the first maxAlertContainers are
// looked up; past that the answer is marked truncated.
//
// A filtered user gets no findings at all. A finding is keyed by container
// names and can describe several containers at once, with no host, so there is
// no way to prove every container it talks about is one they may see: a
// hidden namesake on another host, one on a host that is down, or a removed
// one all read the same. Failing closed is the only answer that holds.
//
// Cloud is still asked when no name is sent (nothing resolved, or a filtered
// user), because the answer carries the instance's review schedule, which is
// about no container. Cloud returns no findings for an empty list.
//
//	200 — findings returned (may be empty)
//	400 — bad filter
//	503 — cloud not configured
//	504 — cloud round-trip exceeded the timeout
//	502 — any other cloud-side error
func (h *handler) cloudFindings(w http.ResponseWriter, r *http.Request) {
	if h.config.Cloud.GetFindings == nil {
		writeError(w, http.StatusServiceUnavailable, "cloud not configured")
		return
	}

	// Same checks the ignore route enforces, decided here so the UI never has
	// to guess.
	canIgnore := h.hasCloudRole(r) && !h.restrictedUser(r)
	truncated := false
	respond := func(result *cloud.FindingsResult) {
		result.CanIgnore = canIgnore
		result.Truncated = result.Truncated || truncated
		writeFindings(w, result)
	}

	q := r.URL.Query()

	filterName := q.Get("filter")
	if filterName == "" {
		filterName = "new"
	}
	filter, ok := findingsFilters[filterName]
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid filter")
		return
	}

	// A filtered user never sends names: they only get the schedule.
	restricted := h.restrictedUser(r)

	requested := q["container"]
	if restricted {
		requested = nil
	}
	if len(requested) > maxAlertContainers {
		requested = requested[:maxAlertContainers]
		truncated = true
	}

	limit := int32(20)
	if v := q.Get("limit"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 32); err == nil && n > 0 {
			limit = int32(min(n, 50))
		}
	}

	// Several containers can share a name (one per host), so a name maps back
	// to every container on screen that carries it.
	refs := make([]containerRef, 0, len(requested))
	seen := make(map[containerRef]struct{}, len(requested))
	for _, raw := range requested {
		host, id, found := strings.Cut(raw, "/")
		if !found || host == "" || id == "" {
			continue
		}
		ref := containerRef{host: host, id: id}
		if _, dup := seen[ref]; dup {
			continue
		}
		seen[ref] = struct{}{}
		refs = append(refs, ref)
	}

	// Several at a time, so one slow agent host does not hold up the rest. A
	// container that is gone, outside the caller's labels or on a failing host is
	// left out: the answer is about the ones that resolved.
	services, _ := h.resolveContainers(refs, h.resolveLabels(r), nil)
	if r.Context().Err() != nil {
		return
	}

	byName := make(map[string][]cloud.FindingContainer)
	var names []string
	for _, service := range services {
		c := service.Container
		if _, known := byName[c.Name]; !known {
			names = append(names, c.Name)
		}
		byName[c.Name] = append(byName[c.Name], cloud.FindingContainer{Host: c.Host, ID: c.ID, Name: c.Name})
	}

	ctx, cancel := context.WithTimeout(r.Context(), cloudAlertsTimeout)
	defer cancel()

	result, err := h.config.Cloud.GetFindings(ctx, names, filter.status, filter.ignored, limit)
	if err != nil {
		if errors.Is(err, cloud.ErrNotConfigured) {
			writeError(w, http.StatusServiceUnavailable, "cloud not configured")
			return
		}
		if errors.Is(err, context.DeadlineExceeded) || status.Code(err) == codes.DeadlineExceeded {
			writeError(w, http.StatusGatewayTimeout, "cloud findings timed out")
			return
		}
		// An older Cloud has no findings RPC. That is an empty list, not an
		// error banner.
		if status.Code(err) == codes.Unimplemented {
			respond(&cloud.FindingsResult{})
			return
		}
		log.Warn().Err(err).Msg("cloud findings failed")
		writeError(w, http.StatusBadGateway, "cloud findings failed")
		return
	}

	// Map each finding's names back to the containers asked about. A finding
	// that maps to none was not about anything this caller asked for, so it is
	// dropped rather than shown with no container.
	findings := make([]cloud.Finding, 0, len(result.Findings))
	for _, f := range result.Findings {
		f.Containers = []cloud.FindingContainer{}
		mapped := make(map[string]struct{})
		for _, name := range f.ContainerNames {
			if _, done := mapped[name]; done {
				continue
			}
			mapped[name] = struct{}{}
			f.Containers = append(f.Containers, byName[name]...)
		}
		if len(f.Containers) == 0 {
			continue
		}
		findings = append(findings, f)
	}

	respond(&cloud.FindingsResult{
		Findings:   findings,
		FixLocked:  result.FixLocked,
		Truncated:  result.Truncated,
		LastScanAt: result.LastScanAt,
		NextScanAt: result.NextScanAt,
	})
}

// ignoreCloudFinding ignores one finding (POST), or takes that back (DELETE).
//
// Ignoring is account-wide: it changes what Cloud reports and emails for every
// instance on the account. The route requires the cloud role, and a filtered
// user is refused here too, since they could otherwise silence a finding about
// containers they cannot see.
//
//	200 — the updated finding
//	400 — Cloud rejected the id
//	403 — the caller is confined by a filter
//	404 — no such finding on this instance
//	501 — Cloud does not serve this yet
//	503 — cloud not configured
//	504 — cloud round-trip exceeded the timeout
//	502 — any other cloud-side error
func (h *handler) ignoreCloudFinding(w http.ResponseWriter, r *http.Request) {
	if h.restrictedUser(r) {
		writeError(w, http.StatusForbidden, "filtered users cannot ignore findings")
		return
	}
	if h.config.Cloud.IgnoreFinding == nil {
		writeError(w, http.StatusServiceUnavailable, "cloud not configured")
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing finding id")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), cloudAlertsTimeout)
	defer cancel()

	finding, err := h.config.Cloud.IgnoreFinding(ctx, id, r.Method == http.MethodDelete)
	if err != nil {
		switch {
		case errors.Is(err, cloud.ErrNotConfigured):
			writeError(w, http.StatusServiceUnavailable, "cloud not configured")
		case errors.Is(err, context.DeadlineExceeded) || status.Code(err) == codes.DeadlineExceeded:
			writeError(w, http.StatusGatewayTimeout, "cloud timed out")
		case status.Code(err) == codes.NotFound:
			writeError(w, http.StatusNotFound, "finding not found")
		case status.Code(err) == codes.InvalidArgument:
			writeError(w, http.StatusBadRequest, "invalid finding id")
		case status.Code(err) == codes.Unimplemented:
			writeError(w, http.StatusNotImplemented, "cloud does not support ignoring findings yet")
		default:
			log.Warn().Err(err).Msg("cloud ignore finding failed")
			writeError(w, http.StatusBadGateway, "cloud ignore finding failed")
		}
		return
	}

	if finding.Containers == nil {
		finding.Containers = []cloud.FindingContainer{}
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(finding); err != nil {
		log.Error().Err(err).Msg("failed to encode cloud finding")
	}
}

func writeFindings(w http.ResponseWriter, result *cloud.FindingsResult) {
	if result.Findings == nil {
		result.Findings = []cloud.Finding{}
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		log.Error().Err(err).Msg("failed to encode cloud findings")
	}
}

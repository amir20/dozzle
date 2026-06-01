package cloud

import (
	"fmt"
	"strings"

	"github.com/amir20/dozzle/internal/container"
)

// resolveContainerRef turns an LLM-supplied container reference (a name OR an
// id) plus an optional host reference (a name OR an id) into the concrete
// (hostID, containerID) pair that HostService.FindContainer expects.
//
// This is the STRICT (write) resolver used by the destructive tools
// (start/stop/restart/remove/update). It NEVER picks between multiple live
// containers — acting on the wrong one is irreversible — so a name that matches
// more than one container resolves only when exactly one candidate is running
// (the others being stopped task corpses); otherwise it fails with the
// candidate listing. Read-only tools use resolveContainerRefRead, which is
// allowed to pick the most-relevant container in one shot (see that function).
//
// LLMs almost always have only the container name (it is what logs, events and
// listings surface), so every container-scoped tool funnels through here rather
// than passing the raw reference straight to FindContainer — which only matches
// exact ids on a known host.
//
// Matching is tiered, id-first so existing id-based callers behave exactly as
// before (this capability is purely additive — names are the new fallback):
//  1. exact id match (full id or short/prefix id) — the legacy path
//  2. exact name match (case-insensitive)
//  3. unique substring of the name
//
// The first tier that yields any candidates wins. An id match is always unique,
// so it is never treated as ambiguous. When a name tier yields more than one
// container we try to break the tie by running state: if exactly one candidate
// is running it is the unambiguous live referent (the others are stopped task
// corpses left by Swarm redeploys / restart churn) and we resolve to it. Only
// when the tie is real — zero running, or several live containers — does the
// call fail with an error listing every candidate (name + id + state + host) so
// the caller can disambiguate. We NEVER pick between multiple *live* containers,
// which is what keeps the write tools (stop/restart/remove/update) safe by name.
//
// hostRef is optional: when empty the container must resolve unambiguously
// across all hosts; when supplied it scopes the search to that host (matched by
// id or name).
func resolveContainerRef(containerRef, hostRef string, deps ToolDeps) (hostID, containerID string, err error) {
	tier, trimmedHostRef, scopedHostID, hostNames, err := matchContainerTier(containerRef, hostRef, deps)
	if err != nil {
		return "", "", err
	}

	switch len(tier) {
	case 0:
		// No tier matched. When an explicit host was given (and resolved to a
		// real host id) but the listing produced no match — e.g. a host returned
		// a partial error and was omitted from ListAllContainers — pass the
		// reference straight through to FindContainer's direct lookup, exactly as
		// before this resolver existed. This guarantees id-based callers never
		// regress.
		if scopedHostID != "" {
			return scopedHostID, strings.TrimSpace(containerRef), nil
		}
		return "", "", fmt.Errorf("no container matching %q found across all connected hosts; call find_containers to list available containers", strings.TrimSpace(containerRef))
	case 1:
		return tier[0].Host, tier[0].ID, nil
	default:
		// Multiple matches. The usual benign cause is Docker Swarm (or plain
		// restart churn) leaving stopped task containers behind across redeploys
		// — the short name "svc.1" substring-matches every historical
		// "svc.1.<taskid>" corpse alongside the one live task. When exactly one
		// candidate is running it is the unambiguous live referent, so resolve to
		// it rather than make the caller hunt for an id. We still refuse to
		// choose between multiple *live* containers — that is the real ambiguity
		// the write tools must never guess at.
		if live := runningContainers(tier); len(live) == 1 {
			return live[0].Host, live[0].ID, nil
		}
		return "", "", ambiguousError(strings.TrimSpace(containerRef), trimmedHostRef, tier, hostNames)
	}
}

// resolveContainerRefRead is the PERMISSIVE (read-only) resolver used by the
// non-destructive tools (inspect_container, fetch_container_logs, stream_logs).
//
// It matches identically to resolveContainerRef, but when a name is ambiguous
// it does NOT error — it resolves to the single most-relevant candidate in one
// shot, eliminating the find-then-act round-trip that otherwise costs a whole
// extra LLM call on the investigation path. The pick is safe precisely because
// the tool is read-only: reading the wrong replica's logs is cheap and
// recoverable, and same-service replicas are interchangeable for diagnosis,
// whereas restarting the wrong one is not. The strict resolver above keeps the
// write tools honest.
//
// Selection (bestCandidate): running containers beat stopped ones, then newest
// StartedAt, then most-recently-finished/created. For the crash-loop case —
// every "svc.1.*" task exited — this lands on the corpse that died most
// recently, which is what the investigation is usually about.
//
// When it picks among several candidates it returns a one-line note naming the
// chosen container and its siblings. The caller surfaces that note in the tool
// RESULT (not via a round-trip), so the model gets the disambiguation context
// for free in the same turn and can re-call with an explicit id if it genuinely
// wants a different replica — but it is never forced to. note is empty when the
// match was unique (nothing to disclose).
func resolveContainerRefRead(containerRef, hostRef string, deps ToolDeps) (hostID, containerID, note string, err error) {
	tier, _, scopedHostID, hostNames, err := matchContainerTier(containerRef, hostRef, deps)
	if err != nil {
		return "", "", "", err
	}

	switch len(tier) {
	case 0:
		if scopedHostID != "" {
			return scopedHostID, strings.TrimSpace(containerRef), "", nil
		}
		return "", "", "", fmt.Errorf("no container matching %q found across all connected hosts; call find_containers to list available containers", strings.TrimSpace(containerRef))
	case 1:
		return tier[0].Host, tier[0].ID, "", nil
	default:
		// Exactly one running candidate is unambiguous — the corpses are never a
		// valid target — so resolve cleanly with no note, identical to the strict
		// resolver. A note is only warranted when the read path actually exercises
		// its relaxation: zero running (pick the newest corpse) or several running
		// replicas (pick the newest live one).
		if live := runningContainers(tier); len(live) == 1 {
			return live[0].Host, live[0].ID, "", nil
		}
		best := bestCandidate(tier)
		note = resolutionNote(strings.TrimSpace(containerRef), best, tier, hostNames)
		return best.Host, best.ID, note, nil
	}
}

// matchContainerTier runs the shared scoping + tiered matching used by both
// resolvers and returns the winning tier (the first non-empty of exact-id,
// exact-name, substring). It also returns the trimmed hostRef, the resolved
// scoped host id (empty when no host was supplied), and the host-name map for
// building human-readable notes/errors. An empty returned tier means no match.
func matchContainerTier(containerRef, hostRef string, deps ToolDeps) (tier []container.Container, trimmedHostRef, scopedHostID string, hostNames map[string]string, err error) {
	containerRef = strings.TrimSpace(containerRef)
	if containerRef == "" {
		return nil, "", "", nil, fmt.Errorf("container_id is required")
	}

	containers, errs := deps.HostService.ListAllContainers(deps.Labels)
	logHostErrors(errs)
	hostNames = buildHostNameMap(deps.HostService)

	// Scope to a host if one was supplied. The host reference may be an id or a
	// name; an unknown host is an explicit error rather than a silent no-match.
	trimmedHostRef = strings.TrimSpace(hostRef)
	if trimmedHostRef != "" {
		scopedHostID, err = resolveHostRef(trimmedHostRef, deps)
		if err != nil {
			return nil, trimmedHostRef, "", hostNames, err
		}
		filtered := containers[:0:0]
		for _, c := range containers {
			if c.Host == scopedHostID {
				filtered = append(filtered, c)
			}
		}
		containers = filtered
	}

	// Tiered matching. Each tier collects candidates; the first non-empty tier
	// decides the outcome. Id is checked first so a value that is a real
	// container id resolves directly — identical to the legacy behavior — and is
	// never confused with a name.
	var exactID, exactName, substring []container.Container
	for _, c := range containers {
		switch {
		case matchesID(c.ID, containerRef):
			exactID = append(exactID, c)
		case strings.EqualFold(c.Name, containerRef):
			exactName = append(exactName, c)
		case containsIgnoreCase(c.Name, containerRef):
			substring = append(substring, c)
		}
	}

	for _, t := range [][]container.Container{exactID, exactName, substring} {
		if len(t) > 0 {
			return t, trimmedHostRef, scopedHostID, hostNames, nil
		}
	}
	return nil, trimmedHostRef, scopedHostID, hostNames, nil
}

// matchesID reports whether ref identifies the container id — either the full id
// or a short/prefix form (Docker ids are commonly referenced by their first 12
// characters). Comparison is case-insensitive to tolerate sloppy input.
func matchesID(id, ref string) bool {
	if id == "" {
		return false
	}
	lid, lref := strings.ToLower(id), strings.ToLower(ref)
	if lid == lref {
		return true
	}
	// Treat ref as a prefix only when it is reasonably id-shaped to avoid a
	// short generic string accidentally prefix-matching an id.
	if len(lref) >= 12 && strings.HasPrefix(lid, lref) {
		return true
	}
	return false
}

// resolveHostRef resolves a host reference (id or name) to its host id.
func resolveHostRef(hostRef string, deps ToolDeps) (string, error) {
	hosts := deps.HostService.Hosts()
	var byName []container.Host
	for _, h := range hosts {
		if h.ID == hostRef {
			return h.ID, nil
		}
		if strings.EqualFold(h.Name, hostRef) {
			byName = append(byName, h)
		}
	}
	switch len(byName) {
	case 0:
		return "", fmt.Errorf("no host matching %q found; call list_hosts to see available hosts", hostRef)
	case 1:
		return byName[0].ID, nil
	default:
		names := make([]string, len(byName))
		for i, h := range byName {
			names[i] = fmt.Sprintf("%s (id %s)", h.Name, h.ID)
		}
		return "", fmt.Errorf("host name %q is ambiguous; matches: %s. Pass the host id instead", hostRef, strings.Join(names, "; "))
	}
}

// ambiguousError builds an actionable error listing every candidate so the
// caller can re-issue the call unambiguously. The hint is tailored to the
// candidate set because the LLM reads it to choose its next action: when the
// candidates span multiple hosts, host_id disambiguates; when they all sit on
// one host, host_id is useless and only the exact id or full name will do.
func ambiguousError(containerRef, hostRef string, candidates []container.Container, hostNames map[string]string) error {
	parts := make([]string, len(candidates))
	sameHost := true
	for i, c := range candidates {
		parts[i] = fmt.Sprintf("%s (id %s, %s, on host %s)", c.Name, shortID(c.ID), describeState(c), resolveHostName(c.Host, hostNames))
		if c.Host != candidates[0].Host {
			sameHost = false
		}
	}

	var hint string
	switch {
	case hostRef != "" || sameHost:
		// A host was already supplied, or every candidate is on the same host —
		// scoping by host_id cannot narrow it further.
		hint = "pass the exact container id or the full container name to disambiguate"
	default:
		hint = "pass host_id to scope to one host, or pass the exact container id"
	}
	return fmt.Errorf("%q matches multiple containers: %s. To act on the right one, %s", containerRef, strings.Join(parts, "; "), hint)
}

// runningContainers returns only the candidates in the running state. It breaks
// a multi-match tie down to the single live container when every other candidate
// is a stopped corpse (the typical Swarm-redeploy / restart-churn case), which
// is the one situation where picking among matches is unambiguous and safe.
func runningContainers(cs []container.Container) []container.Container {
	live := make([]container.Container, 0, len(cs))
	for _, c := range cs {
		if strings.EqualFold(c.State, "running") {
			live = append(live, c)
		}
	}
	return live
}

// bestCandidate picks the single most-relevant container from an ambiguous name
// match for the read-only path. Ordering: running beats stopped, then newest
// StartedAt, then most-recently-finished, then most-recently-created. For the
// crash-loop case (every "svc.1.*" task exited) this lands on the corpse that
// died most recently — the one the investigation is almost always about. The
// caller guarantees len(cs) > 0.
func bestCandidate(cs []container.Container) container.Container {
	best := cs[0]
	for _, c := range cs[1:] {
		if moreRelevant(c, best) {
			best = c
		}
	}
	return best
}

// moreRelevant reports whether a should be preferred over b as the read-path
// referent. Running always wins; among the same running-ness, the more-recently
// active container wins (StartedAt, then FinishedAt, then Created).
func moreRelevant(a, b container.Container) bool {
	aRunning, bRunning := isRunning(a), isRunning(b)
	if aRunning != bRunning {
		return aRunning
	}
	if !a.StartedAt.Equal(b.StartedAt) {
		return a.StartedAt.After(b.StartedAt)
	}
	if !a.FinishedAt.Equal(b.FinishedAt) {
		return a.FinishedAt.After(b.FinishedAt)
	}
	return a.Created.After(b.Created)
}

func isRunning(c container.Container) bool {
	return strings.EqualFold(c.State, "running")
}

// resolutionNote renders the one-line transparency note the read-only tools
// surface in their result when a name resolved to one of several candidates. It
// names the chosen container (state-annotated), summarizes how many siblings of
// each running-ness exist, and lists the sibling names — so the model can re-call
// with an explicit id if it wants a different replica, without ever being forced
// to. Returned as a parenthetical prefix the tools prepend to a human-visible
// field.
func resolutionNote(containerRef string, chosen container.Container, candidates []container.Container, hostNames map[string]string) string {
	running := len(runningContainers(candidates))

	// Annotate sibling names with their host only when the candidates span more
	// than one host — that is the case where the model needs the host to re-target
	// a sibling; on a single host it would just be noise.
	multiHost := false
	for _, c := range candidates {
		if c.Host != chosen.Host {
			multiHost = true
			break
		}
	}

	siblings := make([]string, 0, len(candidates)-1)
	for _, c := range candidates {
		if c.ID == chosen.ID {
			continue
		}
		if multiHost {
			siblings = append(siblings, fmt.Sprintf("%s on host %s", c.Name, resolveHostName(c.Host, hostNames)))
		} else {
			siblings = append(siblings, c.Name)
		}
	}

	var summary string
	switch {
	case running > 1 && isRunning(chosen):
		summary = fmt.Sprintf("newest of %d running replicas", running)
	case running == 0:
		summary = fmt.Sprintf("most recently active of %d stopped containers", len(candidates))
	default:
		summary = fmt.Sprintf("1 of %d matching containers", len(candidates))
	}

	return fmt.Sprintf("(resolved %q → %s [%s], %s; others: %s)",
		containerRef, chosen.Name, describeState(chosen), summary, strings.Join(siblings, ", "))
}

// describeState renders a candidate's state (with health when known) for the
// ambiguity listing, so the caller can tell a live replica from a stopped corpse
// and pick the right one. A blank state is reported as "unknown".
func describeState(c container.Container) string {
	state := c.State
	if state == "" {
		state = "unknown"
	}
	if c.Health != "" {
		return fmt.Sprintf("%s (%s)", state, c.Health)
	}
	return state
}

// shortID trims a full docker id to its conventional 12-character form for
// display in errors.
func shortID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

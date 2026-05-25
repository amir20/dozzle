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
// so it is never treated as ambiguous. If a name tier yields more than one
// container the call fails with an error listing every candidate (name + id +
// host) so the caller can disambiguate — we NEVER silently pick one. This is
// what makes the write tools (stop/restart/remove/update) safe to drive by name.
//
// hostRef is optional: when empty the container must resolve unambiguously
// across all hosts; when supplied it scopes the search to that host (matched by
// id or name).
func resolveContainerRef(containerRef, hostRef string, deps ToolDeps) (hostID, containerID string, err error) {
	containerRef = strings.TrimSpace(containerRef)
	if containerRef == "" {
		return "", "", fmt.Errorf("container_id is required")
	}

	containers, errs := deps.HostService.ListAllContainers(deps.Labels)
	logHostErrors(errs)
	hostNames := buildHostNameMap(deps.HostService)

	// Scope to a host if one was supplied. The host reference may be an id or a
	// name; an unknown host is an explicit error rather than a silent no-match.
	hostRef = strings.TrimSpace(hostRef)
	var scopedHostID string
	if hostRef != "" {
		scopedHostID, err = resolveHostRef(hostRef, deps)
		if err != nil {
			return "", "", err
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

	for _, tier := range [][]container.Container{exactID, exactName, substring} {
		switch len(tier) {
		case 0:
			continue
		case 1:
			return tier[0].Host, tier[0].ID, nil
		default:
			return "", "", ambiguousError(containerRef, hostRef, tier, hostNames)
		}
	}

	// Legacy fallback: when an explicit host was given (and resolved to a real
	// host id) but the listing produced no match — e.g. a host returned a
	// partial error and was omitted from ListAllContainers — pass the reference
	// straight through to FindContainer's direct lookup, exactly as before this
	// resolver existed. This guarantees id-based callers never regress.
	if scopedHostID != "" {
		return scopedHostID, containerRef, nil
	}

	return "", "", fmt.Errorf("no container matching %q found across all connected hosts; call find_containers to list available containers", containerRef)
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
		parts[i] = fmt.Sprintf("%s (id %s on host %s)", c.Name, shortID(c.ID), resolveHostName(c.Host, hostNames))
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

// shortID trims a full docker id to its conventional 12-character form for
// display in errors.
func shortID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

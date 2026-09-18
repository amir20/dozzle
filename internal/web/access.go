package web

import (
	"net/http"

	"github.com/amir20/dozzle/internal/auth"
	"github.com/amir20/dozzle/internal/container"
)

func (h *handler) resolveLabels(r *http.Request) container.ContainerLabels {
	labels := h.config.Labels
	if h.config.Authorization.Provider != NONE {
		user := auth.UserFromContext(r.Context())
		if user.ContainerLabels.Exists() {
			labels = user.ContainerLabels
		}
	}
	return labels
}

// restrictedUser reports whether the caller is confined by a per-user label
// filter. The global --filter applies to everyone, so it doesn't count.
func (h *handler) restrictedUser(r *http.Request) bool {
	if h.config.Authorization.Provider == NONE {
		return false
	}
	user := auth.UserFromContext(r.Context())
	return user != nil && user.ContainerLabels.Exists()
}

// visibleContainerIDs returns the set of container ids the caller may see.
// Used by handlers that receive container ids from the client (or get them
// back from Cloud) and have no other way to run them through the store's ACL.
func (h *handler) visibleContainerIDs(r *http.Request) map[string]struct{} {
	containers, _ := h.hostService.ListAllContainers(h.resolveLabels(r))
	ids := make(map[string]struct{}, len(containers))
	for _, c := range containers {
		ids[c.ID] = struct{}{}
	}
	return ids
}

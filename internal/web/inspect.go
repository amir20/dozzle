package web

import (
	"errors"
	"net/http"
	"slices"
	"strings"

	"github.com/amir20/dozzle/internal/auth"
	"github.com/amir20/dozzle/internal/container"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

// containerInspect is what the inspect drawer adds to the container the
// frontend already holds. Ports, mounts, labels and limits ride on the
// container itself, so only the facts that need an inspect are here.
type containerInspect struct {
	RestartPolicy string `json:"restartPolicy,omitempty"`
	RestartCount  int    `json:"restartCount"`
	OOMKilled     bool   `json:"oomKilled"`
	ExitCode      int    `json:"exitCode"`
	NetworkMode   string `json:"networkMode,omitempty"`
	// Networks are what the container is attached to, with its IP on each.
	Networks []container.ContainerNetwork `json:"networks"`
	Env      []inspectEnv                 `json:"env"`
	// EnvRevealed says whether Env carries values. Anyone who can open a shell
	// can already read them, so values follow the shell role and nothing else.
	EnvRevealed bool `json:"envRevealed"`
}

type inspectEnv struct {
	Key   string `json:"key"`
	Value string `json:"value,omitempty"`
}

func (h *handler) inspectContainer(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	containerService, err := h.hostService.FindContainer(hostKey(r), id, h.resolveLabels(r))
	if err != nil {
		log.Error().Err(err).Msg("error while trying to find container")
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	c := containerService.Container
	reveal := h.config.EnableShell && h.userRoles(r).Has(auth.Shell)

	env := make([]inspectEnv, 0, len(c.Env))
	for _, kv := range c.Env {
		key, value, _ := strings.Cut(kv, "=")
		entry := inspectEnv{Key: key}
		if reveal {
			entry.Value = value
		}
		env = append(env, entry)
	}

	networks := c.Networks
	if networks == nil {
		networks = []container.ContainerNetwork{}
	}

	writeJSON(w, http.StatusOK, containerInspect{
		RestartPolicy: c.RestartPolicy,
		RestartCount:  c.RestartCount,
		OOMKilled:     c.OOMKilled,
		ExitCode:      c.ExitCode,
		NetworkMode:   c.NetworkMode,
		Networks:      networks,
		Env:           env,
		EnvRevealed:   reveal,
	})
}

// inspectNetwork reads one network the container is attached to. Neighbours the
// user's label filter hides are left out, the same as everywhere else.
func (h *handler) inspectNetwork(w http.ResponseWriter, r *http.Request) {
	containerService, visible, ok := h.findForResource(w, r)
	if !ok {
		return
	}
	network, err := containerService.InspectNetwork(r.Context(), chi.URLParam(r, "network"))
	if !writeResourceError(w, err) {
		return
	}
	network.Containers = slices.DeleteFunc(network.Containers, func(e container.NetworkEndpoint) bool { return !visible[e.ID] })
	writeJSON(w, http.StatusOK, network)
}

// inspectVolume reads one volume the container mounts, and who else mounts it.
func (h *handler) inspectVolume(w http.ResponseWriter, r *http.Request) {
	containerService, visible, ok := h.findForResource(w, r)
	if !ok {
		return
	}
	volume, err := containerService.InspectVolume(r.Context(), chi.URLParam(r, "volume"))
	if !writeResourceError(w, err) {
		return
	}
	volume.Containers = slices.DeleteFunc(volume.Containers, func(u container.VolumeUser) bool { return !visible[u.ID] })
	writeJSON(w, http.StatusOK, volume)
}

// findForResource finds the container a network or volume is reached through,
// and the IDs of every container on its host the user may see.
func (h *handler) findForResource(w http.ResponseWriter, r *http.Request) (*container.ContainerService, map[string]bool, bool) {
	host := hostKey(r)
	labels := h.resolveLabels(r)
	containerService, err := h.hostService.FindContainer(host, chi.URLParam(r, "id"), labels)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return nil, nil, false
	}
	containers, err := h.hostService.ListContainersForHost(host, labels)
	if err != nil {
		log.Error().Err(err).Str("host", host).Msg("error while listing containers")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil, nil, false
	}
	visible := make(map[string]bool, len(containers))
	for _, c := range containers {
		visible[c.ID] = true
	}
	return containerService, visible, true
}

// writeResourceError answers a failed network or volume inspect and reports
// whether there was nothing to answer.
func writeResourceError(w http.ResponseWriter, err error) bool {
	switch {
	case err == nil:
		return true
	case errors.Is(err, container.ErrNotAttached):
		http.Error(w, err.Error(), http.StatusNotFound)
	case errors.Is(err, errors.ErrUnsupported):
		http.Error(w, err.Error(), http.StatusNotImplemented)
	default:
		log.Error().Err(err).Msg("error while inspecting a network or volume")
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	return false
}

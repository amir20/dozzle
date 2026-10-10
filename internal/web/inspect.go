package web

import (
	"net/http"
	"strings"

	"github.com/amir20/dozzle/internal/auth"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

// containerInspect is what the inspect drawer adds to the container the
// frontend already holds. Ports, mounts, labels and limits ride on the
// container itself, so only the facts that need an inspect are here.
type containerInspect struct {
	RestartPolicy string       `json:"restartPolicy,omitempty"`
	RestartCount  int          `json:"restartCount"`
	OOMKilled     bool         `json:"oomKilled"`
	ExitCode      int          `json:"exitCode"`
	NetworkMode   string       `json:"networkMode,omitempty"`
	Env           []inspectEnv `json:"env"`
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

	writeJSON(w, http.StatusOK, containerInspect{
		RestartPolicy: c.RestartPolicy,
		RestartCount:  c.RestartCount,
		OOMKilled:     c.OOMKilled,
		ExitCode:      c.ExitCode,
		NetworkMode:   c.NetworkMode,
		Env:           env,
		EnvRevealed:   reveal,
	})
}

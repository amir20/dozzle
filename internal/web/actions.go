package web

import (
	"context"
	"net/http"
	"time"

	"github.com/amir20/dozzle/internal/docker"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

func (h *handler) containerActions(w http.ResponseWriter, r *http.Request) {
	action := chi.URLParam(r, "action")
	id := chi.URLParam(r, "id")

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	containerService, err := h.multiHostService.FindContainer(ctx, hostKey(r), id)
	if err != nil {
		log.Error().Err(err).Msg("error while trying to find container")
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	parsedAction, err := docker.ParseContainerAction(action)
	if err != nil {
		log.Error().Err(err).Msg("error while trying to parse action")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := containerService.Action(ctx, parsedAction); err != nil {
		log.Error().Err(err).Msg("error while trying to perform container action")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Info().Str("action", action).Str("container", containerService.Container.Name).Msg("container action performed")
	http.Error(w, "", http.StatusNoContent)
}

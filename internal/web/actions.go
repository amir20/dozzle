package web

import (
	"context"
	"net/http"
	"time"

	"github.com/amir20/dozzle/internal/auth"
	"github.com/amir20/dozzle/internal/container"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

func (h *handler) containerActions(w http.ResponseWriter, r *http.Request) {
	action := chi.URLParam(r, "action")
	id := chi.URLParam(r, "id")

	usersFilter := h.config.Filter
	if h.config.Authorization.Provider != NONE {
		user := auth.UserFromContext(r.Context())
		if user.ContainerFilter.Exists() {
			usersFilter = user.ContainerFilter
		}
	}

	containerService, err := h.multiHostService.FindContainer(hostKey(r), id, usersFilter)
	if err != nil {
		log.Error().Err(err).Msg("error while trying to find container")
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	parsedAction, err := container.ParseContainerAction(action)
	if err != nil {
		log.Error().Err(err).Msg("error while trying to parse action")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := containerService.Action(ctx, parsedAction); err != nil {
		log.Error().Err(err).Msg("error while trying to perform container action")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Info().Str("action", action).Str("container", containerService.Container.Name).Msg("container action performed")
	http.Error(w, "", http.StatusNoContent)
}

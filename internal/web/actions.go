package web

import (
	"net/http"

	"github.com/amir20/dozzle/internal/auth"
	"github.com/amir20/dozzle/internal/container"
	container_support "github.com/amir20/dozzle/internal/support/container"
	support_web "github.com/amir20/dozzle/internal/support/web"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

func (h *handler) findContainerWithActions(w http.ResponseWriter, r *http.Request) (*container_support.ContainerService, bool) {
	id := chi.URLParam(r, "id")

	userLabels := h.config.Labels
	permit := true
	if h.config.Authorization.Provider != NONE {
		user := auth.UserFromContext(r.Context())
		if user.ContainerLabels.Exists() {
			userLabels = user.ContainerLabels
		}
		permit = user.Roles.Has(auth.Actions)
	}

	if !permit {
		log.Warn().Msg("user is not permitted to perform actions on container")
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return nil, false
	}

	containerService, err := h.hostService.FindContainer(hostKey(r), id, userLabels)
	if err != nil {
		log.Error().Err(err).Msg("error while trying to find container")
		http.Error(w, err.Error(), http.StatusNotFound)
		return nil, false
	}

	return containerService, true
}

func (h *handler) containerActions(w http.ResponseWriter, r *http.Request) {
	action := chi.URLParam(r, "action")

	containerService, ok := h.findContainerWithActions(w, r)
	if !ok {
		return
	}

	parsedAction, err := container.ParseContainerAction(action)
	if err != nil {
		log.Error().Err(err).Msg("error while trying to parse action")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := containerService.Action(r.Context(), parsedAction); err != nil {
		log.Error().Err(err).Msg("error while trying to perform container action")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Info().Str("action", action).Str("container", containerService.Container.Name).Msg("container action performed")
	http.Error(w, "", http.StatusNoContent)
}

func (h *handler) containerUpdate(w http.ResponseWriter, r *http.Request) {
	containerService, ok := h.findContainerWithActions(w, r)
	if !ok {
		return
	}

	sse, err := support_web.NewSSEWriter(r.Context(), w, r)
	if err != nil {
		log.Error().Err(err).Msg("error creating SSE writer")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer sse.Close()

	progressCh := make(chan container.UpdateProgress, 50)
	errCh := make(chan error, 1)

	go func() {
		_, err := containerService.Update(r.Context(), progressCh)
		errCh <- err
	}()

	for progress := range progressCh {
		if err := sse.Event("update-progress", progress); err != nil {
			log.Error().Err(err).Msg("error writing SSE event")
			return
		}
	}

	if err := <-errCh; err != nil {
		log.Error().Err(err).Msg("container update failed")
	}

	log.Info().Str("container", containerService.Container.Name).Msg("container update completed")
}

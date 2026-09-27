package web

import (
	"net/http"

	"github.com/amir20/dozzle/internal/analytics"
	"github.com/amir20/dozzle/internal/auth"
	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/web/sse"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

func (h *handler) findContainerWithActions(w http.ResponseWriter, r *http.Request) (*container.ContainerService, bool) {
	id := chi.URLParam(r, "id")

	if !h.permitActions(r) {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return nil, false
	}

	containerService, err := h.hostService.FindContainer(hostKey(r), id, h.resolveLabels(r))
	if err != nil {
		log.Error().Err(err).Msg("error while trying to find container")
		http.Error(w, err.Error(), http.StatusNotFound)
		return nil, false
	}

	return containerService, true
}

// permitActions reports whether the caller holds the actions role. Without
// login everyone does.
func (h *handler) permitActions(r *http.Request) bool {
	if h.config.Authorization.Provider == NONE || auth.UserFromContext(r.Context()).Roles.Has(auth.Actions) {
		return true
	}
	log.Warn().Msg("user is not permitted to perform actions on container")
	return false
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

	analytics.Count("action." + string(parsedAction))
	log.Info().Str("action", action).Str("container", containerService.Container.Name).Msg("container action performed")
	http.Error(w, "", http.StatusNoContent)
}

func (h *handler) containerUpdate(w http.ResponseWriter, r *http.Request) {
	containerService, ok := h.findContainerWithActions(w, r)
	if !ok {
		return
	}
	analytics.Count("action.update")

	sseWriter, err := sse.NewWriter(r.Context(), w, r)
	if err != nil {
		log.Error().Err(err).Msg("error creating SSE writer")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer sseWriter.Close()

	progressCh := make(chan container.UpdateProgress, 50)
	errCh := make(chan error, 1)

	go func() {
		_, err := containerService.Update(r.Context(), progressCh)
		errCh <- err
	}()

	for progress := range progressCh {
		if err := sseWriter.Event("update-progress", progress); err != nil {
			log.Error().Err(err).Msg("error writing SSE event")
			return
		}
	}

	if err := <-errCh; err != nil {
		log.Error().Err(err).Msg("container update failed")
	}

	log.Info().Str("container", containerService.Container.Name).Msg("container update completed")
}

package web

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/amir20/dozzle/internal/analytics"
	"github.com/amir20/dozzle/internal/auth"
	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/hostservice"
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

// updateRequest is the optional body of a single update.
type updateRequest struct {
	// WatchInCloud is "Have Dozzle Cloud watch this update". See update_watch.go.
	WatchInCloud bool `json:"watchInCloud"`
}

func (h *handler) containerUpdate(w http.ResponseWriter, r *http.Request) {
	containerService, ok := h.findContainerWithActions(w, r)
	if !ok {
		return
	}
	analytics.Count("action.update")

	// The body is optional: an older UI, and anything scripted, posts none.
	var req updateRequest
	if isJSONRequest(r) {
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<10)).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	updateWatches.set(containerService.Container.Host, containerService.Container.ID, req.WatchInCloud)

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
		_, err := containerService.Update(r.Context(), container.UpdateOptions{Source: container.UpdateSourceDozzle}, progressCh)
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

// containerRollbackTarget answers what the container's rollback would go back
// to, the same target containerRollback takes, as JSON, or 204 when it has
// none. The UI offers the rollback only with one, and sends its image id back
// as ?to=.
func (h *handler) containerRollbackTarget(w http.ResponseWriter, r *http.Request) {
	containerService, ok := h.findContainerWithActions(w, r)
	if !ok {
		return
	}
	target, err := containerService.RollbackTarget()
	if err != nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(target); err != nil {
		log.Error().Err(err).Msg("error writing rollback target")
	}
}

// containerRollback swaps a container back to the image it ran before its last
// update, streaming progress like containerUpdate. ?to= is the image id the UI
// offered, so a container whose target changed since is refused rather than
// rolled back somewhere else.
func (h *handler) containerRollback(w http.ResponseWriter, r *http.Request) {
	containerService, ok := h.findContainerWithActions(w, r)
	if !ok {
		return
	}
	analytics.Count("action.rollback")

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
		errCh <- containerService.Rollback(r.Context(), container.RollbackOptions{
			ToImageID: r.URL.Query().Get("to"),
		}, progressCh)
	}()

	for progress := range progressCh {
		if err := sseWriter.Event("update-progress", progress); err != nil {
			log.Error().Err(err).Msg("error writing SSE event")
			// The rollback carries on without its watcher; the schedule must
			// still learn that it happened.
			go func() {
				if err := <-errCh; err == nil {
					RecordRolledBack(containerService.Container)
				}
			}()
			return
		}
	}

	if err := <-errCh; err != nil {
		log.Error().Err(err).Str("container", containerService.Container.Name).Msg("container rollback failed")
		return
	}
	RecordRolledBack(containerService.Container)
	log.Info().Str("container", containerService.Container.Name).Msg("container rollback completed")
}

// workloadRestarter is implemented by the k8s host service only.
type workloadRestarter interface {
	RolloutRestart(ctx context.Context, namespace, kind, name string, labels container.ContainerLabels) error
}

func (h *handler) rolloutRestart(w http.ResponseWriter, r *http.Request) {
	if !h.permitActions(r) {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}

	restarter, ok := h.hostService.(workloadRestarter)
	if !ok {
		http.Error(w, "rollout restart is only supported in Kubernetes mode", http.StatusNotFound)
		return
	}

	namespace, kind, name := chi.URLParam(r, "namespace"), chi.URLParam(r, "kind"), chi.URLParam(r, "name")
	err := restarter.RolloutRestart(r.Context(), namespace, kind, name, h.resolveLabels(r))
	switch {
	case errors.Is(err, hostservice.ErrWorkloadNotFound):
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	case errors.Is(err, errors.ErrUnsupported):
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	case err != nil:
		log.Error().Err(err).Str("kind", kind).Str("name", name).Msg("error while trying to rollout restart")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	analytics.Count("action.rollout-restart")
	http.Error(w, "", http.StatusNoContent)
}

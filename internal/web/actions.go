package web

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

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

func (h *handler) containerUpdate(w http.ResponseWriter, r *http.Request) {
	containerService, ok := h.findContainerWithActions(w, r)
	if !ok {
		return
	}
	// A stopped container is never updated. The host refuses one too, but
	// this answers before the progress stream opens.
	if containerService.Container.State != "running" {
		http.Error(w, container.ErrNotRunning.Error(), http.StatusConflict)
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
		_, err := containerService.Update(r.Context(), container.UpdateSourceDozzle, progressCh)
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

// hostPruner is implemented by the docker host services (server and swarm), not k8s.
type hostPruner interface {
	PruneHost(ctx context.Context, host string, opts container.PruneOptions) (container.PruneResult, *container.Reclaimable, error)
}

// pruneTimeout bounds a prune detached from its request. Deleting hundreds of
// images can take minutes, and a half-done prune because a tab closed is worse
// than one that finishes unseen.
const pruneTimeout = 10 * time.Minute

type pruneResponse struct {
	SpaceReclaimed uint64                 `json:"spaceReclaimed"`
	Reclaimable    *container.Reclaimable `json:"reclaimable"`
}

func (h *handler) pruneHost(w http.ResponseWriter, r *http.Request) {
	if !h.permitActions(r) {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}

	pruner, ok := h.hostService.(hostPruner)
	if !ok {
		http.Error(w, "this mode has nothing to prune", http.StatusNotFound)
		return
	}

	var opts container.PruneOptions
	if err := json.NewDecoder(r.Body).Decode(&opts); err != nil || !opts.Any() {
		http.Error(w, "pick at least one of images, volumes, containers or buildCache", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), pruneTimeout)
	defer cancel()
	host := hostKey(r)
	result, reclaimable, err := pruner.PruneHost(ctx, host, opts)
	switch {
	case errors.Is(err, hostservice.ErrHostNotFound):
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	case errors.Is(err, errors.ErrUnsupported):
		http.Error(w, err.Error(), http.StatusNotImplemented)
		return
	case err != nil:
		log.Error().Err(err).Str("host", host).Msg("error while pruning host")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	analytics.Count("action.prune")
	log.Info().Str("host", host).Interface("prune", opts).Uint64("freed", result.SpaceReclaimed).Msg("host pruned")
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(pruneResponse{SpaceReclaimed: result.SpaceReclaimed, Reclaimable: reclaimable}); err != nil {
		log.Error().Err(err).Msg("error writing prune response")
	}
}

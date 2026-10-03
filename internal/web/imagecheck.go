package web

import (
	"net/http"
	"time"

	"github.com/amir20/dozzle/internal/analytics"
	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/imagecheck"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

// checkImageUpdate reports whether a newer image exists upstream for a
// container. It is deliberately not gated behind EnableActions: knowing a
// container is out of date is useful even when the user updates it themselves
// through compose. Only the update button depends on actions being enabled.
func (h *handler) checkImageUpdate(w http.ResponseWriter, r *http.Request) {
	analytics.Count("image.check")
	id := chi.URLParam(r, "id")

	containerService, err := h.hostService.FindContainer(hostKey(r), id, h.resolveLabels(r))
	if err != nil {
		log.Error().Err(err).Msg("error while trying to find container")
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// A forced check bypasses the digest cache and is what the explicit
	// "check now" affordance sends.
	force := r.URL.Query().Get("force") == "true"

	// In manual mode Dozzle never reaches a registry on its own. Background
	// checks are answered without egress so the frontend can stay uniform.
	if !h.config.ImageCheckMode.Allows(force) {
		writeJSON(w, http.StatusOK, imagecheck.Result{
			Image:     containerService.Container.Image,
			Status:    imagecheck.StatusSkipped,
			Reason:    "image checks are set to manual",
			CheckedAt: time.Now(),
		})
		return
	}

	result, err := containerService.CheckImageUpdate(r.Context(), force)
	if err != nil {
		log.Error().Err(err).Str("container", id).Msg("error while checking for image update")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

type containerImageCheck struct {
	Host   string            `json:"host"`
	ID     string            `json:"id"`
	Result imagecheck.Result `json:"result"`
}

// checkAllImageUpdates answers for every container the caller can see, so the
// dashboard can count what is out of date without anyone opening each one.
func (h *handler) checkAllImageUpdates(w http.ResponseWriter, r *http.Request) {
	analytics.Count("image.check")
	force := r.URL.Query().Get("force") == "true"
	if !h.config.ImageCheckMode.Allows(force) {
		writeJSON(w, http.StatusOK, []containerImageCheck{})
		return
	}

	containers, errs := h.hostService.ListAllContainers(h.resolveLabels(r))
	for _, err := range errs {
		log.Debug().Err(err).Msg("image update check: host unavailable")
	}

	updates := container.CheckImageUpdates(r.Context(), h.hostService, containers, force)
	results := make([]containerImageCheck, len(updates))
	for i, u := range updates {
		results[i] = containerImageCheck{Host: u.Container.Host, ID: u.Container.ID, Result: u.Result}
	}

	writeJSON(w, http.StatusOK, results)
}

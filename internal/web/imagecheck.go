package web

import (
	"net/http"
	"sync"
	"time"

	"github.com/amir20/dozzle/internal/analytics"
	"github.com/amir20/dozzle/internal/imagecheck"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
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
	if h.config.ImageCheckMode == imagecheck.ModeManual && !force {
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
// The checker looks each image up once however many containers run it.
func (h *handler) checkAllImageUpdates(w http.ResponseWriter, r *http.Request) {
	analytics.Count("image.check")
	force := r.URL.Query().Get("force") == "true"
	if h.config.ImageCheckMode == imagecheck.ModeManual && !force {
		writeJSON(w, http.StatusOK, []containerImageCheck{})
		return
	}

	labels := h.resolveLabels(r)
	containers, errs := h.hostService.ListAllContainers(labels)
	for _, err := range errs {
		log.Debug().Err(err).Msg("image update check: host unavailable")
	}

	var (
		mu      sync.Mutex
		results = make([]containerImageCheck, 0, len(containers))
		group   errgroup.Group
	)
	// Each check inspects the container on its own host first, which is the
	// slow half for a remote agent.
	group.SetLimit(8)
	for _, c := range containers {
		if c.State == "deleted" {
			continue
		}
		group.Go(func() error {
			// The list is already filtered to what the caller may see. Passing
			// their labels again would re-list the host for every container.
			service, err := h.hostService.FindContainer(c.Host, c.ID, nil)
			if err != nil {
				return nil
			}
			result, err := service.CheckImageUpdate(r.Context(), force)
			if err != nil {
				// Reported rather than dropped, so an earlier "update available"
				// does not outlive the check that could no longer confirm it.
				log.Debug().Err(err).Str("container", c.Name).Msg("image update check failed")
				result = imagecheck.Result{Image: c.Image, Status: imagecheck.StatusUnknown, Reason: err.Error(), CheckedAt: time.Now()}
			}
			mu.Lock()
			results = append(results, containerImageCheck{Host: c.Host, ID: c.ID, Result: result})
			mu.Unlock()
			return nil
		})
	}
	_ = group.Wait()

	writeJSON(w, http.StatusOK, results)
}

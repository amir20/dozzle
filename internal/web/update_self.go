package web

import (
	"context"
	"net/http"
	"time"

	"github.com/amir20/dozzle/internal/auth"
	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/imagecheck"
	"github.com/amir20/dozzle/internal/web/sse"
	"github.com/rs/zerolog/log"
)

// checkSelfUpdate reports what updating Dozzle would actually pull: the same
// check the scheduler makes, against the tag this container follows. Release
// tags say nothing here. A container on amir20/dozzle:master is offered
// whatever that tag now points at, not the newest release.
func (h *handler) checkSelfUpdate(w http.ResponseWriter, r *http.Request) {
	support := checkAutoUpdateSupport(r.Context(), h.config, h.hostService)
	force := r.URL.Query().Get("force") == "true"
	result := imagecheck.Result{Image: support.Image, CheckedAt: time.Now()}

	switch {
	case support.self.Ref == "":
		result.Status = imagecheck.StatusNotCheckable
		result.Reason = support.Reason
	case pinnedReference(support.self.Ref):
		// A digest or a full version tag never moves, so there is nothing to ask
		// a registry.
		result.Status = imagecheck.StatusPinned
	case h.config.ImageCheckMode == imagecheck.ModeManual && !force:
		// Manual mode means Dozzle never reaches a registry on its own, and this
		// runs whenever someone opens settings.
		result.Status = imagecheck.StatusSkipped
		result.Reason = "image checks are set to manual"
	default:
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		result = selfUpdateCheck(ctx, support.self.Ref, support.self.RepoDigests, force)
	}

	writeJSON(w, http.StatusOK, result)
}

// updateSelf streams the same update-progress events as a container update.
// "done" means the helper container was launched and this process is about to
// be replaced.
func (h *handler) updateSelf(w http.ResponseWriter, r *http.Request) {
	if h.config.Authorization.Provider != NONE {
		user := auth.UserFromContext(r.Context())
		if user == nil || !user.Roles.Has(auth.Actions) {
			log.Warn().Msg("user is not permitted to update dozzle")
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}
	}

	sseWriter, err := sse.NewWriter(r.Context(), w, r)
	if err != nil {
		log.Error().Err(err).Msg("error creating SSE writer")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer sseWriter.Close()

	var last string
	writeFailed := false
	emit := func(p container.UpdateProgress) {
		last = p.Status
		if writeFailed {
			return
		}
		if err := sseWriter.Event("update-progress", p); err != nil {
			log.Error().Err(err).Msg("error writing SSE event")
			writeFailed = true
		}
	}

	id := setupSelfID()
	if id == "" {
		emit(container.UpdateProgress{Status: "error", Error: "Dozzle is not running in a container it can find"})
		return
	}

	log.Info().Str("container", id).Msg("updating dozzle")
	updated, err := runSelfUpdate(r.Context(), id, emit)
	switch {
	case err != nil:
		log.Error().Err(err).Msg("dozzle update failed")
		if last != "error" {
			emit(container.UpdateProgress{Status: "error", Error: err.Error()})
		}
	case updated:
		log.Info().Msg("dozzle update helper launched")
		if last != "done" {
			emit(container.UpdateProgress{Status: "done"})
		}
	default:
		if last != "up-to-date" {
			emit(container.UpdateProgress{Status: "up-to-date"})
		}
	}
}

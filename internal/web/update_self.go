package web

import (
	"net/http"

	"github.com/amir20/dozzle/internal/auth"
	"github.com/amir20/dozzle/internal/container"
	support_web "github.com/amir20/dozzle/internal/support/web"
	"github.com/rs/zerolog/log"
)

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

	sse, err := support_web.NewSSEWriter(r.Context(), w, r)
	if err != nil {
		log.Error().Err(err).Msg("error creating SSE writer")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer sse.Close()

	var last string
	writeFailed := false
	emit := func(p container.UpdateProgress) {
		last = p.Status
		if writeFailed {
			return
		}
		if err := sse.Event("update-progress", p); err != nil {
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

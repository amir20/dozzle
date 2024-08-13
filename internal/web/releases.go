package web

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/amir20/dozzle/internal/cache"
	"github.com/amir20/dozzle/internal/releases"
	"github.com/rs/zerolog/log"
)

var cachedReleases *cache.Cache[[]releases.Release]

func (h *handler) releases(w http.ResponseWriter, r *http.Request) {
	if cachedReleases == nil {
		cachedReleases = cache.New(func() ([]releases.Release, error) {
			return releases.Fetch(h.config.Version)
		}, time.Hour)
	}
	releases, err, hit := cachedReleases.GetWithHit()

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Debug().Err(err).Msg("error fetching releases")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if hit {
		w.Header().Set("X-Cache", "HIT")
	}

	if err := json.NewEncoder(w).Encode(releases); err != nil {
		log.Error().Err(err).Msg("error encoding releases")
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

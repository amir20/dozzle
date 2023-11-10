package web

import (
	"encoding/json"
	"net/http"

	"github.com/amir20/dozzle/internal/releases"
	log "github.com/sirupsen/logrus"
)

func (h *handler) releases(w http.ResponseWriter, r *http.Request) {
	releases, err := releases.Fetch(h.config.Version)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Warnf("error reading releases: %v", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(releases); err != nil {
		log.Errorf("json encoding error while streaming %v", err.Error())
	}
}

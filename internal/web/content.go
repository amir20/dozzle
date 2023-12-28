package web

import (
	"encoding/json"
	"net/http"

	"github.com/amir20/dozzle/internal/content"
	"github.com/go-chi/chi/v5"
	log "github.com/sirupsen/logrus"
)

func (h *handler) staticContent(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	content, err := content.Read(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Warnf("error reading content: %v", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(content); err != nil {
		log.Errorf("json encoding error while streaming %v", err.Error())
	}
}

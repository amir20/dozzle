package web

import (
	"net/http"

	"github.com/rs/zerolog/log"
)

func (h *handler) healthcheck(w http.ResponseWriter, r *http.Request) {
	log.Trace().Msg("Healthcheck request received")

	_, errors := h.multiHostService.ListAllContainers()
	if len(errors) > 0 {
		log.Error().Err(errors[0]).Msg("Error listing containers")
		http.Error(w, "Error listing containers", http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

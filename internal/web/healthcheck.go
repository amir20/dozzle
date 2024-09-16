package web

import (
	"context"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

func (h *handler) healthcheck(w http.ResponseWriter, r *http.Request) {
	log.Trace().Msg("Healthcheck request received")
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	_, errors := h.multiHostService.ListAllContainers(ctx)
	if len(errors) > 0 {
		log.Error().Err(errors[0]).Msg("Error listing containers")
		http.Error(w, "Error listing containers", http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

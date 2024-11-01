package web

import (
	"net/http"

	"github.com/rs/zerolog/log"
)

func (h *handler) healthcheck(w http.ResponseWriter, r *http.Request) {
	log.Trace().Msg("Healthcheck request received")

	for _, host := range h.multiHostService.Hosts() {
		if host.Type == "agent" {
			log.Debug().Str("host", host.ID).Msg("Skipping agent host")
			continue
		}

		_, err := h.multiHostService.ListContainersForHost(host.ID)
		if err != nil {
			log.Error().Err(err).Str("host", host.ID).Msg("Error listing containers")
			http.Error(w, "Error listing containers", http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}

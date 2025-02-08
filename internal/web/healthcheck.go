package web

import (
	"net/http"

	"github.com/rs/zerolog/log"
)

func (h *handler) healthcheck(w http.ResponseWriter, r *http.Request) {
	log.Debug().Msg("Executing healthcheck")

	clients := h.hostService.LocalClients()
	for _, client := range clients {
		if err := client.Ping(r.Context()); err != nil {
			log.Error().Err(err).Str("host", client.Host().Name).Msg("error pinging host")
			w.WriteHeader(http.StatusInternalServerError)
		}
	}

	w.WriteHeader(http.StatusOK)
}

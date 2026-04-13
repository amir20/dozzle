package web

import (
	"context"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog/log"
)

func (h *handler) healthcheck(w http.ResponseWriter, r *http.Request) {
	log.Debug().Msg("Executing healthcheck")

	clients := h.hostService.LocalClients()

	var (
		healthy atomic.Bool
		wg      sync.WaitGroup
	)
	healthy.Store(true)

	for _, client := range clients {
		wg.Go(func() {
			ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
			defer cancel()
			if err := client.Ping(ctx); err != nil {
				log.Error().Err(err).Str("host", client.Host().Name).Msg("error pinging host")
				healthy.Store(false)
			}
		})
	}

	wg.Wait()

	if !healthy.Load() {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

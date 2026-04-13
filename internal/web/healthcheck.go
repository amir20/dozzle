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
		anyHealthy atomic.Bool
		wg         sync.WaitGroup
	)

	for _, client := range clients {
		wg.Go(func() {
			ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
			defer cancel()
			if err := client.Ping(ctx); err != nil {
				log.Warn().Err(err).Str("host", client.Host().Name).Msg("error pinging host")
			} else {
				anyHealthy.Store(true)
			}
		})
	}

	wg.Wait()

	if !anyHealthy.Load() {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

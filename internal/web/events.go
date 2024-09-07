package web

import (
	"net/http"

	"github.com/amir20/dozzle/internal/analytics"
	"github.com/amir20/dozzle/internal/docker"
	docker_support "github.com/amir20/dozzle/internal/support/docker"
	support_web "github.com/amir20/dozzle/internal/support/web"
	"github.com/rs/zerolog/log"
)

func (h *handler) streamEvents(w http.ResponseWriter, r *http.Request) {
	sseWriter, err := support_web.NewSSEWriter(r.Context(), w)
	if err != nil {
		log.Error().Err(err).Msg("error creating sse writer")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ctx := r.Context()
	events := make(chan docker.ContainerEvent)
	stats := make(chan docker.ContainerStat)
	availableHosts := make(chan docker.Host)

	h.multiHostService.SubscribeEventsAndStats(ctx, events, stats)
	h.multiHostService.SubscribeAvailableHosts(ctx, availableHosts)

	allContainers, errors := h.multiHostService.ListAllContainers()

	for _, err := range errors {
		log.Warn().Err(err).Msg("error listing containers")
		if hostNotAvailableError, ok := err.(*docker_support.HostUnavailableError); ok {
			if err := sseWriter.Event("update-host", hostNotAvailableError.Host); err != nil {
				log.Error().Err(err).Msg("error writing event to event stream")
			}
		}
	}

	if err := sseWriter.Event("containers-changed", allContainers); err != nil {
		log.Error().Err(err).Msg("error writing containers to event stream")
	}

	go sendBeaconEvent(h, r, len(allContainers))

	for {
		select {
		case host := <-availableHosts:
			if err := sseWriter.Event("update-host", host); err != nil {
				log.Error().Err(err).Msg("error writing event to event stream")
				return
			}
		case stat := <-stats:
			if err := sseWriter.Event("container-stat", stat); err != nil {
				log.Error().Err(err).Msg("error writing event to event stream")
				return
			}
		case event, ok := <-events:
			if !ok {
				return
			}
			switch event.Name {
			case "start", "die", "destroy":
				if event.Name == "start" {
					log.Debug().Str("container", event.ActorID).Msg("container started")
					if containers, err := h.multiHostService.ListContainersForHost(event.Host); err == nil {
						if err := sseWriter.Event("containers-changed", containers); err != nil {
							log.Error().Err(err).Msg("error writing containers to event stream")
							return
						}
					}
				}

				if err := sseWriter.Event("container-event", event); err != nil {
					log.Error().Err(err).Msg("error writing event to event stream")
					return
				}

			case "health_status: healthy", "health_status: unhealthy":
				log.Debug().Str("container", event.ActorID).Str("health", event.Name).Msg("container health status")
				healthy := "unhealthy"
				if event.Name == "health_status: healthy" {
					healthy = "healthy"
				}
				payload := map[string]string{
					"actorId": event.ActorID,
					"health":  healthy,
				}

				if err := sseWriter.Event("container-health", payload); err != nil {
					log.Error().Err(err).Msg("error writing event to event stream")
					return
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

func sendBeaconEvent(h *handler, r *http.Request, runningContainers int) {
	if h.config.NoAnalytics {
		return
	}
	b := analytics.BeaconEvent{
		AuthProvider:      string(h.config.Authorization.Provider),
		Browser:           r.Header.Get("User-Agent"),
		Clients:           h.multiHostService.TotalClients(),
		HasActions:        h.config.EnableActions,
		HasCustomAddress:  h.config.Addr != ":8080",
		HasCustomBase:     h.config.Base != "/",
		HasHostname:       h.config.Hostname != "",
		Name:              "events",
		RunningContainers: runningContainers,
		Version:           h.config.Version,
	}

	local, err := h.multiHostService.LocalHost()
	if err == nil {
		b.ServerID = local.ID
	}

	if err := analytics.SendBeacon(b); err != nil {
		log.Debug().Err(err).Msg("error sending beacon")
	}
}

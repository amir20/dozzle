package web

import (
	"net/http"

	"github.com/amir20/dozzle/internal/analytics"
	"github.com/amir20/dozzle/internal/auth"
	"github.com/amir20/dozzle/internal/container"
	docker_support "github.com/amir20/dozzle/internal/support/docker"
	support_web "github.com/amir20/dozzle/internal/support/web"
	"github.com/amir20/dozzle/types"
	"github.com/rs/zerolog/log"
)

func (h *handler) streamEvents(w http.ResponseWriter, r *http.Request) {
	sseWriter, err := support_web.NewSSEWriter(r.Context(), w, r)
	if err != nil {
		log.Error().Err(err).Msg("error creating sse writer")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer sseWriter.Close()

	events := make(chan container.ContainerEvent)
	stats := make(chan container.ContainerStat)
	availableHosts := make(chan container.Host)

	h.hostService.SubscribeEventsAndStats(r.Context(), events, stats)
	h.hostService.SubscribeAvailableHosts(r.Context(), availableHosts)

	userLabels := h.config.Labels
	if h.config.Authorization.Provider != NONE {
		user := auth.UserFromContext(r.Context())
		if user.ContainerLabels.Exists() {
			userLabels = user.ContainerLabels
		}
	}

	allContainers, errors := h.hostService.ListAllContainers(userLabels)

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
			log.Debug().Str("event", event.Name).Str("id", event.ActorID).Msg("container event from store")
			switch event.Name {
			case "start", "die", "destroy", "rename":
				if event.Name == "start" || event.Name == "rename" {
					if containers, err := h.hostService.ListContainersForHost(event.Host, userLabels); err == nil {
						log.Debug().Str("host", event.Host).Int("count", len(containers)).Msg("updating containers for host")
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

			case "update":
				if err := sseWriter.Event("container-updated", event.Container); err != nil {
					log.Error().Err(err).Msg("error writing event to event stream")
					return
				}
			case "health_status: healthy", "health_status: unhealthy":
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
		case <-r.Context().Done():
			return
		}
	}
}

func sendBeaconEvent(h *handler, r *http.Request, runningContainers int) {
	if h.config.NoAnalytics {
		return
	}
	b := types.BeaconEvent{
		AuthProvider:      string(h.config.Authorization.Provider),
		Browser:           r.Header.Get("User-Agent"),
		Clients:           len(h.hostService.Hosts()),
		HasActions:        h.config.EnableActions,
		HasCustomAddress:  h.config.Addr != ":8080",
		HasCustomBase:     h.config.Base != "/",
		HasHostname:       h.config.Hostname != "",
		Name:              "events",
		RunningContainers: runningContainers,
		Version:           h.config.Version,
	}

	local, err := h.hostService.LocalHost()
	if err == nil {
		b.ServerID = local.ID
	}

	if err := analytics.SendBeacon(b); err != nil {
		log.Debug().Err(err).Msg("error sending beacon")
	}
}

package web

import (
	"net/http"
	"time"

	"github.com/amir20/dozzle/internal/analytics"
	"github.com/amir20/dozzle/internal/auth"
	"github.com/amir20/dozzle/internal/container"
	docker_support "github.com/amir20/dozzle/internal/support/docker"
	support_web "github.com/amir20/dozzle/internal/support/web"
	"github.com/amir20/dozzle/types"
	"github.com/rs/zerolog/log"
)

const (
	eventBufferSize = 64
	statBufferSize  = 128
	// how often a host whose container list failed to refresh is retried
	staleHostRetryInterval = 5 * time.Second
)

func (h *handler) streamEvents(w http.ResponseWriter, r *http.Request) {
	sseWriter, err := support_web.NewSSEWriter(r.Context(), w, r)
	if err != nil {
		log.Error().Err(err).Msg("error creating sse writer")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer sseWriter.Close()

	// buffered so a momentarily slow client can't stall the shared per-host store loop,
	// which broadcasts to every subscriber with a blocking send
	events := make(chan container.ContainerEvent, eventBufferSize)
	stats := make(chan container.ContainerStat, statBufferSize)
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

	// per-host set of container IDs the caller may see, so stat/event channels stay filtered like the list
	visibleByHost := make(map[string]map[string]struct{})
	setVisible := func(host string, containers []container.Container) {
		ids := make(map[string]struct{}, len(containers))
		for _, c := range containers {
			ids[c.ID] = struct{}{}
		}
		visibleByHost[host] = ids
	}

	// A host whose refresh failed keeps a set that is missing containers started since,
	// and nothing else repopulates it. Without a retry those containers stay invisible for
	// the life of this stream, so the client never sees them start, stop or update again.
	staleHosts := make(map[string]struct{})
	refreshHost := func(host string) ([]container.Container, bool) {
		containers, err := h.hostService.ListContainersForHost(host, userLabels)
		if err != nil {
			log.Warn().Err(err).Str("host", host).Msg("failed to refresh containers, will retry")
			staleHosts[host] = struct{}{}
			return nil, false
		}
		delete(staleHosts, host)
		setVisible(host, containers)
		return containers, true
	}
	isVisible := func(host, id string) bool {
		if host != "" {
			ids, ok := visibleByHost[host]
			if !ok {
				return false
			}
			_, ok = ids[id]
			return ok
		}
		// container-stat payloads carry no host, so fall back to scanning all hosts
		for _, ids := range visibleByHost {
			if _, ok := ids[id]; ok {
				return true
			}
		}
		return false
	}

	for _, c := range allContainers {
		ids, ok := visibleByHost[c.Host]
		if !ok {
			ids = make(map[string]struct{})
			visibleByHost[c.Host] = ids
		}
		ids[c.ID] = struct{}{}
	}

	for _, err := range errors {
		log.Warn().Err(err).Msg("error listing containers")
		if hostNotAvailableError, ok := err.(*docker_support.HostUnavailableError); ok {
			// this host has no visible set at all, so retry it until it answers
			staleHosts[hostNotAvailableError.Host.ID] = struct{}{}
			if err := sseWriter.Event("update-host", hostNotAvailableError.Host); err != nil {
				log.Error().Err(err).Msg("error writing event to event stream")
			}
		}
	}

	// sent on every (re)connect so a long-lived tab can tell it is running UI from an older build
	if err := sseWriter.Event("server-version", map[string]string{"version": h.config.Version}); err != nil {
		log.Error().Err(err).Msg("error writing version to event stream")
	}

	if err := sseWriter.Event("containers-changed", allContainers); err != nil {
		log.Error().Err(err).Msg("error writing containers to event stream")
	}

	go sendBeaconEvent(h, r, len(allContainers))

	// stats never repair a stale host (their payload carries no host) and a quiet host may
	// not emit another event for a long time, so retry on a timer as well
	retry := time.NewTicker(staleHostRetryInterval)
	defer retry.Stop()

	for {
		select {
		case <-retry.C:
			for host := range staleHosts {
				if containers, ok := refreshHost(host); ok {
					log.Debug().Str("host", host).Int("count", len(containers)).Msg("recovered stale host")
					if err := sseWriter.Event("containers-changed", containers); err != nil {
						log.Error().Err(err).Msg("error writing containers to event stream")
						return
					}
				}
			}
		case host := <-availableHosts:
			if err := sseWriter.Event("update-host", host); err != nil {
				log.Error().Err(err).Msg("error writing event to event stream")
				return
			}
		case stat := <-stats:
			if !isVisible("", stat.ID) {
				continue
			}
			if err := sseWriter.Event("container-stat", stat); err != nil {
				log.Error().Err(err).Msg("error writing event to event stream")
				return
			}
		case event, ok := <-events:
			if !ok {
				return
			}
			log.Trace().Str("event", event.Name).Str("id", event.ActorID).Msg("container event from store")

			// a host left stale by an earlier failure gets repaired on its next event, so a
			// single transient list failure can't hide containers for the rest of the stream.
			// start/rename refresh on their own below, so skip the extra list call for those.
			_, stale := staleHosts[event.Host]
			if stale && event.Host != "" && event.Name != "start" && event.Name != "rename" {
				if containers, ok := refreshHost(event.Host); ok {
					if err := sseWriter.Event("containers-changed", containers); err != nil {
						log.Error().Err(err).Msg("error writing containers to event stream")
						return
					}
				}
			}

			switch event.Name {
			case "start", "die", "destroy", "rename", "pause", "unpause":
				var refreshed []container.Container
				if event.Name == "start" || event.Name == "rename" {
					if containers, ok := refreshHost(event.Host); ok {
						log.Debug().Str("host", event.Host).Int("count", len(containers)).Msg("updating containers for host")
						refreshed = containers
					}
				}

				// gate both containers-changed and the raw event so out-of-scope
				// containers don't leak via payload or as a timing side-channel
				if !isVisible(event.Host, event.ActorID) {
					continue
				}

				if refreshed != nil {
					if err := sseWriter.Event("containers-changed", refreshed); err != nil {
						log.Error().Err(err).Msg("error writing containers to event stream")
						return
					}
				}

				if err := sseWriter.Event("container-event", event); err != nil {
					log.Error().Err(err).Msg("error writing event to event stream")
					return
				}

			case "update":
				if event.Container == nil || !isVisible(event.Host, event.Container.ID) {
					continue
				}
				if err := sseWriter.Event("container-updated", event.Container); err != nil {
					log.Error().Err(err).Msg("error writing event to event stream")
					return
				}
			case "health_status: healthy", "health_status: unhealthy":
				if !isVisible(event.Host, event.ActorID) {
					continue
				}
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

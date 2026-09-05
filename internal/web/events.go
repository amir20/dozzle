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
	// minimum gap between retries of a host whose container list failed to refresh
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
	// Retries are driven by the host's own traffic and throttled, so a host that is down
	// and silent is never polled and can't block this loop on every attempt.
	staleHosts := make(map[string]time.Time) // host -> last failed attempt
	refreshHost := func(host string) ([]container.Container, bool) {
		containers, err := h.hostService.ListContainersForHost(host, userLabels)
		if err != nil {
			log.Warn().Err(err).Str("host", host).Msg("failed to refresh containers, will retry")
			staleHosts[host] = time.Now()
			return nil, false
		}
		delete(staleHosts, host)
		setVisible(host, containers)
		return containers, true
	}
	// retries a stale host and pushes the recovered list; false means the client is gone
	repairHost := func(host string) bool {
		if last, stale := staleHosts[host]; !stale || time.Since(last) < staleHostRetryInterval {
			return true
		}
		containers, ok := refreshHost(host)
		if !ok {
			return true
		}
		log.Debug().Str("host", host).Int("count", len(containers)).Msg("recovered stale host")
		if err := sseWriter.Event("containers-changed", containers); err != nil {
			log.Error().Err(err).Msg("error writing containers to event stream")
			return false
		}
		return true
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
			// this host has no visible set at all, so retry as soon as it produces traffic
			staleHosts[hostNotAvailableError.Host.ID] = time.Time{}
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

	for {
		select {
		case host := <-availableHosts:
			// an agent that reconnected has no visible set yet; its first traffic fills it
			if _, ok := visibleByHost[host.ID]; host.Available && !ok {
				staleHosts[host.ID] = time.Time{}
			}
			if err := sseWriter.Event("update-host", host); err != nil {
				log.Error().Err(err).Msg("error writing event to event stream")
				return
			}
		case stat := <-stats:
			if !isVisible("", stat.ID) {
				// an unknown ID may belong to a container a stale host never got to report.
				// a just-started container produces a stat every second, so this also covers
				// a host that is otherwise quiet
				for host := range staleHosts {
					if !repairHost(host) {
						return
					}
				}
				if !isVisible("", stat.ID) {
					continue
				}
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

			// start/rename refresh on their own below; anything else from a stale host is a
			// chance to repair it
			if event.Name != "start" && event.Name != "rename" && !repairHost(event.Host) {
				return
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

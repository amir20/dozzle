package web

import (
	"fmt"
	"net/http"

	"github.com/goccy/go-json"

	"github.com/amir20/dozzle/internal/analytics"
	"github.com/amir20/dozzle/internal/docker"
	docker_support "github.com/amir20/dozzle/internal/support/docker"

	log "github.com/sirupsen/logrus"
)

func (h *handler) streamEvents(w http.ResponseWriter, r *http.Request) {
	f, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-transform")
	w.Header().Add("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	ctx := r.Context()

	allContainers, errors := h.multiHostService.ListAllContainers()

	for _, err := range errors {
		log.Warnf("error listing containers: %v", err)
		if hostNotAvailableError, ok := err.(*docker_support.HostUnavailableError); ok {
			if _, err := fmt.Fprintf(w, "event: host-unavailable\ndata: %s\n\n", hostNotAvailableError.Host.ID); err != nil {
				log.Errorf("error writing event to event stream: %v", err)
			}
		}
	}
	events := make(chan docker.ContainerEvent)
	stats := make(chan docker.ContainerStat)

	h.multiHostService.SubscribeEventsAndStats(ctx, events, stats)

	if err := sendContainersJSON(allContainers, w); err != nil {
		log.Errorf("error writing containers to event stream: %v", err)
	}

	f.Flush()

	go sendBeaconEvent(h, r, len(allContainers))

	for {
		select {
		case stat := <-stats:
			bytes, _ := json.Marshal(stat)
			if _, err := fmt.Fprintf(w, "event: container-stat\ndata: %s\n\n", string(bytes)); err != nil {
				log.Errorf("error writing stat to event stream: %v", err)
				return
			}
			f.Flush()
		case event, ok := <-events:
			if !ok {
				return
			}
			switch event.Name {
			case "start", "die":
				if event.Name == "start" {
					log.Debugf("found new container with id: %v", event.ActorID)
					if containers, err := h.multiHostService.ListContainersForHost(event.Host); err == nil {
						if err := sendContainersJSON(containers, w); err != nil {
							log.Errorf("error encoding containers to stream: %v", err)
							return
						}
					}
				}

				bytes, _ := json.Marshal(event)
				if _, err := fmt.Fprintf(w, "event: container-%s\ndata: %s\n\n", event.Name, string(bytes)); err != nil {
					log.Errorf("error writing event to event stream: %v", err)
					return
				}

				f.Flush()

			case "health_status: healthy", "health_status: unhealthy":
				log.Debugf("triggering docker health event: %v", event.Name)
				healthy := "unhealthy"
				if event.Name == "health_status: healthy" {
					healthy = "healthy"
				}
				payload := map[string]string{
					"actorId": event.ActorID,
					"health":  healthy,
				}
				bytes, _ := json.Marshal(payload)
				if _, err := fmt.Fprintf(w, "event: container-health\ndata: %s\n\n", string(bytes)); err != nil {
					log.Errorf("error writing event to event stream: %v", err)
					return
				}
				f.Flush()
			}
		case <-ctx.Done():
			log.Debugf("context done, closing event stream")
			return
		}
	}
}

func sendBeaconEvent(h *handler, r *http.Request, runningContainers int) {
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

	if h.multiHostService.SwarmMode {
		b.Mode = "swarm"
	}

	if !h.config.NoAnalytics {
		if err := analytics.SendBeacon(b); err != nil {
			log.Debugf("error sending beacon: %v", err)
		}
	}
}

func sendContainersJSON(containers []docker.Container, w http.ResponseWriter) error {
	if _, err := fmt.Fprint(w, "event: containers-changed\ndata: "); err != nil {
		return err
	}

	if err := json.NewEncoder(w).Encode(containers); err != nil {
		return err
	}

	if _, err := fmt.Fprint(w, "\n\n"); err != nil {
		return err
	}

	return nil
}

package web

import (
	"fmt"
	"net/http"

	"github.com/goccy/go-json"

	"github.com/amir20/dozzle/internal/analytics"
	"github.com/amir20/dozzle/internal/docker"

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

	b := analytics.BeaconEvent{
		Name:             "events",
		Version:          h.config.Version,
		Browser:          r.Header.Get("User-Agent"),
		AuthProvider:     string(h.config.Authorization.Provider),
		HasHostname:      h.config.Hostname != "",
		HasCustomBase:    h.config.Base != "/",
		HasCustomAddress: h.config.Addr != ":8080",
		Clients:          len(h.clients),
		HasActions:       h.config.EnableActions,
	}

	allContainers := make([]docker.Container, 0)
	events := make(chan docker.ContainerEvent)
	stats := make(chan docker.ContainerStat)

	for _, store := range h.stores {
		allContainers = append(allContainers, store.List()...)
		store.SubscribeStats(ctx, stats)
		store.Subscribe(ctx, events)
	}

	defer func() {
		for _, store := range h.stores {
			store.Unsubscribe(ctx)
		}
	}()

	if err := sendContainersJSON(allContainers, w); err != nil {
		log.Errorf("error writing containers to event stream: %v", err)
	}
	b.RunningContainers = len(allContainers)
	f.Flush()

	if !h.config.NoAnalytics {
		go func() {
			if err := analytics.SendBeacon(b); err != nil {
				log.Debugf("error sending beacon: %v", err)
			}
		}()
	}

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
					containers := h.stores[event.Host].List()
					if err := sendContainersJSON(containers, w); err != nil {
						log.Errorf("error encoding containers to stream: %v", err)
						return
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

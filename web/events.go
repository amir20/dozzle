package web

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/amir20/dozzle/docker"

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

	client := h.clientFromRequest(r)
	events, err := client.Events(ctx)
	stats := make(chan docker.ContainerStat)

	if err := sendContainersJSON(client, w); err != nil {
		log.Errorf("error while encoding containers to stream: %v", err)
	}
	f.Flush()

	if containers, err := client.ListContainers(); err == nil {
		go func() {
			for _, c := range containers {
				if c.State == "running" {
					if err := client.ContainerStats(ctx, c.ID, stats); err != nil && !errors.Is(err, context.Canceled) {
						log.Errorf("error while streaming container stats: %v", err)
					}
				}
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
				log.Debugf("triggering docker event: %v", event.Name)
				if event.Name == "start" {
					log.Debugf("found new container with id: %v", event.ActorID)
					if err := client.ContainerStats(ctx, event.ActorID, stats); err != nil && !errors.Is(err, context.Canceled) {
						log.Errorf("error when streaming new container stats: %v", err)
					}
					if err := sendContainersJSON(client, w); err != nil {
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
			default:
				log.Tracef("ignoring docker event: %v", event.Name)
				// do nothing
			}
		case <-ctx.Done():
			return
		case <-err:
			return
		}
	}
}

func sendContainersJSON(client docker.Client, w http.ResponseWriter) error {
	containers, err := client.ListContainers()
	if err != nil {
		return err
	}

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

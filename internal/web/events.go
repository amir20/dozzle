package web

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"

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

	events := make(chan docker.ContainerEvent)
	stats := make(chan docker.ContainerStat)

	{
		wg := sync.WaitGroup{}
		wg.Add(len(h.clients))
		results := make(chan []docker.Container, len(h.clients))

		for _, client := range h.clients {
			client.Events(ctx, events)

			go func(client DockerClient) {
				defer wg.Done()
				if containers, err := client.ListContainers(); err == nil {
					results <- containers
					go func(client DockerClient) {
						for _, c := range containers {
							if c.State == "running" {
								if err := client.ContainerStats(ctx, c.ID, stats); err != nil && !errors.Is(err, context.Canceled) {
									log.Errorf("error while streaming container stats: %v", err)
								}
							}
						}
					}(client)
				} else {
					log.Errorf("error while listing containers: %v", err)
				}
			}(client)
		}
		wg.Wait()
		close(results)

		allContainers := []docker.Container{}
		for containers := range results {
			allContainers = append(allContainers, containers...)
		}

		if err := sendContainersJSON(allContainers, w); err != nil {
			log.Errorf("error writing containers to event stream: %v", err)
		}

		f.Flush()
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

					if err := h.clients[event.Host].ContainerStats(ctx, event.ActorID, stats); err != nil && !errors.Is(err, context.Canceled) {
						log.Errorf("error when streaming new container stats: %v", err)
					}
					containers, err := h.clients[event.Host].ListContainers()
					if err != nil {
						log.Errorf("error when listing containers: %v", err)
					}
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
			default:
				log.Tracef("ignoring docker event: %v", event.Name)
				// do nothing
			}
		case <-ctx.Done():
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

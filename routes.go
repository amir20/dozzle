package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/amir20/dozzle/docker"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

func createRoutes(base string, h *handler) *mux.Router {
	r := mux.NewRouter()
	r.Use(setCSPHeaders)
	if base != "/" {
		r.HandleFunc(base, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			http.Redirect(w, req, base+"/", http.StatusMovedPermanently)
		}))
	}
	s := r.PathPrefix(base).Subrouter()
	s.HandleFunc("/api/logs/stream", h.streamLogs)
	s.HandleFunc("/api/logs", h.fetchLogsBetweenDates)
	s.HandleFunc("/api/events/stream", h.streamEvents)
	s.HandleFunc("/version", h.version)
	s.PathPrefix("/").Handler(http.StripPrefix(base, http.HandlerFunc(h.index)))
	return r
}

func setCSPHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'self'; style-src 'self' 'unsafe-inline' fonts.googleapis.com; img-src 'self'; manifest-src 'self'; font-src fonts.gstatic.com; connect-src 'self' api.github.com; require-trusted-types-for 'script'")
		next.ServeHTTP(w, r)
	})
}

func (h *handler) index(w http.ResponseWriter, req *http.Request) {
	fileServer := http.FileServer(h.box)
	if h.box.Has(req.URL.Path) && req.URL.Path != "" && req.URL.Path != "/" {
		fileServer.ServeHTTP(w, req)
	} else {
		text, err := h.box.FindString("index.html")
		if err != nil {
			panic(err)
		}
		tmpl, err := template.New("index.html").Parse(text)
		if err != nil {
			panic(err)
		}

		path := ""
		if base != "/" {
			path = base
		}

		data := struct {
			Base    string
			Version string
		}{path, version}
		err = tmpl.Execute(w, data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func (h *handler) fetchLogsBetweenDates(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=UTF-8")

	from, _ := time.Parse(time.RFC3339, r.URL.Query().Get("from"))
	to, _ := time.Parse(time.RFC3339, r.URL.Query().Get("to"))
	id := r.URL.Query().Get("id")

	messages, _ := h.client.ContainerLogsBetweenDates(r.Context(), id, from, to)

	for _, m := range messages {
		fmt.Fprintln(w, m)
	}
}

func (h *handler) streamLogs(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}

	f, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	container, e := h.client.FindContainer(id)
	if e != nil {
		http.Error(w, e.Error(), http.StatusNotFound)
		return
	}

	messages, err := h.client.ContainerLogs(r.Context(), container.ID, tailSize, r.Header.Get("Last-Event-ID"))

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
Loop:
	for {
		select {
		case message, ok := <-messages:
			if !ok {
				fmt.Fprintf(w, "event: container-stopped\ndata: end of stream\n\n")
				break Loop
			}
			fmt.Fprintf(w, "data: %s\n", message)
			if index := strings.IndexAny(message, " "); index != -1 {
				id := message[:index]
				if _, err := time.Parse(time.RFC3339Nano, id); err == nil {
					fmt.Fprintf(w, "id: %s\n", id)
				}
			}
			fmt.Fprintf(w, "\n")
			f.Flush()
		case e := <-err:
			if e == io.EOF {
				log.Debugf("Container stopped: %v", container.ID)
				fmt.Fprintf(w, "event: container-stopped\ndata: end of stream\n\n")
				f.Flush()
			} else {
				log.Debugf("Error while reading from log stream: %v", e)
				break Loop
			}
		}
	}

	log.WithField("NumGoroutine", runtime.NumGoroutine()).Debug("runtime stats")
}

func (h *handler) streamEvents(w http.ResponseWriter, r *http.Request) {
	f, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	ctx := r.Context()

	messages, err := h.client.Events(ctx)
	stats := make(chan docker.ContainerStat)

	runningContainers := map[string]docker.Container{}
	if containers, err := h.client.ListContainers(); err == nil {
		for _, c := range containers {
			if c.State == "running" {
				h.client.ContainerStats(ctx, c.ID, stats)
				runningContainers[c.ID] = c
			}
		}
	}

	if err := sendContainersJSON(h.client, w); err != nil {
		log.Errorf("Error while encoding containers to stream: %v", err)
	}

	f.Flush()

	delayer := delayedFunc(time.Second)

Loop:
	for {
		select {
		case stat := <-stats:
			bytes, _ := json.Marshal(stat)
			_, err := fmt.Fprintf(w, "event: container-stat\ndata: %s\n\n", string(bytes))
			if err != nil {
				log.Debugf("Error while writing to event stream: %v", err)
				break
			}
			f.Flush()
		case message, ok := <-messages:
			if !ok {
				break Loop
			}
			switch message.Action {
			case "start", "connect", "disconnect", "die":
				log.Debugf("Triggering docker event: %v", message.Action)
				if message.Action == "start" {
					log.Debugf("Scanning for new containers")
					if containers, err := h.client.ListContainers(); err == nil {
						for _, c := range containers {
							if _, ok = runningContainers[c.ID]; c.State == "running" && !ok {
								log.Debugf("Found a new container %v", c.ID)
								h.client.ContainerStats(ctx, c.ID, stats)
								runningContainers[c.ID] = c
							}
						}
					}
				}

				delayer(func() {
					if err := sendContainersJSON(h.client, w); err != nil {
						log.Errorf("Error while encoding containers to stream: %v", err)
					}
				})

				f.Flush()
			default:
				log.Debugf("Ignoring docker event: %v", message.Action)
			}
		case <-ctx.Done():
			break Loop
		case <-err:
			break Loop
		}
	}
}

func (h *handler) version(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, version)
}

func sendContainersJSON(client docker.Client, w http.ResponseWriter) error {
	if containers, err := client.ListContainers(); err != nil {
		return err
	} else {
		if _, err := fmt.Fprint(w, "event: containers-changed\ndata: "); err != nil {
			return err
		}

		if err := json.NewEncoder(w).Encode(containers); err != nil {
			return err
		}

		if _, err := fmt.Fprint(w, "\n\n"); err != nil {
			return err
		}
	}

	return nil
}

type delayedFun struct {
	mu    sync.Mutex
	after time.Duration
	timer *time.Timer
}

func delayedFunc(after time.Duration) func(f func()) {
	d := &delayedFun{after: after}

	return func(f func()) {
		d.add(f)
	}
}

func (d *delayedFun) add(f func()) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.timer != nil {
		d.timer.Stop()
	}
	d.timer = time.AfterFunc(d.after, f)
}

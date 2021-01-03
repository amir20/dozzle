package web

import (
	"bufio"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"runtime"
	"strings"

	"time"

	"github.com/amir20/dozzle/docker"
	"github.com/dustin/go-humanize"
	"github.com/gobuffalo/packr"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

// Config is a struct for configuring the web service
type Config struct {
	Base     string
	Addr     string
	Version  string
	TailSize int
}

type handler struct {
	client docker.Client
	box    packr.Box
	config *Config
}

// CreateServer creates a service for http handler
func CreateServer(c docker.Client, b packr.Box, config Config) *http.Server {
	handler := &handler{
		client: c,
		box:    b,
		config: &config,
	}
	return &http.Server{Addr: config.Addr, Handler: createRouter(handler)}
}

func createRouter(h *handler) *mux.Router {
	base := h.config.Base
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
		if h.config.Base != "/" {
			path = h.config.Base
		}

		data := struct {
			Base    string
			Version string
		}{path, h.config.Version}
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

	reader, err := h.client.ContainerLogsBetweenDates(r.Context(), id, from, to)
	defer reader.Close()

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	io.Copy(w, reader)
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

	container, err := h.client.FindContainer(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	reader, err := h.client.ContainerLogs(r.Context(), container.ID, h.config.TailSize, r.Header.Get("Last-Event-ID"))
	if err != nil {
		if err == io.EOF {
			fmt.Fprintf(w, "event: container-stopped\ndata: end of stream\n\n")
			f.Flush()
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	defer reader.Close()

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		message := scanner.Text()
		fmt.Fprintf(w, "data: %s\n", message)
		if index := strings.IndexAny(message, " "); index != -1 {
			id := message[:index]
			if _, err := time.Parse(time.RFC3339Nano, id); err == nil {
				fmt.Fprintf(w, "id: %s\n", id)
			}
		}
		fmt.Fprintf(w, "\n")
		f.Flush()
	}

	log.Debugf("container stopped: %v", container.ID)
	fmt.Fprintf(w, "event: container-stopped\ndata: end of stream\n\n")
	f.Flush()

	log.WithField("routines", runtime.NumGoroutine()).Debug("runtime goroutine stats")

	if log.IsLevelEnabled(log.DebugLevel) {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		// For info on each, see: https://golang.org/pkg/runtime/#MemStats
		log.WithFields(log.Fields{
			"allocated":      humanize.Bytes(m.Alloc),
			"totalAllocated": humanize.Bytes(m.TotalAlloc),
			"system":         humanize.Bytes(m.Sys),
		}).Debug("runtime mem stats")
	}
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

	events, err := h.client.Events(ctx)
	stats := make(chan docker.ContainerStat)

	if containers, err := h.client.ListContainers(); err == nil {
		for _, c := range containers {
			if c.State == "running" {
				if err := h.client.ContainerStats(ctx, c.ID, stats); err != nil {
					log.Errorf("Error while streaming container stats: %v", err)
				}
			}
		}
	}

	if err := sendContainersJSON(h.client, w); err != nil {
		log.Errorf("Error while encoding containers to stream: %v", err)
	}

	f.Flush()

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
					if err := h.client.ContainerStats(ctx, event.ActorID, stats); err != nil {
						log.Errorf("error when streaming new container stats: %v", err)
					}
					if err := sendContainersJSON(h.client, w); err != nil {
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
			default:
				// do nothing
			}
		case <-ctx.Done():
			return
		case <-err:
			return
		}
	}
}

func (h *handler) version(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, h.config.Version)
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

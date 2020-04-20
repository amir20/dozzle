package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"runtime"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

func (h *handler) index(w http.ResponseWriter, req *http.Request) {
	fileServer := http.FileServer(h.box)
	if h.box.Has(req.URL.Path) && req.URL.Path != "" && req.URL.Path != "/" {
		fileServer.ServeHTTP(w, req)
	} else {
		text, err := h.box.FindString("index.html")
		if err != nil {
			panic(err)
		}
		text = strings.Replace(text, "__BASE__", "{{ .Base }}", -1)
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

func (h *handler) listContainers(w http.ResponseWriter, r *http.Request) {
	containers, err := h.client.ListContainers(h.showAll)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(w).Encode(containers)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
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
		http.Error(w, e.Error(), http.StatusInternalServerError)
		return
	}

	messages, err := h.client.ContainerLogs(r.Context(), container.ID, tailSize)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Transfer-Encoding", "chunked")

	log.Debugf("Starting to stream logs for %s", id)
Loop:
	for {
		select {
		case message, ok := <-messages:
			if !ok {
				break Loop
			}
			_, e := fmt.Fprintf(w, "data: %s\n\n", message)
			if e != nil {
				log.Debugf("Error while writing to log stream: %v", e)
				break Loop
			}
			f.Flush()
		case e := <-err:
			log.Debugf("Error while reading from log stream: %v", e)
			break Loop
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
	w.Header().Set("Transfer-Encoding", "chunked")

	ctx := r.Context()
	messages, err := h.client.Events(ctx)

Loop:
	for {
		select {
		case message, ok := <-messages:
			if !ok {
				break Loop
			}
			switch message.Action {
			case "connect", "disconnect", "create", "destroy", "start", "stop":
				log.Debugf("Triggering docker event: %v", message.Action)
				_, err := fmt.Fprintf(w, "event: containers-changed\ndata: %s\n\n", message.Action)

				if err != nil {
					log.Debugf("Error while writing to event stream: %v", err)
					break
				}
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

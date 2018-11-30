package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/amir20/dozzle/docker"
	"github.com/gobuffalo/packr"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"html/template"
	"io"
	"net/http"
	"strings"
)

var (
	addr    = ""
	base    = ""
	level   = ""
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

type handler struct {
	client docker.Client
	box    packr.Box
}

func init() {
	flag.StringVar(&addr, "addr", ":8080", "http service address")
	flag.StringVar(&base, "base", "/", "base address of the application to mount")
	flag.StringVar(&level, "level", "info", "logging level")
	flag.Parse()

	l, _ := log.ParseLevel(level)
	log.SetLevel(l)

	log.SetFormatter(&log.TextFormatter{
		DisableTimestamp:       true,
		DisableLevelTruncation: true,
	})
}

func main() {
	dockerClient := docker.NewClient()
	_, err := dockerClient.ListContainers()

	if err != nil {
		log.Fatalf("Could not connect to Docker Engine: %v", err)
	}

	box := packr.NewBox("./static")
	h := &handler{dockerClient, box}

	r := mux.NewRouter()

	if base != "/" {
		r.HandleFunc(base, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			http.Redirect(w, req, base+"/", http.StatusMovedPermanently)
		}))
	}

	s := r.PathPrefix(base).Subrouter()
	s.HandleFunc("/api/containers.json", h.listContainers)
	s.HandleFunc("/api/logs/stream", h.streamLogs)
	s.HandleFunc("/api/events/stream", h.streamEvents)
	s.HandleFunc("/version", h.version)
	s.PathPrefix("/").Handler(http.StripPrefix(base, http.HandlerFunc(h.index)))

	log.Infof("Accepting connections on %s", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}

func (h *handler) index(w http.ResponseWriter, req *http.Request) {
	fileServer := http.FileServer(h.box)
	if h.box.Has(req.URL.Path) && req.URL.Path != "" && req.URL.Path != "/" {
		fileServer.ServeHTTP(w, req)
	} else {
		text, _ := h.box.FindString("index.html")
		text = strings.Replace(text, "__BASE__", "{{ .Base }}", -1)
		tmpl, err := template.New("index.html").Parse(text)
		if err != nil {
			panic(err)
		}

		path := ""
		if base != "/" {
			path = base
		}

		data := struct{ Base string }{path}
		err = tmpl.Execute(w, data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func (h *handler) listContainers(w http.ResponseWriter, r *http.Request) {
	containers, err := h.client.ListContainers()
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	reader, err := h.client.ContainerLogs(ctx, id)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer reader.Close()
	go func() {
		<-r.Context().Done()
		reader.Close()
	}()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Transfer-Encoding", "chunked")

	log.Debugf("Starting to stream logs for %s", id)
	hdr := make([]byte, 8)
	var buffer bytes.Buffer
	for {
		_, err := reader.Read(hdr)
		if err != nil {
			log.Debugf("Error while reading from log stream: %v", err)
			break
		}
		count := binary.BigEndian.Uint32(hdr[4:])
		_, err = io.CopyN(&buffer, reader, int64(count))

		if err != nil {
			log.Debugf("Error while reading from log stream: %v", err)
			break
		}
		_, err = fmt.Fprintf(w, "data: %s\n\n", buffer.String())
		buffer.Reset()
		if err != nil {
			log.Debugf("Error while writing to log stream: %v", err)
			break
		}
		f.Flush()
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
	w.Header().Set("Transfer-Encoding", "chunked")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
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
				_, err := fmt.Fprintf(w, "event: containers-changed\n")
				_, err = fmt.Fprintf(w, "data: %s\n\n", message.Action)

				if err != nil {
					log.Debugf("Error while writing to event stream: %v", err)
					break
				}
				f.Flush()
			default:
				log.Debugf("Ignoring docker event: %v", message.Action)
			}
		case <-r.Context().Done():
			cancel()
			break Loop
		case <-err:
			cancel()
			break Loop
		}
	}
}

func (h *handler) version(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, version)
	io.WriteString(w, commit)
	io.WriteString(w, date)
}

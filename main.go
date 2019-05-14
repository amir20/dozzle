package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/amir20/dozzle/docker"
	"github.com/gobuffalo/packr"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"html/template"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"time"
)

var (
	addr    = ""
	base    = ""
	level   = ""
	tailSize = 300
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

type handler struct {
	client docker.Client
	box    packr.Box
}

func init() {
	pflag.String("addr", ":8080", "http service address")
	pflag.String("base", "/", "base address of the application to mount")
	pflag.String("level", "info", "logging level")
	pflag.Int("tailSize", 300, "Tail size to use for initial container logs")
	pflag.Parse()

	viper.AutomaticEnv()
	viper.SetEnvPrefix("DOZZLE")
	viper.BindPFlags(pflag.CommandLine)

	addr = viper.GetString("addr")
	base = viper.GetString("base")
	level = viper.GetString("level")
	tailSize = viper.GetInt("tailSize")

	l, _ := log.ParseLevel(level)
	log.SetLevel(l)

	log.SetFormatter(&log.TextFormatter{
		DisableTimestamp:       true,
		DisableLevelTruncation: true,
	})
}

func createRoutes(base string, h *handler) *mux.Router {
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
	return r
}

func main() {
	log.Infof("Dozzle version %s", version)
	dockerClient := docker.NewClient()
	_, err := dockerClient.ListContainers()

	if err != nil {
		log.Fatalf("Could not connect to Docker Engine: %v", err)
	}

	box := packr.NewBox("./static")
	r := createRoutes(base, &handler{dockerClient, box})
	srv := &http.Server{Addr: addr, Handler: r}

	go func() {
		log.Infof("Accepting connections on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, os.Kill)
	<-c
	log.Infof("Shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
	os.Exit(0)
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
	messages, err := h.client.ContainerLogs(r.Context(), id, tailSize)

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
	fmt.Fprintln(w, commit)
	fmt.Fprintln(w, date)
}

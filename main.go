package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"time"

	"github.com/amir20/dozzle/docker"
	"github.com/gobuffalo/packr"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var (
	addr     = ""
	base     = ""
	level    = ""
	showAll  = false
	tailSize = 300
	filters  map[string]string
	version  = "dev"
	commit   = "none"
	date     = "unknown"
)

type handler struct {
	client  docker.Client
	showAll bool
	box     packr.Box
}

func init() {
	pflag.String("addr", ":8080", "http service address")
	pflag.String("base", "/", "base address of the application to mount")
	pflag.Bool("showAll", false, "show all containers, even stopped")
	pflag.String("level", "info", "logging level")
	pflag.Int("tailSize", 300, "Tail size to use for initial container logs")
	pflag.StringToStringVar(&filters, "filter", map[string]string{}, "Container filters to use for showing logs")
	pflag.Parse()

	viper.AutomaticEnv()
	viper.SetEnvPrefix("DOZZLE")
	viper.BindPFlags(pflag.CommandLine)

	addr = viper.GetString("addr")
	base = viper.GetString("base")
	level = viper.GetString("level")
	tailSize = viper.GetInt("tailSize")
	showAll = viper.GetBool("showAll")

	// Until https://github.com/spf13/viper/issues/608 is fixed. We have to use this hacky way.
	// filters = viper.GetStringSlice("filter")
	if value, ok := os.LookupEnv("DOZZLE_FILTER"); ok {
		log.Infof("Parsing %s", value)
		urlValues, err := url.ParseQuery(strings.ReplaceAll(value, ",", "&"))
		if err != nil {
			log.Fatal(err)
		}
		filters = map[string]string{}
		for k, v := range urlValues {
			filters[k] = v[0]
		}
	}

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
	s.HandleFunc("/api/logs", h.fetchLogsBetweenDates)
	s.HandleFunc("/api/events/stream", h.streamEvents)
	s.HandleFunc("/version", h.version)
	s.PathPrefix("/").Handler(http.StripPrefix(base, http.HandlerFunc(h.index)))
	return r
}

func main() {
	log.Infof("Dozzle version %s", version)
	dockerClient := docker.NewClientWithFilters(filters)
	_, err := dockerClient.ListContainers(true)

	if err != nil {
		log.Fatalf("Could not connect to Docker Engine: %v", err)
	}

	box := packr.NewBox("./static")
	r := createRoutes(base, &handler{
		client:  dockerClient,
		showAll: showAll,
		box:     box,
	})
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
	fmt.Fprintln(w, commit)
	fmt.Fprintln(w, date)
}

package main

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"os/signal"
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

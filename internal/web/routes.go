package web

import (
	"context"
	"io"
	"io/fs"
	"time"

	"net/http"
	"strings"

	"github.com/amir20/dozzle/internal/auth"
	"github.com/amir20/dozzle/internal/docker"
	"github.com/docker/docker/api/types"
	"github.com/go-chi/chi/v5"
	log "github.com/sirupsen/logrus"
)

// Config is a struct for configuring the web service
type Config struct {
	Base         string
	Addr         string
	Version      string
	Username     string
	Password     string
	Hostname     string
	NoAnalytics  bool
	Dev          bool
	AuthProvider string
}

type handler struct {
	clients map[string]DockerClient
	content fs.FS
	config  *Config
}

// Client is a proxy around the docker client
type DockerClient interface {
	ListContainers() ([]docker.Container, error)
	FindContainer(string) (docker.Container, error)
	ContainerLogs(context.Context, string, string, docker.StdType) (io.ReadCloser, error)
	Events(context.Context, chan<- docker.ContainerEvent) <-chan error
	ContainerLogsBetweenDates(context.Context, string, time.Time, time.Time, docker.StdType) (io.ReadCloser, error)
	ContainerStats(context.Context, string, chan<- docker.ContainerStat) error
	Ping(context.Context) (types.Ping, error)
	Host() *docker.Host
}

func CreateServer(clients map[string]DockerClient, content fs.FS, config Config) *http.Server {
	handler := &handler{
		clients: clients,
		content: content,
		config:  &config,
	}
	return &http.Server{Addr: config.Addr, Handler: createRouter(handler)}
}

var fileServer http.Handler

func createRouter(h *handler) *chi.Mux {
	initializeAuth(h)

	base := h.config.Base
	r := chi.NewRouter()

	if !h.config.Dev {
		r.Use(cspHeaders)
	}

	r.Route(base, func(r chi.Router) {
		r.Group(func(r chi.Router) {
			if h.config.AuthProvider == "forward-proxy" {
				r.Use(auth.ForwardProxyAuthorizationRequired)
			}
			r.Group(func(r chi.Router) {
				r.Use(authorizationRequired)
				r.Get("/api/logs/stream/{host}/{id}", h.streamLogs)
				r.Get("/api/logs/download/{host}/{id}", h.downloadLogs)
				r.Get("/api/logs/{host}/{id}", h.fetchLogsBetweenDates)
				r.Get("/api/events/stream", h.streamEvents)
				r.Get("/logout", h.clearSession)
				r.Get("/version", h.version)
			})

			defaultHandler := http.StripPrefix(strings.Replace(base+"/", "//", "/", 1), http.HandlerFunc(h.index))
			r.Get("/*", func(w http.ResponseWriter, req *http.Request) {
				defaultHandler.ServeHTTP(w, req)
			})
		})

		r.Post("/api/validateCredentials", h.validateCredentials)
		r.Get("/healthcheck", h.healthcheck)
	})

	if base != "/" {
		r.Get(base, func(w http.ResponseWriter, req *http.Request) {
			http.Redirect(w, req, base+"/", http.StatusMovedPermanently)
		})
	}

	fileServer = http.FileServer(http.FS(h.content))

	return r
}

func (h *handler) clientFromRequest(r *http.Request) DockerClient {
	host := chi.URLParam(r, "host")

	if host == "" {
		log.Fatalf("No host found for url %v", r.URL)
	}

	if client, ok := h.clients[host]; ok {
		return client
	}

	log.Fatalf("No client found for host %v and url %v", host, r.URL)
	return nil
}

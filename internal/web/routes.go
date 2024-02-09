package web

import (
	"context"
	"io/fs"

	"net/http"
	"strings"

	"github.com/amir20/dozzle/internal/auth"
	"github.com/amir20/dozzle/internal/docker"

	"github.com/go-chi/chi/v5"
	log "github.com/sirupsen/logrus"
)

type AuthProvider string

const (
	NONE          AuthProvider = "none"
	SIMPLE        AuthProvider = "simple"
	FORWARD_PROXY AuthProvider = "forward-proxy"
)

// Config is a struct for configuring the web service
type Config struct {
	Base          string
	Addr          string
	Version       string
	Hostname      string
	NoAnalytics   bool
	Dev           bool
	Authorization Authorization
	EnableActions bool
}

type Authorization struct {
	Provider   AuthProvider
	Authorizer Authorizer
}

type Authorizer interface {
	AuthMiddleware(http.Handler) http.Handler
	CreateToken(string, string) (string, error)
}

type handler struct {
	clients map[string]docker.Client
	stores  map[string]*docker.ContainerStore
	content fs.FS
	config  *Config
}

func CreateServer(clients map[string]docker.Client, content fs.FS, config Config) *http.Server {
	stores := make(map[string]*docker.ContainerStore)
	for host, client := range clients {
		stores[host] = docker.NewContainerStore(context.Background(), client)
	}

	handler := &handler{
		clients: clients,
		content: content,
		config:  &config,
		stores:  stores,
	}

	return &http.Server{Addr: config.Addr, Handler: createRouter(handler)}
}

var fileServer http.Handler

func createRouter(h *handler) *chi.Mux {
	base := h.config.Base
	r := chi.NewRouter()

	if !h.config.Dev {
		r.Use(cspHeaders)
	}

	if h.config.Authorization.Provider != NONE && h.config.Authorization.Authorizer == nil {
		log.Panic("Authorization provider is set but no authorizer is provided")
	}

	r.Route(base, func(r chi.Router) {
		if h.config.Authorization.Provider != NONE {
			r.Use(h.config.Authorization.Authorizer.AuthMiddleware)
		}
		r.Group(func(r chi.Router) {
			r.Group(func(r chi.Router) {
				if h.config.Authorization.Provider != NONE {
					r.Use(auth.RequireAuthentication)
				}
				r.Get("/api/logs/stream/{host}/{id}", h.streamLogs)
				r.Get("/api/logs/download/{host}/{id}", h.downloadLogs)
				r.Get("/api/logs/{host}/{id}", h.fetchLogsBetweenDates)
				r.Get("/api/events/stream", h.streamEvents)
				if h.config.EnableActions {
					r.Post("/api/actions/{action}/{host}/{id}", h.containerActions)
				}
				r.Get("/api/releases", h.releases)
				r.Get("/api/profile/avatar", h.avatar)
				r.Patch("/api/profile", h.updateProfile)
				r.Get("/version", h.version)
			})

			defaultHandler := http.StripPrefix(strings.Replace(base+"/", "//", "/", 1), http.HandlerFunc(h.index))
			r.Get("/*", func(w http.ResponseWriter, req *http.Request) {
				defaultHandler.ServeHTTP(w, req)
			})
		})

		if h.config.Authorization.Provider == SIMPLE {
			r.Post("/api/token", h.createToken)
			r.Delete("/api/token", h.deleteToken)
		}

		r.Get("/healthcheck", h.healthcheck)

		// r.Mount("/debug", middleware.Profiler())
	})

	if base != "/" {
		r.Get(base, func(w http.ResponseWriter, req *http.Request) {
			http.Redirect(w, req, base+"/", http.StatusMovedPermanently)
		})
	}

	fileServer = http.FileServer(http.FS(h.content))

	return r
}

func (h *handler) clientFromRequest(r *http.Request) docker.Client {
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

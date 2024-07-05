package web

import (
	"io/fs"

	"net/http"
	"strings"

	"github.com/amir20/dozzle/internal/auth"
	docker_support "github.com/amir20/dozzle/internal/support/docker"

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
	content          fs.FS
	config           *Config
	multiHostService *docker_support.MultiHostService
}

type MultiHostService = docker_support.MultiHostService
type ContainerFilter = docker_support.ContainerFilter

func CreateServer(multiHostService *MultiHostService, content fs.FS, config Config) *http.Server {
	handler := &handler{
		content:          content,
		config:           &config,
		multiHostService: multiHostService,
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
				r.Get("/api/hosts/{host}/containers/{id}/logs/stream", h.streamContainerLogs)
				r.Get("/api/hosts/{host}/containers/{id}/logs/download", h.downloadLogs)
				r.Get("/api/hosts/{host}/containers/{id}/logs", h.fetchLogsBetweenDates)
				r.Get("/api/hosts/{host}/logs/mergedStream", h.streamLogsMerged)
				r.Get("/api/stacks/{stack}/logs/stream", h.streamStackLogs)
				r.Get("/api/services/{service}/logs/stream", h.streamServiceLogs)
				r.Get("/api/groups/{group}/logs/stream", h.streamGroupedLogs)
				r.Get("/api/events/stream", h.streamEvents)
				if h.config.EnableActions {
					r.Post("/api/hosts/{host}/containers/{id}/actions/{action}", h.containerActions)
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

func hostKey(r *http.Request) string {
	host := chi.URLParam(r, "host")

	if host == "" {
		log.Fatalf("No host found for url %v", r.URL)
	}

	return host
}

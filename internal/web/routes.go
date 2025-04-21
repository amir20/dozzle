package web

import (
	"context"
	"io/fs"
	"time"

	"net/http"
	"strings"

	"github.com/amir20/dozzle/internal/auth"
	"github.com/amir20/dozzle/internal/container"
	container_support "github.com/amir20/dozzle/internal/support/container"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog/log"
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
	EnableShell   bool
	Labels        container.ContainerLabels
}

type Authorization struct {
	Provider   AuthProvider
	Authorizer Authorizer
	TTL        time.Duration
}

type Authorizer interface {
	AuthMiddleware(http.Handler) http.Handler
	CreateToken(string, string) (string, error)
}

type HostService interface {
	FindContainer(host string, id string, labels container.ContainerLabels) (*container_support.ContainerService, error)
	ListContainersForHost(host string, labels container.ContainerLabels) ([]container.Container, error)
	ListAllContainers(labels container.ContainerLabels) ([]container.Container, []error)
	ListAllContainersFiltered(userFilter container.ContainerLabels, filter container_support.ContainerFilter) ([]container.Container, []error)
	SubscribeEventsAndStats(ctx context.Context, events chan<- container.ContainerEvent, stats chan<- container.ContainerStat)
	SubscribeContainersStarted(ctx context.Context, containers chan<- container.Container, filter container_support.ContainerFilter)
	Hosts() []container.Host
	LocalHost() (container.Host, error)
	SubscribeAvailableHosts(ctx context.Context, hosts chan<- container.Host)
	LocalClients() []container.Client
}

type handler struct {
	content     fs.FS
	config      *Config
	hostService HostService
}

func CreateServer(hostService HostService, content fs.FS, config Config) *http.Server {
	handler := &handler{
		content:     content,
		config:      &config,
		hostService: hostService,
	}

	return &http.Server{Addr: config.Addr, Handler: createRouter(handler)}
}

var fileServer http.Handler

func createRouter(h *handler) *chi.Mux {
	fileServer = http.FileServer(http.FS(h.content))
	base := h.config.Base
	r := chi.NewRouter()

	if !h.config.Dev {
		r.Use(cspHeaders)
	}

	if h.config.Authorization.Provider != NONE && h.config.Authorization.Authorizer == nil {
		log.Fatal().Msg("Authorization provider is set but no authorizer is provided")
	}

	r.Route(base, func(r chi.Router) {
		if h.config.Authorization.Provider != NONE {
			r.Use(h.config.Authorization.Authorizer.AuthMiddleware)
		}

		r.Route("/api", func(r chi.Router) {
			// Authenticated routes
			r.Group(func(r chi.Router) {
				if h.config.Authorization.Provider != NONE {
					r.Use(auth.RequireAuthentication)
				}
				r.Get("/hosts/{host}/containers/{id}/logs/stream", h.streamContainerLogs)
				r.Get("/hosts/{host}/logs/stream", h.streamHostLogs)
				r.Get("/hosts/{host}/containers/{id}/logs", h.fetchLogsBetweenDates)
				r.Get("/hosts/{host}/logs/mergedStream/{ids}", h.streamLogsMerged)
				r.Get("/containers/{hostIds}/download", h.downloadLogs) // formatted as host:container,host:container
				r.Get("/stacks/{stack}/logs/stream", h.streamStackLogs)
				r.Get("/services/{service}/logs/stream", h.streamServiceLogs)
				r.Get("/groups/{group}/logs/stream", h.streamGroupedLogs)
				r.Get("/events/stream", h.streamEvents)
				if h.config.EnableActions {
					r.Post("/hosts/{host}/containers/{id}/actions/{action}", h.containerActions)
				}
				if h.config.EnableShell {
					r.Get("/hosts/{host}/containers/{id}/attach", h.attach)
					r.Get("/hosts/{host}/containers/{id}/exec", h.exec)
				}
				r.Get("/releases", h.releases)
				r.Get("/profile/avatar", h.avatar)
				r.Patch("/profile", h.updateProfile)
				r.Get("/version", h.version)
				if log.Debug().Enabled() {
					r.Get("/debug/store", h.debugStore)
				}
			})

			// Public API routes
			if h.config.Authorization.Provider == SIMPLE {
				r.Post("/token", h.createToken)
				r.Delete("/token", h.deleteToken)
			}
		})

		r.Get("/healthcheck", h.healthcheck)

		defaultHandler := http.StripPrefix(strings.Replace(base+"/", "//", "/", 1), http.HandlerFunc(h.index))
		r.With(Brotli).Get("/*", func(w http.ResponseWriter, req *http.Request) {
			defaultHandler.ServeHTTP(w, req)
		})
	})

	if base != "/" {
		r.Get(base, func(w http.ResponseWriter, req *http.Request) {
			http.Redirect(w, req, base+"/", http.StatusMovedPermanently)
		})
	}

	if log.Debug().Enabled() {
		r.Mount("/debug", middleware.Profiler())
	}

	return r
}

func hostKey(r *http.Request) string {
	host := chi.URLParam(r, "host")

	if host == "" {
		log.Fatal().Str("url", r.URL.String()).Msg("Host parameter not found in the URL path")
	}

	return host
}

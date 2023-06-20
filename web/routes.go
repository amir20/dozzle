package web

import (
	"fmt"
	"html/template"
	"io"
	"io/fs"

	"net/http"
	"os"
	"path"
	"strings"

	"github.com/amir20/dozzle/analytics"
	"github.com/amir20/dozzle/docker"
	"github.com/go-chi/chi/v5"

	log "github.com/sirupsen/logrus"
)

// Config is a struct for configuring the web service
type Config struct {
	Base        string
	Addr        string
	Version     string
	Username    string
	Password    string
	Hostname    string
	NoAnalytics bool
}

type handler struct {
	clients map[string]docker.Client
	content fs.FS
	config  *Config
}

// CreateServer creates a service for http handler
func CreateServer(clients map[string]docker.Client, content fs.FS, config Config) *http.Server {
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

	r.Use(cspHeaders)

	r.Route(base, func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(authorizationRequired)
			r.Get("/api/logs/stream", h.streamLogs)
			r.Get("/api/events/stream", h.streamEvents)
			r.Get("/api/logs/download", h.downloadLogs)
			r.Get("/api/logs", h.fetchLogsBetweenDates)
			r.Get("/logout", h.clearSession)
			r.Get("/version", h.version)
		})

		r.Post("/api/validateCredentials", h.validateCredentials)
		r.Get("/healthcheck", h.healthcheck)
		defaultHandler := http.StripPrefix(strings.Replace(base+"/", "//", "/", 1), http.HandlerFunc(h.index))
		r.NotFound(func(w http.ResponseWriter, req *http.Request) {
			defaultHandler.ServeHTTP(w, req)
		})
	})

	if base != "/" {
		r.Get(base, func(w http.ResponseWriter, req *http.Request) {
			http.Redirect(w, req, base+"/", http.StatusMovedPermanently)
		})
	}

	fileServer = http.FileServer(http.FS(h.content))

	return r
}

func (h *handler) index(w http.ResponseWriter, req *http.Request) {
	_, err := h.content.Open(req.URL.Path)
	if err == nil && req.URL.Path != "" && req.URL.Path != "/" {
		fileServer.ServeHTTP(w, req)
		if !h.config.NoAnalytics {
			go func() {
				host, _ := os.Hostname()

				var client docker.Client
				for _, v := range h.clients {
					client = v
					break
				}

				if containers, err := client.ListContainers(); err == nil {
					totalContainers := len(containers)
					runningContainers := 0
					for _, container := range containers {
						if container.State == "running" {
							runningContainers++
						}
					}

					re := analytics.RequestEvent{
						ClientId:          host,
						TotalContainers:   totalContainers,
						RunningContainers: runningContainers,
					}
					analytics.SendRequestEvent(re)
				}
			}()
		}
	} else {
		if !isAuthorized(req) && req.URL.Path != "login" {
			http.Redirect(w, req, path.Clean(h.config.Base+"/login"), http.StatusTemporaryRedirect)
			return
		}
		h.executeTemplate(w, req)
	}
}

func (h *handler) executeTemplate(w http.ResponseWriter, req *http.Request) {
	file, err := h.content.Open("index.html")
	if err != nil {
		log.Panic(err)
	}
	bytes, err := io.ReadAll(file)
	if err != nil {
		log.Panic(err)
	}
	tmpl, err := template.New("index.html").Parse(string(bytes))
	if err != nil {
		log.Panic(err)
	}

	path := ""
	if h.config.Base != "/" {
		path = h.config.Base
	}

	// Get all keys from hosts map
	hosts := make([]string, 0, len(h.clients))
	for k := range h.clients {
		hosts = append(hosts, k)
	}

	data := struct {
		Base                string
		Version             string
		AuthorizationNeeded bool
		Secured             bool
		Hostname            string
		Hosts               string
	}{
		path,
		h.config.Version,
		h.isAuthorizationNeeded(req),
		secured,
		h.config.Hostname,
		strings.Join(hosts, ","),
	}
	err = tmpl.Execute(w, data)
	if err != nil {
		log.Panic(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *handler) version(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/html")
	fmt.Fprintf(w, "<pre>%v</pre>", h.config.Version)
}

func (h *handler) healthcheck(w http.ResponseWriter, r *http.Request) {
	log.Trace("Executing healthcheck request")
	var client docker.Client
	for _, v := range h.clients {
		client = v
		break
	}

	if ping, err := client.Ping(r.Context()); err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		fmt.Fprintf(w, "OK API Version %v", ping.APIVersion)
	}
}

func (h *handler) clientFromRequest(r *http.Request) docker.Client {
	if !r.URL.Query().Has("host") {
		log.Fatalf("No host parameter found in request %v", r.URL)
	}

	host := r.URL.Query().Get("host")
	if client, ok := h.clients[host]; ok {
		return client
	}

	log.Fatalf("No client found for host %v and url %v", host, r.URL)
	return nil
}

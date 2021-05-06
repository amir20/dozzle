package web

import (
	"fmt"
	"html/template"
	"io/fs"
	"io/ioutil"
	"net/http"
	"path"

	"github.com/amir20/dozzle/docker"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

// Config is a struct for configuring the web service
type Config struct {
	Base     string
	Addr     string
	Version  string
	TailSize int
	Key      string
	Username string
	Password string
}

type handler struct {
	client  docker.Client
	content fs.FS
	config  *Config
}

// CreateServer creates a service for http handler
func CreateServer(c docker.Client, content fs.FS, config Config) *http.Server {
	handler := &handler{
		client:  c,
		content: content,
		config:  &config,
	}
	return &http.Server{Addr: config.Addr, Handler: createRouter(handler)}
}

var fileServer http.Handler

func createRouter(h *handler) *mux.Router {
	initializeAuth(h)

	base := h.config.Base
	r := mux.NewRouter()
	r.Use(cspHeaders)
	if base != "/" {
		r.HandleFunc(base, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			http.Redirect(w, req, base+"/", http.StatusMovedPermanently)
		}))
	}
	s := r.PathPrefix(base).Subrouter()
	s.Handle("/api/logs/stream", authorizationRequired(h.streamLogs))
	s.Handle("/api/logs/download", authorizationRequired(h.downloadLogs))
	s.Handle("/api/logs", authorizationRequired(h.fetchLogsBetweenDates))
	s.Handle("/api/events/stream", authorizationRequired(h.streamEvents))
	s.HandleFunc("/api/validateCredentials", h.validateCredentials)
	s.Handle("/logout", authorizationRequired(h.clearSession))
	s.Handle("/version", authorizationRequired(h.version))

	if log.IsLevelEnabled(log.DebugLevel) {
		s.PathPrefix("/debug/pprof/").Handler(http.DefaultServeMux)
	}

	if base != "/" {
		s.PathPrefix("/").Handler(http.StripPrefix(base+"/", http.HandlerFunc(h.index)))
	} else {
		s.PathPrefix("/").Handler(http.StripPrefix(base, http.HandlerFunc(h.index)))
	}

	fileServer = http.FileServer(http.FS(h.content))

	return r
}

func (h *handler) index(w http.ResponseWriter, req *http.Request) {
	_, err := h.content.Open(req.URL.Path)
	if err == nil && req.URL.Path != "" && req.URL.Path != "/" {
		fileServer.ServeHTTP(w, req)
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
	bytes, err := ioutil.ReadAll(file)
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

	data := struct {
		Base                string
		Version             string
		AuthorizationNeeded bool
		Secured             bool
	}{
		path,
		h.config.Version,
		h.isAuthorizationNeeded(req),
		secured,
	}
	err = tmpl.Execute(w, data)
	if err != nil {
		log.Panic(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *handler) version(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "%v", h.config.Version)
}

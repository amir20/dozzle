package web

import (
	"encoding/json"
	"html/template"
	"io"
	"sort"

	"net/http"
	"os"
	"path"

	"github.com/amir20/dozzle/internal/analytics"
	"github.com/amir20/dozzle/internal/auth"
	"github.com/amir20/dozzle/internal/docker"
	"github.com/amir20/dozzle/internal/profile"

	log "github.com/sirupsen/logrus"
)

func (h *handler) index(w http.ResponseWriter, req *http.Request) {
	_, err := h.content.Open(req.URL.Path)
	if err == nil && req.URL.Path != "" && req.URL.Path != "/" {
		fileServer.ServeHTTP(w, req)
	} else {
		if !isAuthorized(req) && req.URL.Path != "login" {
			http.Redirect(w, req, path.Clean(h.config.Base+"/login"), http.StatusTemporaryRedirect)
			return
		}
		go h.sendRequestEvent()
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
	tmpl, err := template.New("index.html").Funcs(template.FuncMap{
		"marshal": func(v interface{}) template.JS {
			a, _ := json.Marshal(v)
			return template.JS(a)
		},
	}).Parse(string(bytes))
	if err != nil {
		log.Panic(err)
	}

	path := ""
	if h.config.Base != "/" {
		path = h.config.Base
	}

	hosts := make([]*docker.Host, 0, len(h.clients))
	for _, v := range h.clients {
		hosts = append(hosts, v.Host())
	}
	sort.Slice(hosts, func(i, j int) bool {
		return hosts[i].Name < hosts[j].Name
	})

	config := map[string]interface{}{
		"base":                path,
		"version":             h.config.Version,
		"authorizationNeeded": h.isAuthorizationNeeded(req),
		"secured":             secured,
		"hostname":            h.config.Hostname,
		"hosts":               hosts,
	}

	if h.config.AuthProvider == FORWARD_PROXY {
		user := auth.RemoteUserFromContext(req.Context())
		if user == nil {
			log.Error("Unable to find remote user. Please check your proxy configuration. Expecting headers Remote-Email, Remote-User, Remote-Name.")
			log.Debugf("Dumping all headers for url /%s", req.URL.String())
			for k, v := range req.Header {
				log.Debugf("%s: %s", k, v)
			}
			http.Error(w, "Unauthorized user", http.StatusUnauthorized)
			return
		}
		config["user"] = user
		if settings, err := profile.LoadUserSettings(user); err == nil {
			config["serverSettings"] = settings
		} else {
			config["serverSettings"] = struct{}{}
		}
	}

	data := map[string]interface{}{
		"Config":   config,
		"Dev":      h.config.Dev,
		"Manifest": h.readManifest(),
		"Base":     path,
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		log.Panic(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *handler) readManifest() map[string]interface{} {
	if h.config.Dev {
		return map[string]interface{}{}
	} else {
		file, err := h.content.Open("manifest.json")
		if err != nil {
			// this should only happen during test. In production, the file is embedded in the binary and checked in main.go
			return map[string]interface{}{}
		}
		bytes, err := io.ReadAll(file)
		if err != nil {
			log.Fatalf("Could not read manifest.json: %v", err)
		}
		var manifest map[string]interface{}
		err = json.Unmarshal(bytes, &manifest)
		if err != nil {
			log.Fatalf("Could not parse manifest.json: %v", err)
		}
		return manifest
	}
}

func (h *handler) sendRequestEvent() {
	if !h.config.NoAnalytics {
		host, _ := os.Hostname()

		var client DockerClient
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
	}
}

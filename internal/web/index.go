package web

import (
	"html/template"
	"io"
	"mime"
	"path/filepath"
	"sort"
	"strings"

	"github.com/goccy/go-json"

	"net/http"
	"path"

	"github.com/amir20/dozzle/internal/auth"
	"github.com/amir20/dozzle/internal/profile"

	log "github.com/sirupsen/logrus"
)

func (h *handler) index(w http.ResponseWriter, req *http.Request) {
	path := req.URL.Path
	_, err := h.content.Open(path)
	if err == nil && req.URL.Path != "" && req.URL.Path != "/" {
		w.Header().Set("Cache-Control", "max-age=31536000, immutable")
		// if brotli is enabled, then just send over the compressed file
		if file, err := h.content.Open(path + ".br"); strings.Contains(req.Header.Get("Accept-Encoding"), "br") && err == nil {
			w.Header().Set("Content-Encoding", "br")
			w.Header().Set("Content-Type", mime.TypeByExtension(filepath.Ext(path)))
			io.Copy(w, file)
		} else {
			fileServer.ServeHTTP(w, req)
		}
	} else {
		h.executeTemplate(w, req)
	}
}

func (h *handler) executeTemplate(w http.ResponseWriter, req *http.Request) {
	base := ""
	if h.config.Base != "/" {
		base = h.config.Base
	}
	hosts := h.multiHostService.Hosts()
	sort.Slice(hosts, func(i, j int) bool {
		return hosts[i].Name < hosts[j].Name
	})

	config := map[string]interface{}{
		"base": base,
	}

	user := auth.UserFromContext(req.Context())

	if h.config.Authorization.Provider == NONE || user != nil {
		config["authProvider"] = h.config.Authorization.Provider
		config["version"] = h.config.Version
		config["hostname"] = h.config.Hostname
		config["hosts"] = hosts
		config["enableActions"] = h.config.EnableActions
	}

	if user != nil {
		if profile, err := profile.Load(*user); err == nil {
			config["profile"] = profile
		} else {
			config["profile"] = struct{}{}
		}
		config["user"] = user
	} else if h.config.Authorization.Provider == FORWARD_PROXY {
		log.Error("Unable to find remote user. Please check your proxy configuration. Expecting headers Remote-Email, Remote-User, Remote-Name.")
		log.Debugf("Dumping all headers for url /%s", req.URL.String())
		for k, v := range req.Header {
			log.Debugf("%s: %s", k, v)
		}
		http.Error(w, "Unauthorized user", http.StatusUnauthorized)
		return
	} else if h.config.Authorization.Provider == SIMPLE && req.URL.Path != "login" {
		log.Debugf("Redirecting to login page for url /%s", req.URL.String())
		http.Redirect(w, req, path.Clean(h.config.Base+"/login")+"?redirectUrl=/"+req.URL.String(), http.StatusTemporaryRedirect)
		return
	}

	data := map[string]interface{}{
		"Config":   config,
		"Dev":      h.config.Dev,
		"Manifest": h.readManifest(),
		"Base":     base,
	}
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
			var p []byte
			if h.config.Dev {
				p, _ = json.MarshalIndent(v, "", "  ")
			} else {
				p, _ = json.Marshal(v)
			}
			return template.JS(p)
		},
	}).Parse(string(bytes))
	if err != nil {
		log.Panic(err)
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
		file, err := h.content.Open(".vite/manifest.json")
		if err != nil {
			// this should only happen during test. In production, the file is embedded in the binary and checked in main.go
			return map[string]interface{}{}
		}
		bytes, err := io.ReadAll(file)
		if err != nil {
			log.Fatalf("Could not read .vite/manifest.json: %v", err)
		}
		var manifest map[string]interface{}
		err = json.Unmarshal(bytes, &manifest)
		if err != nil {
			log.Fatalf("Could not parse .vite/manifest.json: %v", err)
		}
		return manifest
	}
}

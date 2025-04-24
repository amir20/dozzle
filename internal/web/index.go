package web

import (
	"html/template"
	"io"
	"sort"

	"encoding/json"

	"net/http"
	"path"

	"github.com/amir20/dozzle/internal/auth"
	"github.com/amir20/dozzle/internal/profile"

	"github.com/rs/zerolog/log"
)

func (h *handler) index(w http.ResponseWriter, req *http.Request) {
	path := req.URL.Path
	_, err := h.content.Open(path)
	if err == nil && req.URL.Path != "" && req.URL.Path != "/" {
		w.Header().Set("Cache-Control", "max-age=31536000, immutable")
		fileServer.ServeHTTP(w, req)
	} else {
		h.executeTemplate(w, req)
	}
}

func (h *handler) executeTemplate(w http.ResponseWriter, req *http.Request) {
	base := ""
	if h.config.Base != "/" {
		base = h.config.Base
	}

	hosts := h.hostService.Hosts()
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
		config["enableShell"] = h.config.EnableShell
	}

	if user != nil {
		if profile, err := profile.Load(*user); err == nil {
			config["profile"] = profile
		} else {
			config["profile"] = struct{}{}
		}
		config["user"] = user
	} else if h.config.Authorization.Provider == FORWARD_PROXY {
		log.Error().Msg("Unable to find remote user. Please check your proxy configuration. Expecting headers Remote-Email, Remote-User, Remote-Name.")
		log.Debug().Str("url", req.URL.String()).Msg("Dumping all headers for request")
		for k, v := range req.Header {
			log.Debug().Strs(k, v).Send()
		}
		http.Error(w, "Unauthorized user", http.StatusUnauthorized)
		return
	} else if h.config.Authorization.Provider == SIMPLE && req.URL.Path != "login" {
		log.Debug().Str("url", req.URL.String()).Msg("Redirecting to login page")
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
		log.Fatal().Err(err).Msg("Could not open index.html")
	}
	bytes, err := io.ReadAll(file)
	if err != nil {
		log.Fatal().Err(err).Msg("Could not read index.html")
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
		log.Fatal().Err(err).Msg("Could not parse index.html")
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err = tmpl.Execute(w, data)
	if err != nil {
		log.Fatal().Err(err).Msg("Could not execute index.html")
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
			log.Fatal().Err(err).Msg("Could not read .vite/manifest.json")
		}
		var manifest map[string]interface{}
		err = json.Unmarshal(bytes, &manifest)
		if err != nil {
			log.Fatal().Err(err).Msg("Could not unmarshal .vite/manifest.json")
		}
		return manifest
	}
}

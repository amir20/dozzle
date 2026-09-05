package web

import (
	"html/template"
	"io"
	"sort"
	"strings"

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

	user := auth.UserFromContext(req.Context())

	// Handle unauthorized cases early
	if user == nil {
		switch h.config.Authorization.Provider {
		case FORWARD_PROXY:
			log.Error().Msg("Unable to find remote user. Please check your proxy configuration. Expecting headers Remote-Email, Remote-User, Remote-Name.")
			log.Debug().Str("url", req.URL.String()).Msg("Dumping all headers for request")
			for k, v := range req.Header {
				log.Debug().Strs(k, v).Send()
			}
			http.Error(w, "Unauthorized user", http.StatusUnauthorized)
			return
		case SIMPLE:
			if req.URL.Path != "login" {
				log.Debug().Str("url", req.URL.String()).Msg("Redirecting to login page")
				http.Redirect(w, req, path.Clean(h.config.Base+"/login")+"?redirectUrl=/"+req.URL.String(), http.StatusTemporaryRedirect)
				return
			}
		}
	}

	config := map[string]any{
		"base": base,
	}

	// Build full config when authorized (no auth or authenticated user)
	if h.config.Authorization.Provider == NONE || user != nil {
		hosts := h.hostService.Hosts()
		sort.Slice(hosts, func(i, j int) bool {
			return hosts[i].Name < hosts[j].Name
		})

		config["authProvider"] = h.config.Authorization.Provider
		config["version"] = h.config.Version
		config["hostname"] = h.config.Hostname
		config["mode"] = h.config.Mode
		config["hosts"] = hosts
		config["disableAvatars"] = h.config.DisableAvatars
		config["releaseCheckMode"] = h.config.ReleaseCheckMode
		config["imageCheckMode"] = h.config.ImageCheckMode
		config["enableShell"] = h.config.EnableShell
		config["enableActions"] = h.config.EnableActions
		config["enableDownload"] = true
		config["enableNotifications"] = true
		config["enableCloud"] = true

		if user != nil {
			config["enableShell"] = h.config.EnableShell && user.Roles.Has(auth.Shell)
			config["enableActions"] = h.config.EnableActions && user.Roles.Has(auth.Actions)
			config["enableDownload"] = user.Roles.Has(auth.Download)
			config["enableNotifications"] = user.Roles.Has(auth.Notifications)
			config["enableCloud"] = user.Roles.Has(auth.Cloud)
			config["user"] = user
		}

		if h.config.Authorization.Provider == FORWARD_PROXY && strings.TrimSpace(h.config.Authorization.LogoutUrl) != "" {
			config["logoutUrl"] = strings.TrimSpace(h.config.Authorization.LogoutUrl)
		}
	}

	profileUsername := profile.DefaultUsername
	if user != nil {
		profileUsername = user.Username
	}

	if loadedProfile, err := profile.Load(profileUsername); err == nil {
		config["profile"] = loadedProfile
	} else {
		config["profile"] = struct{}{}
	}

	manifest := h.readManifest()
	entryJS, styles := entryAssets(manifest, "assets/main.ts")

	data := map[string]any{
		"Config": config,
		"Dev":    h.config.Dev,
		"Entry":  entryJS,
		"Styles": styles,
		"Base":   base,
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
		"marshal": func(v any) template.JS {
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

// entryAssets resolves the entry chunk's script and every stylesheet it depends on.
// Vite splits scoped component CSS into its own chunk, and a chunk statically imported by
// the entry gets no <link> of its own, so linking only the entry's `css` left those
// stylesheets to whichever lazy page happened to import them. Deep linking to a page that
// didn't (a container view) then rendered shared components unstyled until the user
// navigated somewhere that pulled the chunk in.
func entryAssets(manifest map[string]any, entry string) (string, []string) {
	chunk, ok := manifest[entry].(map[string]any)
	if !ok {
		return "", nil
	}

	file, _ := chunk["file"].(string)

	seen := make(map[string]bool)
	styles := make([]string, 0, 4)
	var collect func(key string)
	collect = func(key string) {
		if seen[key] {
			return
		}
		seen[key] = true

		chunk, ok := manifest[key].(map[string]any)
		if !ok {
			return
		}

		// Imports first, matching the order Vite itself emits, so the entry's stylesheet
		// (Tailwind's) keeps the last word in the cascade.
		if imports, ok := chunk["imports"].([]any); ok {
			for _, i := range imports {
				if name, ok := i.(string); ok {
					collect(name)
				}
			}
		}

		if css, ok := chunk["css"].([]any); ok {
			for _, c := range css {
				name, ok := c.(string)
				if ok && !seen[name] {
					seen[name] = true
					styles = append(styles, name)
				}
			}
		}
	}
	collect(entry)

	return file, styles
}

func (h *handler) readManifest() map[string]any {
	if h.config.Dev {
		return map[string]any{}
	} else {
		file, err := h.content.Open(".vite/manifest.json")
		if err != nil {
			// this should only happen during test. In production, the file is embedded in the binary and checked in main.go
			return map[string]any{}
		}
		bytes, err := io.ReadAll(file)
		if err != nil {
			log.Fatal().Err(err).Msg("Could not read .vite/manifest.json")
		}
		var manifest map[string]any
		err = json.Unmarshal(bytes, &manifest)
		if err != nil {
			log.Fatal().Err(err).Msg("Could not unmarshal .vite/manifest.json")
		}
		return manifest
	}
}

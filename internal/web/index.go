package web

import (
	"html/template"
	"io"
	"sort"
	"strconv"
	"strings"

	"encoding/json"

	"net/http"
	"path"

	"github.com/amir20/dozzle/internal/auth"
	"github.com/amir20/dozzle/internal/profile"
	"github.com/andybalholm/brotli"
	"github.com/rs/zerolog/log"
)

func (h *handler) index(w http.ResponseWriter, req *http.Request) {
	if path := req.URL.Path; path != "" && path != "/" && h.serveAsset(w, req, path) {
		return
	}
	h.executeTemplate(w, req)
}

// serveAsset serves a built asset, reporting whether it handled the request; anything
// unknown falls through to the SPA template. Text assets exist only as the `.br`
// sibling written by scripts/compress-dist.js, so they are served as-is to the usual
// client and inflated for the rare one that does not accept brotli.
func (h *handler) serveAsset(w http.ResponseWriter, req *http.Request, name string) bool {
	if file, err := h.content.Open(name); err == nil {
		file.Close()
		w.Header().Set("Cache-Control", cacheControlFor(name))
		fileServer.ServeHTTP(w, req)
		return true
	}

	file, err := h.content.Open(name + ".br")
	if err != nil {
		return false
	}
	defer file.Close()

	w.Header().Set("Cache-Control", cacheControlFor(name))
	w.Header().Set("Content-Type", contentTypeFor(name))
	w.Header().Add("Vary", "Accept-Encoding")

	if acceptsBrotli(req) {
		w.Header().Set("Content-Encoding", "br")
		if stat, err := file.Stat(); err == nil {
			w.Header().Set("Content-Length", strconv.FormatInt(stat.Size(), 10))
		}
		io.Copy(w, file)
		return true
	}

	io.Copy(w, brotli.NewReader(file))
	return true
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
	entryJS, styles, preloads := entryAssets(manifest, entryModule)

	data := map[string]any{
		"Config":  config,
		"Dev":     h.config.Dev,
		"Entry":   entryJS,
		"Styles":  styles,
		"Preload": preloads,
		"Base":    base,
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

// cacheControlFor keeps the year-long immutable cache for hashed files under assets/,
// which can always be busted by a new name. The unhashed root files (favicon.png,
// apple-touch-icon.png) cannot, so pinning them would strand a stale icon for a year.
func cacheControlFor(name string) string {
	if strings.HasPrefix(name, "assets/") {
		return "max-age=31536000, immutable"
	}
	return "max-age=3600"
}

const entryModule = "assets/main.ts"

// entryAssets resolves the entry chunk's script, every stylesheet it depends on, and
// every other chunk in its static import graph.
//
// Vite splits scoped component CSS into its own chunk, and a chunk statically imported by
// the entry gets no <link> of its own, so linking only the entry's `css` left those
// stylesheets to whichever lazy page happened to import them. Deep linking to a page that
// didn't (a container view) then rendered shared components unstyled until the user
// navigated somewhere that pulled the chunk in.
//
// The same walk yields the modulepreload list. Vite emits those hints when it owns
// index.html, but the page is a Go template served from public/, so without them the
// browser only discovers a chunk after parsing the one that imports it, serializing the
// graph into several round trips.
func entryAssets(manifest map[string]any, entry string) (string, []string, []string) {
	chunk, ok := manifest[entry].(map[string]any)
	if !ok {
		return "", nil, nil
	}

	file, _ := chunk["file"].(string)

	seen := make(map[string]bool)
	styles := make([]string, 0, 4)
	preloads := make([]string, 0, 32)
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

		// The entry itself is already requested by the <script> tag.
		if f, ok := chunk["file"].(string); ok && key != entry {
			preloads = append(preloads, f)
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

	return file, styles, preloads
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

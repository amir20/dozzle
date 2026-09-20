package web

import (
	"encoding/json"
	"net/http"
)

func (h *handler) manifest(w http.ResponseWriter, req *http.Request) {
	base := basePrefix(h.config.Base)

	manifest := map[string]any{
		"name":       "Dozzle",
		"short_name": "Dozzle",
		// An installed app is identified by its id, and an id that defaults to
		// start_url means any later change to start_url installs a second app
		// beside the first rather than updating it.
		"id":          base + "/",
		"start_url":   base + "/",
		"display":     "standalone",
		"lang":        "en",
		"scope":       base + "/",
		"description": "A log viewer for containers",
		// Both are the dark theme's values, matching what App.vue writes to the
		// theme-color meta. background_color paints the splash before the app has
		// rendered anything, so the alternative is a white flash on every cold
		// start for the theme most of these installs are on.
		"theme_color":      "#121212",
		"background_color": "#121212",
		// Android wants a 192 as well as a 512, and favicon.png is already exactly
		// that. Neither is declared maskable: the logo runs to the edges of the
		// square, so a circular mask would cut the antennae off.
		"icons": []map[string]string{
			{"src": base + "/favicon.png", "sizes": "192x192", "type": "image/png"},
			{"src": base + "/apple-touch-icon.png", "sizes": "512x512", "type": "image/png"},
		},
	}

	w.Header().Set("Content-Type", "application/manifest+json")
	json.NewEncoder(w).Encode(manifest)
}

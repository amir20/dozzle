package web

import (
	"encoding/json"
	"net/http"
)

func (h *handler) manifest(w http.ResponseWriter, req *http.Request) {
	base := ""
	if h.config.Base != "/" {
		base = h.config.Base
	}

	manifest := map[string]any{
		"name":        "Dozzle",
		"short_name":  "Dozzle",
		"start_url":   base + "/",
		"display":     "standalone",
		"lang":        "en",
		"scope":       base + "/",
		"description":  "A log viewer for containers",
		"icons": []map[string]string{
			{"src": base + "/apple-touch-icon.png", "sizes": "512x512", "type": "image/png"},
		},
	}

	w.Header().Set("Content-Type", "application/manifest+json")
	json.NewEncoder(w).Encode(manifest)
}

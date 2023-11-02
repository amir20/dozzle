package web

import (
	"net/http"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/yuin/goldmark"
)

func (h *handler) staticContent(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/html")
	data, err := os.ReadFile("data/content/help.md")
	if err != nil {
		log.Fatalf("Error reading help.md: %v", err)
	}

	if err := goldmark.Convert(data, w); err != nil {
		panic(err)
	}
}

package web

import (
	"fmt"
	"net/http"

	"github.com/amir20/dozzle/internal/support/cli"
)

func (h *handler) version(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/html")
	fmt.Fprintf(w, "<pre>%v  commit: %v</pre>", h.config.Version, cli.SHA)
}

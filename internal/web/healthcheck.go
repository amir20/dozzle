package web

import (
	"fmt"
	"net/http"

	"github.com/amir20/dozzle/internal/docker"
	log "github.com/sirupsen/logrus"
)

func (h *handler) healthcheck(w http.ResponseWriter, r *http.Request) {
	log.Trace("Executing healthcheck request")
	var client docker.Client
	for _, v := range h.clients {
		client = v
		break
	}

	if ping, err := client.Ping(r.Context()); err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		fmt.Fprintf(w, "OK API Version %v", ping.APIVersion)
	}
}

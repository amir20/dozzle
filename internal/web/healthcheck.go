package web

import (
	"net/http"

	log "github.com/sirupsen/logrus"
)

func (h *handler) healthcheck(w http.ResponseWriter, r *http.Request) {
	log.Trace("Executing healthcheck request")

	_, errors := h.multiHostService.ListAllContainers()
	if len(errors) > 0 {
		log.Error(errors)
		http.Error(w, "Error listing containers", http.StatusInternalServerError)
	} else {
		http.Error(w, "OK", http.StatusOK)
	}
}

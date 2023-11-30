package web

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	log "github.com/sirupsen/logrus"
)

func (h *handler) containerActions(w http.ResponseWriter, r *http.Request) {
	action := chi.URLParam(r, "action")
	id := chi.URLParam(r, "id")

	log.Debugf("container action: %s, container id: %s", action, id)

	client := h.clientFromRequest(r)

	if client == nil {
		log.Errorf("no client found for host %v", r.URL)
		w.WriteHeader(http.StatusBadRequest)
	} else if _, err := client.FindContainer(id); err != nil {
		log.Errorf("unable to find container id: %s", id)
		w.WriteHeader(http.StatusNotFound)
	} else if client.ContainerActions(action, id) != nil {
		log.Errorf("error while trying to perform action: %s", action)
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		log.Infof("container action performed: %s; container id: %s", action, id)
		w.WriteHeader(http.StatusOK)
	}
}

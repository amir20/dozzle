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
		return
	}

	container, err := client.FindContainer(id)
	if err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	err = client.ContainerActions(action, container.ID)
	if err != nil {
		log.Errorf("error while trying to perform action: %s", action)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Infof("container action performed: %s; container id: %s", action, id)
	w.WriteHeader(http.StatusOK)
}

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

	_, err := h.multiHostService.FindContainer(hostKey(r), id)
	if err != nil {
		log.Errorf("error while trying to find container: %v", err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// err = client.ContainerActions(action, container.ID)
	// if err != nil {
	// 	log.Errorf("error while trying to perform action: %s", action)
	// 	w.WriteHeader(http.StatusInternalServerError)
	// 	return
	// }

	panic("not implemented")

	log.Infof("container action performed: %s; container id: %s", action, id)
	w.WriteHeader(http.StatusOK)
}

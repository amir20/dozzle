package web

import (
	"net/http"

	"github.com/amir20/dozzle/internal/docker"
	"github.com/go-chi/chi/v5"
	log "github.com/sirupsen/logrus"
)

func (h *handler) containerActions(w http.ResponseWriter, r *http.Request) {
	action := chi.URLParam(r, "action")
	id := chi.URLParam(r, "id")

	log.Debugf("container action: %s, container id: %s", action, id)

	containerService, err := h.multiHostService.FindContainer(hostKey(r), id)
	if err != nil {
		log.Errorf("error while trying to find container: %v", err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	parsedAction, err := docker.ParseContainerAction(action)
	if err != nil {
		log.Errorf("error while trying to parse action: %s", action)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := containerService.Action(parsedAction); err != nil {
		log.Errorf("error while trying to perform action: %s", action)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Infof("container action performed: %s; container id: %s", action, id)
	http.Error(w, "", http.StatusNoContent)
}

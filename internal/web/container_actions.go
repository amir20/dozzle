package web

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (h *handler) containerActions(w http.ResponseWriter, r *http.Request) {
	action := chi.URLParam(r, "action")
	id := chi.URLParam(r, "id")

	for _, client := range h.clients {
		cont, err := client.FindContainer(id)

		if err == nil {
			if client.ContainerActions(action, cont.ID) != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			} else {
				w.WriteHeader(http.StatusOK)
				return
			}
		}
	}

	w.WriteHeader(http.StatusNotFound)
}

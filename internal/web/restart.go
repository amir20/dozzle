package web

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (h *handler) restartContainer(w http.ResponseWriter, r *http.Request) {
	for _, client := range h.clients {
		cont, err := client.FindContainer(chi.URLParam(r, "id"))

		if err == nil {
			if client.RestartContainer(cont.ID) != nil {
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

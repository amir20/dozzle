package web

import (
	"encoding/json"
	"net/http"

	"github.com/amir20/dozzle/internal/container"
)

func (h *handler) debugStore(w http.ResponseWriter, r *http.Request) {
	respone := make(map[string]interface{})
	respone["hosts"] = h.hostService.Hosts()
	containers, errors := h.hostService.ListAllContainers(container.ContainerLabels{})
	respone["containers"] = containers
	respone["errors"] = errors

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(respone)
}

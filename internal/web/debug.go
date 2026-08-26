package web

import (
	"encoding/json"
	"net/http"
)

func (h *handler) debugStore(w http.ResponseWriter, r *http.Request) {
	respone := make(map[string]any)
	respone["hosts"] = h.hostService.Hosts()
	containers, errors := h.hostService.ListAllContainers(h.resolveLabels(r))
	respone["containers"] = containers
	respone["errors"] = errors

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(respone)
}

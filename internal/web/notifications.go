package web

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/amir20/dozzle/internal/notification"
	"github.com/amir20/dozzle/internal/notification/dispatcher"
	"github.com/go-chi/chi/v5"
)

func (h *handler) listSubscriptions(w http.ResponseWriter, r *http.Request) {
	subscriptions := h.hostService.Subscriptions()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(subscriptions)
}

func (h *handler) createSubscription(w http.ResponseWriter, r *http.Request) {
	var sub notification.Subscription
	if err := json.NewDecoder(r.Body).Decode(&sub); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.hostService.AddSubscription(&sub); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(sub)
}

func (h *handler) deleteSubscription(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	h.hostService.RemoveSubscription(id)
	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) listDispatchers(w http.ResponseWriter, r *http.Request) {
	dispatchers := h.hostService.Dispatchers()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dispatchers)
}

type createDispatcherRequest struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

func (h *handler) createDispatcher(w http.ResponseWriter, r *http.Request) {
	var req createDispatcherRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var d dispatcher.Dispatcher
	switch req.Type {
	case "webhook":
		if req.URL == "" {
			http.Error(w, "url is required for webhook dispatcher", http.StatusBadRequest)
			return
		}
		d = dispatcher.NewWebhookDispatcher(req.URL)
	default:
		http.Error(w, "unknown dispatcher type", http.StatusBadRequest)
		return
	}

	id := h.hostService.AddDispatcher(d)

	response := notification.DispatcherConfig{
		ID:   id,
		Type: req.Type,
		URL:  req.URL,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (h *handler) deleteDispatcher(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	h.hostService.RemoveDispatcher(id)
	w.WriteHeader(http.StatusNoContent)
}

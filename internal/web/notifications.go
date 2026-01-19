package web

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/amir20/dozzle/internal/notification"
	"github.com/amir20/dozzle/internal/notification/dispatcher"
	"github.com/go-chi/chi/v5"
)

type subscriptionResponse struct {
	notification.Subscription
	TriggeredContainers int `json:"triggeredContainers"`
}

func (h *handler) listSubscriptions(w http.ResponseWriter, r *http.Request) {
	subscriptions := h.hostService.Subscriptions()
	response := make([]subscriptionResponse, len(subscriptions))
	for i, sub := range subscriptions {
		response[i] = subscriptionResponse{
			Subscription:        sub,
			TriggeredContainers: sub.TriggeredContainersCount(),
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
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

func (h *handler) replaceSubscription(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var sub notification.Subscription
	if err := json.NewDecoder(r.Body).Decode(&sub); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	sub.ID = id

	if err := h.hostService.ReplaceSubscription(&sub); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sub)
}

func (h *handler) updateSubscription(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var updates map[string]any
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.hostService.UpdateSubscription(id, updates); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) listDispatchers(w http.ResponseWriter, r *http.Request) {
	dispatchers := h.hostService.Dispatchers()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dispatchers)
}

type createDispatcherRequest struct {
	Name string `json:"name"`
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
		d = dispatcher.NewWebhookDispatcher(req.Name, req.URL)
	default:
		http.Error(w, "unknown dispatcher type", http.StatusBadRequest)
		return
	}

	id := h.hostService.AddDispatcher(d)

	response := notification.DispatcherConfig{
		ID:   id,
		Name: req.Name,
		Type: req.Type,
		URL:  req.URL,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (h *handler) updateDispatcher(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

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
		d = dispatcher.NewWebhookDispatcher(req.Name, req.URL)
	default:
		http.Error(w, "unknown dispatcher type", http.StatusBadRequest)
		return
	}

	h.hostService.UpdateDispatcher(id, d)

	response := notification.DispatcherConfig{
		ID:   id,
		Name: req.Name,
		Type: req.Type,
		URL:  req.URL,
	}

	w.Header().Set("Content-Type", "application/json")
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

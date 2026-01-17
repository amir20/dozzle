package web

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/notification"
	"github.com/amir20/dozzle/internal/notification/dispatcher"
	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
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

type previewExpressionRequest struct {
	ContainerExpression string `json:"containerExpression"`
	LogExpression       string `json:"logExpression"`
}

type previewExpressionResponse struct {
	ContainerError    string                `json:"containerError,omitempty"`
	LogError          string                `json:"logError,omitempty"`
	MatchedContainers []container.Container `json:"matchedContainers,omitempty"`
	MatchedLogs       []*container.LogEvent `json:"matchedLogs,omitempty"`
	TotalLogs         int                   `json:"totalLogs"`
}

func (h *handler) previewExpression(w http.ResponseWriter, r *http.Request) {
	var req previewExpressionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response := previewExpressionResponse{}

	// Compile and test container expression
	var containerProgram *vm.Program
	if req.ContainerExpression != "" {
		program, err := expr.Compile(req.ContainerExpression, expr.Env(notification.Container{}))
		if err != nil {
			response.ContainerError = err.Error()
		} else {
			containerProgram = program
		}
	}

	// Compile and test log expression
	var logProgram *vm.Program
	if req.LogExpression != "" {
		program, err := expr.Compile(req.LogExpression, expr.Env(notification.Log{}))
		if err != nil {
			response.LogError = err.Error()
		} else {
			logProgram = program
		}
	}

	// If container expression is valid, find matching containers
	if containerProgram != nil {
		containers, _ := h.hostService.ListAllContainers(container.ContainerLabels{})
		for _, c := range containers {
			nc := notification.FromContainerModel(c)
			result, err := expr.Run(containerProgram, nc)
			if err != nil {
				continue
			}
			if match, ok := result.(bool); ok && match {
				response.MatchedContainers = append(response.MatchedContainers, c)
			}
		}
	}

	// If log expression is valid and we have matching containers, fetch real logs
	if logProgram != nil && len(response.MatchedContainers) > 0 {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		const maxLogs = 10
		totalMatched := 0

		for _, c := range response.MatchedContainers {
			if len(response.MatchedLogs) >= maxLogs {
				break
			}

			containerService, err := h.hostService.FindContainer(c.Host, c.ID, container.ContainerLabels{})
			if err != nil {
				continue
			}

			// Fetch recent logs (last 5 minutes)
			from := time.Now().Add(-5 * time.Minute)
			to := time.Now()

			logChan, err := containerService.LogsBetweenDates(ctx, from, to, container.STDALL)
			if err != nil {
				continue
			}

			for logEvent := range logChan {
				if logEvent == nil {
					continue
				}

				// Convert to notification.Log for expression evaluation
				l := notification.FromLogEvent(*logEvent)
				result, err := expr.Run(logProgram, l)
				if err != nil {
					continue
				}

				if match, ok := result.(bool); ok && match {
					totalMatched++
					if len(response.MatchedLogs) < maxLogs {
						response.MatchedLogs = append(response.MatchedLogs, logEvent)
					}
				}
			}
		}

		response.TotalLogs = totalMatched
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

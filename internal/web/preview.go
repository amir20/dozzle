package web

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/notification"
	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
)

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

	// If container expression is valid, find matching running containers
	if containerProgram != nil {
		containers, _ := h.hostService.ListAllContainers(container.ContainerLabels{})
		for _, c := range containers {
			if c.State != "running" {
				continue
			}
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

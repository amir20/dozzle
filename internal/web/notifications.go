package web

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/amir20/dozzle/internal/cache"
	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/notification"
	"github.com/amir20/dozzle/internal/notification/dispatcher"
	"github.com/amir20/dozzle/internal/releases"
	"github.com/amir20/dozzle/types"
	"github.com/expr-lang/expr"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

// Response types for JSON serialization
type NotificationRuleResponse struct {
	ID                  int                 `json:"id"`
	Name                string              `json:"name"`
	Enabled             bool                `json:"enabled"`
	ContainerExpression string              `json:"containerExpression"`
	LogExpression       string              `json:"logExpression"`
	MetricExpression    string              `json:"metricExpression,omitempty"`
	Cooldown            int                 `json:"cooldown,omitempty"`
	TriggerCount        int64               `json:"triggerCount"`
	TriggeredContainers int                 `json:"triggeredContainers"`
	LastTriggeredAt     *time.Time          `json:"lastTriggeredAt"`
	Dispatcher          *DispatcherResponse `json:"dispatcher"`
}

type DispatcherResponse struct {
	ID        int        `json:"id"`
	Name      string     `json:"name"`
	Type      string     `json:"type"`
	URL       *string    `json:"url,omitempty"`
	Template  *string    `json:"template,omitempty"`
	Prefix    *string    `json:"prefix,omitempty"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
}

type NotificationRuleInput struct {
	Name                string `json:"name"`
	Enabled             bool   `json:"enabled"`
	DispatcherID        int    `json:"dispatcherId"`
	LogExpression       string `json:"logExpression"`
	ContainerExpression string `json:"containerExpression"`
	MetricExpression    string `json:"metricExpression,omitempty"`
	Cooldown            int    `json:"cooldown,omitempty"`
}

type NotificationRuleUpdateInput struct {
	Name                *string `json:"name,omitempty"`
	Enabled             *bool   `json:"enabled,omitempty"`
	DispatcherID        *int    `json:"dispatcherId,omitempty"`
	LogExpression       *string `json:"logExpression,omitempty"`
	ContainerExpression *string `json:"containerExpression,omitempty"`
	MetricExpression    *string `json:"metricExpression,omitempty"`
	Cooldown            *int    `json:"cooldown,omitempty"`
}

type DispatcherInput struct {
	Name     string  `json:"name"`
	Type     string  `json:"type"`
	URL      *string `json:"url,omitempty"`
	Template *string `json:"template,omitempty"`
}

type PreviewInput struct {
	ContainerExpression string  `json:"containerExpression"`
	LogExpression       *string `json:"logExpression,omitempty"`
	MetricExpression    *string `json:"metricExpression,omitempty"`
}

type PreviewResult struct {
	ContainerError    *string               `json:"containerError,omitempty"`
	LogError          *string               `json:"logError,omitempty"`
	MetricError       *string               `json:"metricError,omitempty"`
	MatchedContainers []container.Container `json:"matchedContainers"`
	MatchedLogs       []container.LogEvent  `json:"matchedLogs"`
	TotalLogs         int                   `json:"totalLogs"`
	MessageKeys       []string              `json:"messageKeys,omitempty"`
}

type TestWebhookInput struct {
	URL      string  `json:"url"`
	Template *string `json:"template,omitempty"`
}

type TestWebhookResult struct {
	Success    bool    `json:"success"`
	StatusCode *int    `json:"statusCode,omitempty"`
	Error      *string `json:"error,omitempty"`
}

// Helper functions
func subscriptionToResponse(sub *notification.Subscription, dispatchers []notification.DispatcherConfig) *NotificationRuleResponse {
	var lastTriggeredAt *time.Time
	if t := sub.LastTriggeredAt.Load(); t != nil && !t.IsZero() {
		lastTriggeredAt = t
	}

	var disp *DispatcherResponse
	for _, d := range dispatchers {
		if d.ID == sub.DispatcherID {
			disp = dispatcherConfigToResponse(&d)
			break
		}
	}

	return &NotificationRuleResponse{
		ID:                  sub.ID,
		Name:                sub.Name,
		Enabled:             sub.Enabled,
		Dispatcher:          disp,
		LogExpression:       sub.LogExpression,
		ContainerExpression: sub.ContainerExpression,
		MetricExpression:    sub.MetricExpression,
		Cooldown:            sub.Cooldown,
		TriggerCount:        sub.TriggerCount.Load(),
		LastTriggeredAt:     lastTriggeredAt,
		TriggeredContainers: sub.TriggeredContainersCount(),
	}
}

func dispatcherConfigToResponse(d *notification.DispatcherConfig) *DispatcherResponse {
	var url *string
	if d.URL != "" {
		url = &d.URL
	}
	var template *string
	if d.Template != "" {
		template = &d.Template
	}
	var prefix *string
	if d.Prefix != "" {
		prefix = &d.Prefix
	}
	return &DispatcherResponse{
		ID:        d.ID,
		Name:      d.Name,
		Type:      d.Type,
		URL:       url,
		Template:  template,
		Prefix:    prefix,
		ExpiresAt: d.ExpiresAt,
	}
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Error().Err(err).Msg("failed to encode JSON response")
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// Notification Rules handlers
func (h *handler) listNotificationRules(w http.ResponseWriter, r *http.Request) {
	subscriptions := h.hostService.Subscriptions()
	dispatchers := h.hostService.Dispatchers()
	rules := make([]*NotificationRuleResponse, len(subscriptions))
	for i, sub := range subscriptions {
		rules[i] = subscriptionToResponse(sub, dispatchers)
	}
	writeJSON(w, http.StatusOK, rules)
}

func (h *handler) getNotificationRule(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	dispatchers := h.hostService.Dispatchers()
	for _, sub := range h.hostService.Subscriptions() {
		if sub.ID == id {
			writeJSON(w, http.StatusOK, subscriptionToResponse(sub, dispatchers))
			return
		}
	}
	writeError(w, http.StatusNotFound, "notification rule not found")
}

func (h *handler) createNotificationRule(w http.ResponseWriter, r *http.Request) {
	var input NotificationRuleInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	sub := &notification.Subscription{
		Name:                input.Name,
		Enabled:             input.Enabled,
		DispatcherID:        input.DispatcherID,
		LogExpression:       input.LogExpression,
		ContainerExpression: input.ContainerExpression,
		MetricExpression:    input.MetricExpression,
		Cooldown:            input.Cooldown,
	}

	if err := h.hostService.AddSubscription(sub); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, subscriptionToResponse(sub, h.hostService.Dispatchers()))
}

func (h *handler) replaceNotificationRule(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var input NotificationRuleInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	sub := &notification.Subscription{
		ID:                  id,
		Name:                input.Name,
		Enabled:             input.Enabled,
		DispatcherID:        input.DispatcherID,
		LogExpression:       input.LogExpression,
		ContainerExpression: input.ContainerExpression,
		MetricExpression:    input.MetricExpression,
		Cooldown:            input.Cooldown,
	}

	if err := h.hostService.ReplaceSubscription(sub); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, subscriptionToResponse(sub, h.hostService.Dispatchers()))
}

func (h *handler) updateNotificationRule(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var input NotificationRuleUpdateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	updates := make(map[string]any)
	if input.Name != nil {
		updates["name"] = *input.Name
	}
	if input.Enabled != nil {
		updates["enabled"] = *input.Enabled
	}
	if input.DispatcherID != nil {
		updates["dispatcherId"] = *input.DispatcherID
	}
	if input.LogExpression != nil {
		updates["logExpression"] = *input.LogExpression
	}
	if input.ContainerExpression != nil {
		updates["containerExpression"] = *input.ContainerExpression
	}
	if input.MetricExpression != nil {
		updates["metricExpression"] = *input.MetricExpression
	}
	if input.Cooldown != nil {
		updates["cooldown"] = *input.Cooldown
	}

	if err := h.hostService.UpdateSubscription(id, updates); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Fetch the updated subscription
	dispatchers := h.hostService.Dispatchers()
	for _, sub := range h.hostService.Subscriptions() {
		if sub.ID == id {
			writeJSON(w, http.StatusOK, subscriptionToResponse(sub, dispatchers))
			return
		}
	}

	writeError(w, http.StatusNotFound, "notification rule not found")
}

func (h *handler) deleteNotificationRule(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	h.hostService.RemoveSubscription(id)
	w.WriteHeader(http.StatusNoContent)
}

// Dispatcher handlers
func (h *handler) listDispatchers(w http.ResponseWriter, r *http.Request) {
	dispatchers := h.hostService.Dispatchers()
	result := make([]*DispatcherResponse, len(dispatchers))
	for i, d := range dispatchers {
		result[i] = dispatcherConfigToResponse(&d)
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *handler) getDispatcher(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	for _, d := range h.hostService.Dispatchers() {
		if d.ID == id {
			writeJSON(w, http.StatusOK, dispatcherConfigToResponse(&d))
			return
		}
	}
	writeError(w, http.StatusNotFound, "dispatcher not found")
}

func (h *handler) createDispatcher(w http.ResponseWriter, r *http.Request) {
	var input DispatcherInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var d dispatcher.Dispatcher
	switch input.Type {
	case "webhook":
		url := ""
		if input.URL != nil {
			url = *input.URL
		}
		templateStr := ""
		if input.Template != nil {
			templateStr = *input.Template
		}
		webhook, err := dispatcher.NewWebhookDispatcher(input.Name, url, templateStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		d = webhook
	default:
		writeError(w, http.StatusBadRequest, "unknown dispatcher type")
		return
	}

	id := h.hostService.AddDispatcher(d)

	writeJSON(w, http.StatusCreated, &DispatcherResponse{
		ID:       id,
		Name:     input.Name,
		Type:     input.Type,
		URL:      input.URL,
		Template: input.Template,
	})
}

func (h *handler) updateDispatcher(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var input DispatcherInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var d dispatcher.Dispatcher
	switch input.Type {
	case "webhook":
		url := ""
		if input.URL != nil {
			url = *input.URL
		}
		templateStr := ""
		if input.Template != nil {
			templateStr = *input.Template
		}
		webhook, err := dispatcher.NewWebhookDispatcher(input.Name, url, templateStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		d = webhook
	default:
		writeError(w, http.StatusBadRequest, "unknown dispatcher type")
		return
	}

	h.hostService.UpdateDispatcher(id, d)

	writeJSON(w, http.StatusOK, &DispatcherResponse{
		ID:       id,
		Name:     input.Name,
		Type:     input.Type,
		URL:      input.URL,
		Template: input.Template,
	})
}

func (h *handler) deleteDispatcher(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	h.hostService.RemoveDispatcher(id)
	w.WriteHeader(http.StatusNoContent)
}

// Preview and test handlers
func (h *handler) previewExpression(w http.ResponseWriter, r *http.Request) {
	var input PreviewInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result := &PreviewResult{
		MatchedContainers: []container.Container{},
		MatchedLogs:       []container.LogEvent{},
	}

	sub := &notification.Subscription{
		ContainerExpression: input.ContainerExpression,
	}
	if input.LogExpression != nil {
		sub.LogExpression = *input.LogExpression
	}
	if input.MetricExpression != nil {
		sub.MetricExpression = *input.MetricExpression
	}

	// Compile container expression
	if sub.ContainerExpression != "" {
		program, err := expr.Compile(sub.ContainerExpression, expr.Env(types.NotificationContainer{}))
		if err != nil {
			errStr := err.Error()
			result.ContainerError = &errStr
		} else {
			sub.ContainerProgram = program
		}
	}

	// Compile log expression
	if sub.LogExpression != "" {
		program, err := expr.Compile(sub.LogExpression, expr.Env(types.NotificationLog{}))
		if err != nil {
			errStr := err.Error()
			result.LogError = &errStr
		} else {
			sub.LogProgram = program
		}
	}

	// Compile metric expression
	if sub.MetricExpression != "" {
		_, err := expr.Compile(sub.MetricExpression, expr.Env(types.NotificationStat{}))
		if err != nil {
			errStr := err.Error()
			result.MetricError = &errStr
		}
	}

	// Find matching running containers
	if sub.ContainerProgram != nil {
		containers, _ := h.hostService.ListAllContainers(container.ContainerLabels{})
		for _, c := range containers {
			if c.State != "running" {
				continue
			}
			// Pass empty host for matching - host fields aren't used in container expressions
			nc := notification.FromContainerModel(c, container.Host{})
			if sub.MatchesContainer(nc) {
				result.MatchedContainers = append(result.MatchedContainers, c)
			}
		}
	}

	// Fetch real logs from matched containers
	if len(result.MatchedContainers) > 0 {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		const maxLogs = 10
		totalMatched := 0
		keySet := make(map[string]struct{})

		for _, c := range result.MatchedContainers {
			containerService, err := h.hostService.FindContainer(c.Host, c.ID, container.ContainerLabels{})
			if err != nil {
				continue
			}

			from := time.Now().Add(-2 * time.Hour)
			to := time.Now()

			logChan, err := containerService.LogsBetweenDates(ctx, from, to, container.STDALL)
			if err != nil {
				continue
			}

			for logEvent := range logChan {
				if logEvent == nil {
					continue
				}

				// Collect message keys from structured logs
				switch m := logEvent.Message.(type) {
				case *orderedmap.OrderedMap[string, any]:
					for pair := m.Oldest(); pair != nil; pair = pair.Next() {
						keySet[pair.Key] = struct{}{}
					}
				case *orderedmap.OrderedMap[string, string]:
					for pair := m.Oldest(); pair != nil; pair = pair.Next() {
						keySet[pair.Key] = struct{}{}
					}
				}

				if sub.LogProgram != nil {
					notificationLog := notification.FromLogEvent(*logEvent)
					if sub.MatchesLog(notificationLog) {
						totalMatched++
						if len(result.MatchedLogs) < maxLogs {
							result.MatchedLogs = append(result.MatchedLogs, *logEvent)
						}
					}
				}
			}
		}

		result.TotalLogs = totalMatched

		if len(keySet) > 0 {
			keys := make([]string, 0, len(keySet))
			for k := range keySet {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			result.MessageKeys = keys
		}
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *handler) testWebhook(w http.ResponseWriter, r *http.Request) {
	var input TestWebhookInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	templateStr := ""
	if input.Template != nil {
		templateStr = *input.Template
	}

	webhook, err := dispatcher.NewWebhookDispatcher("test", input.URL, templateStr)
	if err != nil {
		errStr := err.Error()
		writeJSON(w, http.StatusOK, &TestWebhookResult{
			Success: false,
			Error:   &errStr,
		})
		return
	}

	mockNotification := types.Notification{
		ID:        "test-notification",
		Detail:    "This is a test log message from Dozzle",
		Timestamp: time.Now(),
		Container: types.NotificationContainer{
			ID:       "abc123",
			Name:     "test-container",
			Image:    "nginx:latest",
			State:    "running",
			Health:   "healthy",
			HostID:   "localhost",
			HostName: "localhost",
			Labels:   map[string]string{"env": "test"},
		},
		Log: &types.NotificationLog{
			ID:        1,
			Message:   "This is a test log message from Dozzle",
			Timestamp: time.Now().UnixMilli(),
			Level:     "info",
			Stream:    "stdout",
			Type:      "simple",
		},
		Stat: &types.NotificationStat{},
	}

	result := webhook.SendTest(r.Context(), mockNotification)

	var statusCode *int
	if result.StatusCode > 0 {
		statusCode = &result.StatusCode
	}

	var errStr *string
	if result.Error != "" {
		errStr = &result.Error
	}

	writeJSON(w, http.StatusOK, &TestWebhookResult{
		Success:    result.Success,
		StatusCode: statusCode,
		Error:      errStr,
	})
}

// Releases handler
var releasesCache *cache.Cache[[]releases.Release]

func (h *handler) getReleases(w http.ResponseWriter, r *http.Request) {
	if releasesCache == nil {
		releasesCache = cache.New(func() ([]releases.Release, error) {
			return releases.Fetch(h.config.Version)
		}, time.Hour)
	}

	result, err := releasesCache.Get()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

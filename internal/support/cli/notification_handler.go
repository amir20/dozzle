package cli

import (
	"github.com/amir20/dozzle/internal/agent"
	"github.com/amir20/dozzle/internal/notification"
)

// AgentNotificationHandler adapts the notification manager to the agent's NotificationConfigHandler interface
type AgentNotificationHandler struct {
	manager *notification.Manager
}

// NewAgentNotificationHandler creates a new handler that wraps a notification manager
func NewAgentNotificationHandler(manager *notification.Manager) *AgentNotificationHandler {
	return &AgentNotificationHandler{manager: manager}
}

// HandleNotificationConfig implements agent.NotificationConfigHandler
func (h *AgentNotificationHandler) HandleNotificationConfig(subscriptions []agent.SubscriptionConfig, dispatchers []agent.DispatcherConfig) error {
	// Convert agent types to notification types
	notifSubs := make([]*notification.Subscription, len(subscriptions))
	for i, sub := range subscriptions {
		notifSubs[i] = &notification.Subscription{
			ID:                  sub.ID,
			Name:                sub.Name,
			Enabled:             sub.Enabled,
			DispatcherID:        sub.DispatcherID,
			LogExpression:       sub.LogExpression,
			ContainerExpression: sub.ContainerExpression,
		}
	}

	notifDispatchers := make([]notification.DispatcherConfig, len(dispatchers))
	for i, d := range dispatchers {
		notifDispatchers[i] = notification.DispatcherConfig{
			ID:       d.ID,
			Name:     d.Name,
			Type:     d.Type,
			URL:      d.URL,
			Template: d.Template,
		}
	}

	return h.manager.ReplaceState(notifSubs, notifDispatchers)
}

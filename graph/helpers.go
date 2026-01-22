package graph

import (
	"time"

	"github.com/amir20/dozzle/graph/model"
	"github.com/amir20/dozzle/internal/notification"
	"github.com/amir20/dozzle/types"
)

func subscriptionToNotificationRule(sub *notification.Subscription, dispatchers []types.DispatcherConfig) *model.NotificationRule {
	var lastTriggeredAt *time.Time
	if t := sub.LastTriggeredAt.Load(); t != nil && !t.IsZero() {
		lastTriggeredAt = t
	}

	// Find the dispatcher
	var dispatcher *model.Dispatcher
	for _, d := range dispatchers {
		if d.ID == sub.DispatcherID {
			dispatcher = dispatcherConfigToDispatcher(&d)
			break
		}
	}

	return &model.NotificationRule{
		ID:                  int32(sub.ID),
		Name:                sub.Name,
		Enabled:             sub.Enabled,
		Dispatcher:          dispatcher,
		LogExpression:       sub.LogExpression,
		ContainerExpression: sub.ContainerExpression,
		TriggerCount:        int(sub.TriggerCount.Load()),
		LastTriggeredAt:     lastTriggeredAt,
		TriggeredContainers: int32(sub.TriggeredContainersCount()),
	}
}

func dispatcherConfigToDispatcher(d *types.DispatcherConfig) *model.Dispatcher {
	var url *string
	if d.URL != "" {
		url = &d.URL
	}
	var template *string
	if d.Template != "" {
		template = &d.Template
	}
	return &model.Dispatcher{
		ID:       int32(d.ID),
		Name:     d.Name,
		Type:     d.Type,
		URL:      url,
		Template: template,
	}
}

// Error is a simple error type for GraphQL errors
type Error struct {
	Message string
}

func (e *Error) Error() string {
	return e.Message
}

package cloud

import (
	"errors"
	"fmt"
	"strings"

	"github.com/amir20/dozzle/internal/notification"
	pb "github.com/amir20/dozzle/proto/cloud"
)

// cloudDispatcherID is the reserved ID for the Dozzle Cloud dispatcher. All
// alerts created via cloud tools route here so the user receives them through
// their configured cloud channels (Telegram, Discord, etc.).
const cloudDispatcherID = 0

// errNotificationsNotConfigured is returned when notification tools are
// invoked in a mode without a notification manager (e.g. k8s).
var errNotificationsNotConfigured = errors.New("notifications are not configured on this host")

func executeListNotifications(deps ToolDeps) (*pb.CallToolResponse, error) {
	if deps.NotificationService == nil {
		return nil, errNotificationsNotConfigured
	}

	subs := deps.NotificationService.Subscriptions()

	var sb strings.Builder
	fmt.Fprintf(&sb, "Subscriptions (%d):\n", len(subs))
	if len(subs) == 0 {
		sb.WriteString("  (none)\n")
	}
	for _, s := range subs {
		fmt.Fprintf(&sb, "  - #%d %q [%s, enabled=%t]\n", s.ID, s.Name, subscriptionKind(s), s.Enabled)
		if s.ContainerExpression != "" {
			fmt.Fprintf(&sb, "      container: %s\n", s.ContainerExpression)
		}
		if s.LogExpression != "" {
			fmt.Fprintf(&sb, "      log:       %s\n", s.LogExpression)
		}
		if s.MetricExpression != "" {
			fmt.Fprintf(&sb, "      metric:    %s (cooldown=%ds, window=%ds)\n", s.MetricExpression, s.GetCooldownSeconds(), s.GetSampleWindowSeconds())
		}
		if s.EventExpression != "" {
			fmt.Fprintf(&sb, "      event:     %s\n", s.EventExpression)
		}
	}

	return notificationResponse(sb.String()), nil
}

func subscriptionKind(s *notification.Subscription) string {
	var kinds []string
	if s.LogExpression != "" {
		kinds = append(kinds, "log")
	}
	if s.MetricExpression != "" {
		kinds = append(kinds, "metric")
	}
	if s.EventExpression != "" {
		kinds = append(kinds, "event")
	}
	if len(kinds) == 0 {
		return "empty"
	}
	return strings.Join(kinds, "+")
}

type createLogNotificationArgs struct {
	Name                string `json:"name"`
	ContainerExpression string `json:"container_expression"`
	LogExpression       string `json:"log_expression"`
}

func executeCreateLogNotification(argsJSON string, deps ToolDeps) (*pb.CallToolResponse, error) {
	if deps.NotificationService == nil {
		return nil, errNotificationsNotConfigured
	}

	args, err := parseArgs[createLogNotificationArgs](argsJSON)
	if err != nil {
		return nil, err
	}
	if err := requireNonEmpty(map[string]string{
		"name":                 args.Name,
		"container_expression": args.ContainerExpression,
		"log_expression":       args.LogExpression,
	}); err != nil {
		return nil, err
	}

	return addSubscriptionResponse(deps.NotificationService, &notification.Subscription{
		Name:                args.Name,
		DispatcherID:        cloudDispatcherID,
		ContainerExpression: args.ContainerExpression,
		LogExpression:       args.LogExpression,
	})
}

type createMetricNotificationArgs struct {
	Name                string `json:"name"`
	ContainerExpression string `json:"container_expression"`
	MetricExpression    string `json:"metric_expression"`
	CooldownSeconds     int    `json:"cooldown_seconds"`
	SampleWindowSeconds int    `json:"sample_window_seconds"`
}

func executeCreateMetricNotification(argsJSON string, deps ToolDeps) (*pb.CallToolResponse, error) {
	if deps.NotificationService == nil {
		return nil, errNotificationsNotConfigured
	}

	args, err := parseArgs[createMetricNotificationArgs](argsJSON)
	if err != nil {
		return nil, err
	}
	if err := requireNonEmpty(map[string]string{
		"name":                 args.Name,
		"container_expression": args.ContainerExpression,
		"metric_expression":    args.MetricExpression,
	}); err != nil {
		return nil, err
	}

	return addSubscriptionResponse(deps.NotificationService, &notification.Subscription{
		Name:                args.Name,
		DispatcherID:        cloudDispatcherID,
		ContainerExpression: args.ContainerExpression,
		MetricExpression:    args.MetricExpression,
		Cooldown:            args.CooldownSeconds,
		SampleWindow:        args.SampleWindowSeconds,
	})
}

type createEventNotificationArgs struct {
	Name                string `json:"name"`
	ContainerExpression string `json:"container_expression"`
	EventExpression     string `json:"event_expression"`
}

func executeCreateEventNotification(argsJSON string, deps ToolDeps) (*pb.CallToolResponse, error) {
	if deps.NotificationService == nil {
		return nil, errNotificationsNotConfigured
	}

	args, err := parseArgs[createEventNotificationArgs](argsJSON)
	if err != nil {
		return nil, err
	}
	if err := requireNonEmpty(map[string]string{
		"name":                 args.Name,
		"container_expression": args.ContainerExpression,
		"event_expression":     args.EventExpression,
	}); err != nil {
		return nil, err
	}

	return addSubscriptionResponse(deps.NotificationService, &notification.Subscription{
		Name:                args.Name,
		DispatcherID:        cloudDispatcherID,
		ContainerExpression: args.ContainerExpression,
		EventExpression:     args.EventExpression,
	})
}

func requireNonEmpty(fields map[string]string) error {
	for name, v := range fields {
		if v == "" {
			return fmt.Errorf("%s is required", name)
		}
	}
	return nil
}

func notificationResponse(message string) *pb.CallToolResponse {
	return &pb.CallToolResponse{
		Success: true,
		Result: &pb.CallToolResponse_Notification{Notification: &pb.NotificationResult{
			Success: true,
			Message: message,
		}},
	}
}

func addSubscriptionResponse(svc NotificationService, sub *notification.Subscription) (*pb.CallToolResponse, error) {
	if err := svc.AddSubscription(sub); err != nil {
		return nil, fmt.Errorf("creating subscription: %w", err)
	}

	return notificationResponse(fmt.Sprintf("Created alert %q (id=%d).", sub.Name, sub.ID)), nil
}

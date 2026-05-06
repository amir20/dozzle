package cloud

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/notification"
	container_support "github.com/amir20/dozzle/internal/support/container"
	pb "github.com/amir20/dozzle/proto/cloud"
	"github.com/rs/zerolog/log"
)

// ToolHostService is the subset of HostService needed by tool execution.
type ToolHostService interface {
	ListAllContainers(labels container.ContainerLabels) ([]container.Container, []error)
	FindContainer(host string, id string, labels container.ContainerLabels) (*container_support.ContainerService, error)
	Hosts() []container.Host
}

// Tool names. Declared as consts so dispatch and AvailableTools stay in sync
// — a mismatch here would otherwise surface only as a runtime "unknown tool".
const (
	toolListHosts                = "list_hosts"
	toolFindContainers           = "find_containers"
	toolListRunningContainers    = "list_running_containers"
	toolListAllContainers        = "list_all_containers"
	toolGetRunningContainerStats = "get_running_container_stats"
	toolFetchContainerLogs       = "fetch_container_logs"
	toolStreamLogs               = "stream_logs"
	toolListNotifications        = "list_notifications"
	toolInspectContainer         = "inspect_container"
	toolStartContainer           = "start_container"
	toolStopContainer            = "stop_container"
	toolRestartContainer         = "restart_container"
	toolRemoveContainer          = "remove_container"
	toolUpdateContainer          = "update_container"
	toolCreateLogNotification    = "create_log_notification"
	toolCreateMetricNotification = "create_metric_notification"
	toolCreateEventNotification  = "create_event_notification"
)

type paramProperty struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

type paramSchema struct {
	Type                 string                   `json:"type"`
	Properties           map[string]paramProperty `json:"properties"`
	Required             []string                 `json:"required,omitempty"`
	AdditionalProperties *bool                    `json:"additionalProperties,omitempty"`
}

func mustSchema(s paramSchema) string {
	b, err := json.Marshal(s)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal schema: %v", err))
	}
	return string(b)
}

var (
	noParams = mustSchema(paramSchema{
		Type:       "object",
		Properties: map[string]paramProperty{},
	})

	containerIDParam = paramProperty{Type: "string", Description: "The container ID (from find_containers)"}
	hostIDParam      = paramProperty{Type: "string", Description: "The host ID where the container is running (from find_containers)"}
	boolFalse        = false

	targetedParams = mustSchema(paramSchema{
		Type: "object",
		Properties: map[string]paramProperty{
			"container_id": containerIDParam,
			"host_id":      hostIDParam,
		},
		Required:             []string{"container_id", "host_id"},
		AdditionalProperties: &boolFalse,
	})

	findContainerParams = mustSchema(paramSchema{
		Type: "object",
		Properties: map[string]paramProperty{
			"name":   {Type: "string", Description: "Optional container name to search for (partial match supported)"},
			"image":  {Type: "string", Description: "Optional image name to filter by (partial match supported)"},
			"state":  {Type: "string", Description: "Optional state filter (e.g. running, exited, created)"},
			"health": {Type: "string", Description: "Optional health status filter (e.g. healthy, unhealthy, none)"},
		},
	})

	instanceIDParam = paramProperty{
		Type:        "string",
		Description: "The Dozzle instance to target. Get this from list_dozzle_instances. Alerts are scoped to a whole Dozzle instance, not a single Docker host.",
	}

	listNotificationsParams = mustSchema(paramSchema{
		Type: "object",
		Properties: map[string]paramProperty{
			"instance_id": instanceIDParam,
		},
		Required:             []string{"instance_id"},
		AdditionalProperties: &boolFalse,
	})

	containerExpressionParam = paramProperty{
		Type: "string",
		Description: `Required. expr-lang expression selecting which containers trigger the alert. Use "true" to match every container — this is the right default whenever the user asks for an alert without naming a specific target ("all my containers", "any error", "logs of type error" with no target named). Only write a filter when the user names containers or gives a pattern. Available fields when filtering: name, id, image, state, health, host, labels.
Examples: true (match every container — default for unscoped asks); name contains "nginx"; state == "running"; image matches "redis.*"; name contains "api" && health == "healthy".`,
	}

	createLogNotificationParams = mustSchema(paramSchema{
		Type: "object",
		Properties: map[string]paramProperty{
			"name":                 {Type: "string", Description: "Human-readable alert name shown in the UI."},
			"instance_id":          instanceIDParam,
			"container_expression": containerExpressionParam,
			"log_expression": {
				Type: "string",
				Description: `Required. expr-lang expression matched against each log line. Fields: message (string), level (error|warn|info|debug|trace), stream (stdout|stderr), type, timestamp, id. For JSON logs, fields on the parsed object are accessible as message.<key>.
Examples: level == "error"; message contains "timeout"; level == "error" && message contains "database"; stream == "stderr".`,
			},
		},
		Required:             []string{"name", "instance_id", "container_expression", "log_expression"},
		AdditionalProperties: &boolFalse,
	})

	createMetricNotificationParams = mustSchema(paramSchema{
		Type: "object",
		Properties: map[string]paramProperty{
			"name":                 {Type: "string", Description: "Human-readable alert name shown in the UI."},
			"instance_id":          instanceIDParam,
			"container_expression": containerExpressionParam,
			"metric_expression": {
				Type: "string",
				Description: `Required. expr-lang boolean expression evaluated against container stats. Fields: cpu (percent 0-100), memory (percent 0-100), memoryUsage (bytes).
Examples: cpu > 80; memory > 90; cpu > 80 || memory > 95.`,
			},
			"cooldown_seconds":      {Type: "integer", Description: "Optional. Minimum seconds between repeat alerts for the same container. Defaults to 300."},
			"sample_window_seconds": {Type: "integer", Description: "Optional. Seconds of samples required before triggering, to avoid transient spikes. Defaults to 15."},
		},
		Required:             []string{"name", "instance_id", "container_expression", "metric_expression"},
		AdditionalProperties: &boolFalse,
	})

	createEventNotificationParams = mustSchema(paramSchema{
		Type: "object",
		Properties: map[string]paramProperty{
			"name":                 {Type: "string", Description: "Human-readable alert name shown in the UI."},
			"instance_id":          instanceIDParam,
			"container_expression": containerExpressionParam,
			"event_expression": {
				Type: "string",
				Description: `Required. expr-lang expression evaluated against container lifecycle events. Fields: name (start|stop|die|restart|destroy|kill|oom|health_status|...), attributes (map of event-specific fields).
Examples: name == "die"; name == "oom"; name in ["die", "oom", "kill"]; name == "health_status" && attributes.healthStatus == "unhealthy".`,
			},
		},
		Required:             []string{"name", "instance_id", "container_expression", "event_expression"},
		AdditionalProperties: &boolFalse,
	})

	fetchLogsParams = mustSchema(paramSchema{
		Type: "object",
		Properties: map[string]paramProperty{
			"container_id": containerIDParam,
			"host_id":      hostIDParam,
			"start":        {Type: "string", Description: "Optional ISO 8601 start time for log range"},
			"end":          {Type: "string", Description: "Optional ISO 8601 end time for log range"},
			"level":        {Type: "string", Description: "Optional log level filter (e.g. error, warn, info)"},
			"query":        {Type: "string", Description: "Optional text search query (case-insensitive substring match)"},
			"regex":        {Type: "string", Description: "Optional regex pattern to match against log messages"},
		},
		Required:             []string{"container_id", "host_id"},
		AdditionalProperties: &boolFalse,
	})

	streamLogsParams = mustSchema(paramSchema{
		Type: "object",
		Properties: map[string]paramProperty{
			"container_id": containerIDParam,
			"host_id":      hostIDParam,
			"level":        {Type: "string", Description: "Optional log level filter (e.g. error, warn, info)"},
			"query":        {Type: "string", Description: "Optional text search query (case-insensitive substring match)"},
			"regex":        {Type: "string", Description: "Optional regex pattern to match against log messages"},
		},
		Required:             []string{"container_id", "host_id"},
		AdditionalProperties: &boolFalse,
	})
)

// AvailableTools returns the list of tool definitions based on configuration.
func AvailableTools(enableActions bool) []*pb.ToolDefinition {
	tools := []*pb.ToolDefinition{
		{
			Name:           toolListHosts,
			Description:    "List all Docker hosts connected to Dozzle with their name, CPU cores, total memory, Docker version, and availability status.",
			ParametersJson: noParams,
			Scope:          pb.ToolScope_TOOL_SCOPE_INSTANCE,
			ReadOnly:       true,
		},
		{
			Name:           toolFindContainers,
			Description:    "Search for Docker containers by name, state, or health status. All parameters are optional. Returns container ID, name, image, state, health, and host. Use this before start/stop/restart actions to get the container ID and host.",
			ParametersJson: findContainerParams,
			Scope:          pb.ToolScope_TOOL_SCOPE_INSTANCE,
			ReadOnly:       true,
		},
		{
			Name:           toolListRunningContainers,
			Description:    "List all currently running Docker containers. Use find_containers instead if you need to filter by name or health status.",
			ParametersJson: noParams,
			Scope:          pb.ToolScope_TOOL_SCOPE_INSTANCE,
			ReadOnly:       true,
		},
		{
			Name:           toolListAllContainers,
			Description:    "List all Docker containers including stopped, exited, and previously run containers.",
			ParametersJson: noParams,
			Scope:          pb.ToolScope_TOOL_SCOPE_INSTANCE,
			ReadOnly:       true,
		},
		{
			Name:           toolGetRunningContainerStats,
			Description:    "Get real-time CPU, memory, and network usage statistics for all currently running Docker containers. Returns current percentages, peak values over the last 5 minutes, and network rx/tx totals plus bytes transferred in the last 5 minutes.",
			ParametersJson: noParams,
			Scope:          pb.ToolScope_TOOL_SCOPE_INSTANCE,
			ReadOnly:       true,
		},
		{
			Name:           toolFetchContainerLogs,
			Description:    "Fetch raw logs from a running Docker container. Requires container_id and host from find_containers. Optionally filter by time range, log level, text search, or regex pattern. Returns up to 100 matching log lines.",
			ParametersJson: fetchLogsParams,
			Scope:          pb.ToolScope_TOOL_SCOPE_CONTAINER,
			ReadOnly:       true,
		},
		{
			Name:           toolStreamLogs,
			Description:    "Stream live logs from a running Docker container in real time. Requires container_id and host_id from find_containers. Optionally filter by log level, text search, or regex pattern. Streams continuously until cancelled.",
			ParametersJson: streamLogsParams,
			Scope:          pb.ToolScope_TOOL_SCOPE_CONTAINER,
			ReadOnly:       true,
		},
		{
			Name:           toolListNotifications,
			Description:    "List configured alert subscriptions on a Dozzle host. Use this to check whether an alert already exists before creating a new one with create_log_notification, create_metric_notification, or create_event_notification.",
			ParametersJson: listNotificationsParams,
			Scope:          pb.ToolScope_TOOL_SCOPE_INSTANCE,
			ReadOnly:       true,
		},
		{
			Name:           toolInspectContainer,
			Description:    "Get detailed configuration of a Docker container including environment variables, port mappings, mounts, restart policy, network mode, labels, and resource limits.",
			ParametersJson: targetedParams,
			Scope:          pb.ToolScope_TOOL_SCOPE_CONTAINER,
			ReadOnly:       true,
		},
	}

	if enableActions {
		tools = append(tools,
			&pb.ToolDefinition{
				Name:           toolStartContainer,
				Description:    "Start a stopped Docker container",
				ParametersJson: targetedParams,
				Scope:          pb.ToolScope_TOOL_SCOPE_CONTAINER,
			},
			&pb.ToolDefinition{
				Name:           toolStopContainer,
				Description:    "Stop a running Docker container",
				ParametersJson: targetedParams,
				Scope:          pb.ToolScope_TOOL_SCOPE_CONTAINER,
			},
			&pb.ToolDefinition{
				Name:           toolRestartContainer,
				Description:    "Restart a Docker container",
				ParametersJson: targetedParams,
				Scope:          pb.ToolScope_TOOL_SCOPE_CONTAINER,
			},
			&pb.ToolDefinition{
				Name:           toolRemoveContainer,
				Description:    "Remove a Docker container. The container must be stopped first — call stop_container if it is still running. Confirm with the user before removing, since the container is gone permanently.",
				ParametersJson: targetedParams,
				Scope:          pb.ToolScope_TOOL_SCOPE_CONTAINER,
			},
			&pb.ToolDefinition{
				Name:           toolUpdateContainer,
				Description:    "Update a Docker container by pulling the latest version of its image and recreating it with the same configuration. If the image is already up to date, no recreation occurs. For swarm service containers, updates the service instead.",
				ParametersJson: targetedParams,
				Scope:          pb.ToolScope_TOOL_SCOPE_CONTAINER,
			},
			&pb.ToolDefinition{
				Name:           toolCreateLogNotification,
				Description:    "Create an alert that fires when a container log line matches a filter. Requires a container_expression selecting which containers to watch and a log_expression matched against each log line. Alerts are delivered through the user's Dozzle Cloud channels.",
				ParametersJson: createLogNotificationParams,
				Scope:          pb.ToolScope_TOOL_SCOPE_INSTANCE,
			},
			&pb.ToolDefinition{
				Name:           toolCreateMetricNotification,
				Description:    "Create an alert that fires when container CPU/memory usage crosses a threshold. Requires a container_expression selecting which containers to watch and a metric_expression evaluated against their stats. Alerts are delivered through the user's Dozzle Cloud channels.",
				ParametersJson: createMetricNotificationParams,
				Scope:          pb.ToolScope_TOOL_SCOPE_INSTANCE,
			},
			&pb.ToolDefinition{
				Name:           toolCreateEventNotification,
				Description:    "Create an alert that fires on container lifecycle events (start, stop, die, oom, health_status, etc.). Requires a container_expression selecting which containers to watch and an event_expression matched against each event. Alerts are delivered through the user's Dozzle Cloud channels.",
				ParametersJson: createEventNotificationParams,
				Scope:          pb.ToolScope_TOOL_SCOPE_INSTANCE,
			},
		)
	}

	return tools
}

// NotificationService is the subset of the notification manager exposed to
// cloud tools. Implementations must persist changes as appropriate (e.g. in
// server mode the MultiHostService wrapper saves to disk on each mutation).
type NotificationService interface {
	Subscriptions() []*notification.Subscription
	AddSubscription(sub *notification.Subscription) error
}

// ToolDeps bundles the dependencies required to execute cloud tool calls.
// NotificationService may be nil in modes without a notification manager
// (e.g., k8s); notification tools will then return a "not configured" error.
type ToolDeps struct {
	EnableActions       bool
	HostService         ToolHostService
	Labels              container.ContainerLabels
	NotificationService NotificationService
}

// ExecuteTool dispatches a tool call by name and returns a proto CallToolResponse.
func ExecuteTool(ctx context.Context, name string, argsJSON string, deps ToolDeps) *pb.CallToolResponse {
	resp, err := executeTool(ctx, name, argsJSON, deps)
	if err != nil {
		log.Warn().Err(err).Str("tool", name).Str("args", argsJSON).Msg("tool execution failed")
		return &pb.CallToolResponse{
			Success: false,
			Error:   err.Error(),
		}
	}
	return resp
}

// requiresActions lists tools gated behind --enable-actions.
var requiresActions = map[string]struct{}{
	toolStartContainer:           {},
	toolStopContainer:            {},
	toolRestartContainer:         {},
	toolRemoveContainer:          {},
	toolUpdateContainer:          {},
	toolCreateLogNotification:    {},
	toolCreateMetricNotification: {},
	toolCreateEventNotification:  {},
}

func executeTool(ctx context.Context, name string, argsJSON string, deps ToolDeps) (*pb.CallToolResponse, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	if _, gated := requiresActions[name]; gated && !deps.EnableActions {
		return nil, fmt.Errorf("container actions are not enabled")
	}

	switch name {
	case toolListHosts:
		return executeListHosts(deps)
	case toolFindContainers:
		return executeFindContainers(argsJSON, deps)
	case toolListRunningContainers:
		return executeListRunningContainers(deps)
	case toolListAllContainers:
		return executeListAllContainers(deps)
	case toolGetRunningContainerStats:
		return executeGetRunningContainerStats(deps)
	case toolFetchContainerLogs:
		return executeFetchContainerLogs(ctx, argsJSON, deps)
	case toolInspectContainer:
		return executeInspectContainer(argsJSON, deps)
	case toolListNotifications:
		return executeListNotifications(deps)
	case toolStartContainer, toolStopContainer, toolRestartContainer, toolRemoveContainer:
		return executeContainerAction(ctx, name, argsJSON, deps)
	case toolUpdateContainer:
		return executeUpdateContainer(ctx, argsJSON, deps)
	case toolCreateLogNotification:
		return executeCreateLogNotification(argsJSON, deps)
	case toolCreateMetricNotification:
		return executeCreateMetricNotification(argsJSON, deps)
	case toolCreateEventNotification:
		return executeCreateEventNotification(argsJSON, deps)
	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

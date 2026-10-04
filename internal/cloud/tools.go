package cloud

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/imagecheck"
	"github.com/amir20/dozzle/internal/notification"
	pb "github.com/amir20/dozzle/proto/cloud"
	"github.com/rs/zerolog/log"
)

// ToolHostService is the subset of HostService needed by tool execution.
type ToolHostService interface {
	ListAllContainers(labels container.ContainerLabels) ([]container.Container, []error)
	FindContainer(host string, id string, labels container.ContainerLabels) (*container.ContainerService, error)
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
	toolRollbackContainer        = "rollback_container"
	toolCreateLogNotification    = "create_log_notification"
	toolCreateMetricNotification = "create_metric_notification"
	toolCreateEventNotification  = "create_event_notification"
	toolRetroScan                = "retro_scan"
	toolCheckImageUpdates        = "check_image_updates"
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

	containerIDParam = paramProperty{Type: "string", Description: "Container name or ID; a name works directly, no lookup needed. Resolved by ID, then exact name, then a unique name substring. A name matching several containers resolves to the running (or most recently active) one, and the result names its siblings."}
	// writeContainerIDParam is the write tools' variant: they never pick between
	// matches, so the read-side resolution notes would only cost tokens.
	writeContainerIDParam = paramProperty{Type: "string", Description: "Container name or ID. Never guesses between live containers: an ambiguous name fails with the candidate list, then re-issue with the exact ID."}
	hostIDParam           = paramProperty{Type: "string", Description: "Host name or ID (from list_hosts or find_containers). Optional — omit it when the container name is unique across all hosts; supply it (name or ID) only to scope to a specific host when a name is ambiguous."}
	boolFalse             = false

	targetedParams = mustSchema(paramSchema{
		Type: "object",
		Properties: map[string]paramProperty{
			"container_id": containerIDParam,
			"host_id":      hostIDParam,
		},
		Required:             []string{"container_id"},
		AdditionalProperties: &boolFalse,
	})

	writeTargetedParams = mustSchema(paramSchema{
		Type: "object",
		Properties: map[string]paramProperty{
			"container_id": writeContainerIDParam,
			"host_id":      hostIDParam,
		},
		Required:             []string{"container_id"},
		AdditionalProperties: &boolFalse,
	})

	rollbackContainerParams = mustSchema(paramSchema{
		Type: "object",
		Properties: map[string]paramProperty{
			"container_id":         writeContainerIDParam,
			"host_id":              hostIDParam,
			"expected_from_digest": {Type: "string", Description: "The digest the container runs now (repo@sha256:... or sha256:...). Refused if it runs anything else."},
		},
		Required:             []string{"container_id", "expected_from_digest"},
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
		Description: "The Dozzle instance to target. Use an instance_id from the connected-instance list you were given (or list_dozzle_instances). Alerts are scoped to a whole Dozzle instance, not a single Docker host.",
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
		Description: `Required. expr-lang expression selecting which containers trigger the alert. Use "true" (every container) whenever the user names no specific target — that is the default. Filter only when they name containers or a pattern. Fields: name, id, image, state, health, host, labels.
Examples: true; name contains "nginx"; image matches "redis.*"; name contains "api" && health == "healthy".`,
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
		Required:             []string{"container_id"},
		AdditionalProperties: &boolFalse,
	})

	retroScanParams = mustSchema(paramSchema{
		Type: "object",
		Properties: map[string]paramProperty{
			"since":            {Type: "string", Description: "RFC3339 start of the window. Defaults to 24 hours ago; capped at 7 days."},
			"levels":           {Type: "string", Description: "Comma-separated levels to count and keep, e.g. \"error,fatal,warn\" (the default)."},
			"deadline_seconds": {Type: "integer", Description: "How long the scan may run (default 60, max 300). Containers not reached are reported unscanned."},
		},
		AdditionalProperties: &boolFalse,
	})

	checkImageUpdatesParams = mustSchema(paramSchema{
		Type: "object",
		Properties: map[string]paramProperty{
			"name":    {Type: "string", Description: "Optional container name to check (partial match supported). Omit to check every container."},
			"image":   {Type: "string", Description: "Optional image name to check (partial match supported)."},
			"refresh": {Type: "boolean", Description: "Optional. Ask the registries again instead of answering from the cache, which can be up to 6 hours old. Only when the user says they just pushed an image or doubts the answer."},
		},
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
		Required:             []string{"container_id"},
		AdditionalProperties: &boolFalse,
	})
)

// AvailableTools lists the tools deps' principal may actually invoke. Filtering
// here rather than only at dispatch means the model never proposes something it
// will then be refused for.
func AvailableTools(deps ToolDeps) []*pb.ToolDefinition {
	enableActions, p := deps.EnableActions, deps.Principal
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
			Description:    "List or search Docker containers. With no filters it returns EVERY container, including stopped and exited ones; filter by name, image, state, or health to narrow. Returns ID, name, image, state, health, host, and start time. Container-scoped tools accept a name directly, so you don't need this just to get an ID — use it for inventory, state, or to disambiguate a name.",
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
			Description:    "Fetch raw logs from a running Docker container. Identify the container by name or ID via container_id; host_id is optional unless the name is ambiguous. Optionally filter by time range, log level, text search, or regex pattern. Returns up to 100 matching log lines.",
			ParametersJson: fetchLogsParams,
			Scope:          pb.ToolScope_TOOL_SCOPE_CONTAINER,
			ReadOnly:       true,
		},
		{
			Name:           toolStreamLogs,
			Description:    "Stream live logs from a running Docker container in real time. Identify the container by name or ID via container_id; host_id is optional unless the name is ambiguous. Optionally filter by log level, text search, or regex pattern. Streams continuously until cancelled.",
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
			Name:           toolRetroScan,
			Description:    "Read every container's logs since a point in time in one pass, returning true per-level counts, the newest matching lines, and each container's restart count and last exit. For the cloud's first look at a newly linked instance; never offered to a model.",
			ParametersJson: retroScanParams,
			Scope:          pb.ToolScope_TOOL_SCOPE_INSTANCE,
			ReadOnly:       true,
			Internal:       true,
		},
		{
			Name:           toolInspectContainer,
			Description:    "Get detailed configuration of a Docker container including environment variables, port mappings, mounts, restart policy, network mode, labels, and resource limits.",
			ParametersJson: targetedParams,
			Scope:          pb.ToolScope_TOOL_SCOPE_CONTAINER,
			ReadOnly:       true,
		},
	}

	// A cloud call is someone asking, so manual mode answers it too. Off means
	// Dozzle never contacts a registry, so the tool does not exist.
	if deps.ImageCheckMode.Allows(true) {
		tools = append(tools, &pb.ToolDefinition{
			Name:           toolCheckImageUpdates,
			Description:    "Check which containers run an outdated image: asks each image's registry whether its tag now points to a newer image than the one the container runs. Use it for \"what needs updating\" or \"is X up to date\". Each container's status is up-to-date, update-available, pinned (digest-pinned, cannot drift), not-checkable (built locally), auth-required (private registry), skipped (opted out by label) or unknown. Updating is a separate step: call update_container only after the user confirms which containers to update.",
			ParametersJson: checkImageUpdatesParams,
			Scope:          pb.ToolScope_TOOL_SCOPE_INSTANCE,
			ReadOnly:       true,
		})
	}

	if enableActions {
		tools = append(tools,
			&pb.ToolDefinition{
				Name:           toolStartContainer,
				Description:    "Start a stopped Docker container",
				ParametersJson: writeTargetedParams,
				Scope:          pb.ToolScope_TOOL_SCOPE_CONTAINER,
			},
			&pb.ToolDefinition{
				Name:           toolStopContainer,
				Description:    "Stop a running Docker container",
				ParametersJson: writeTargetedParams,
				Scope:          pb.ToolScope_TOOL_SCOPE_CONTAINER,
			},
			&pb.ToolDefinition{
				Name:           toolRestartContainer,
				Description:    "Restart a Docker container",
				ParametersJson: writeTargetedParams,
				Scope:          pb.ToolScope_TOOL_SCOPE_CONTAINER,
			},
			&pb.ToolDefinition{
				Name:           toolRemoveContainer,
				Description:    "Remove a Docker container. The container must be stopped first — call stop_container if it is still running. Confirm with the user before removing, since the container is gone permanently.",
				ParametersJson: writeTargetedParams,
				Scope:          pb.ToolScope_TOOL_SCOPE_CONTAINER,
			},
			&pb.ToolDefinition{
				Name:           toolUpdateContainer,
				Description:    "Update a Docker container by pulling the latest version of its image and recreating it with the same configuration. If the image is already up to date, no recreation occurs. For swarm service containers, updates the service instead. A stopped container is never updated: it has to be started first.",
				ParametersJson: writeTargetedParams,
				Scope:          pb.ToolScope_TOOL_SCOPE_CONTAINER,
			},
			&pb.ToolDefinition{
				Name:           toolRollbackContainer,
				Description:    "Roll a Docker container back to the image it ran before its last update, keeping its configuration and volumes. Only after the user confirms. Not for swarm services or stopped containers.",
				ParametersJson: rollbackContainerParams,
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

	allowed := tools[:0]
	for _, t := range tools {
		if p.mayCall(t.Name, enableActions) == nil {
			allowed = append(allowed, t)
		}
	}
	return allowed
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
	EnableActions bool
	// ImageCheckMode is --image-check-mode. The zero value offers no image
	// update checks.
	ImageCheckMode imagecheck.Mode
	HostService    ToolHostService
	// Principal is who the call runs as. The zero value is PrincipalAPIKey,
	// which is what every tool call meant before principals existed, so an
	// unset field never grants more than it used to.
	Principal           Principal
	NotificationService NotificationService
}

// scoped returns the host service already confined to the principal's labels.
// Tools use this instead of HostService so there is no way to ask for
// containers the caller may not see.
func (d ToolDeps) scoped() scopedHost {
	return scopedHost{hosts: d.HostService, labels: d.Principal.Labels}
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

func executeTool(ctx context.Context, name string, argsJSON string, deps ToolDeps) (*pb.CallToolResponse, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	if err := deps.Principal.mayCall(name, deps.EnableActions); err != nil {
		return nil, err
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
	case toolRetroScan:
		return executeRetroScan(ctx, argsJSON, deps)
	case toolListNotifications:
		return executeListNotifications(deps)
	case toolCheckImageUpdates:
		return executeCheckImageUpdates(ctx, argsJSON, deps)
	case toolStartContainer, toolStopContainer, toolRestartContainer, toolRemoveContainer:
		return executeContainerAction(ctx, name, argsJSON, deps)
	case toolUpdateContainer:
		return executeUpdateContainer(ctx, argsJSON, deps)
	case toolRollbackContainer:
		return executeRollbackContainer(ctx, argsJSON, deps)
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

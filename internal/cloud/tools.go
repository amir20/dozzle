package cloud

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/deploy"
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

	deployComposeParams = mustSchema(paramSchema{
		Type: "object",
		Properties: map[string]paramProperty{
			"yaml":    {Type: "string", Description: "The raw YAML content of the Docker Compose file to deploy"},
			"project": {Type: "string", Description: "Project name used as a prefix for resource names (networks, volumes, containers)"},
			"host_id": {Type: "string", Description: "Host ID to deploy to. Use live_list_hosts to find available host IDs."},
		},
		Required:             []string{"yaml", "project", "host_id"},
		AdditionalProperties: &boolFalse,
	})

	listDeployVersionsParams = mustSchema(paramSchema{
		Type: "object",
		Properties: map[string]paramProperty{
			"project": {Type: "string", Description: "Project name to list version history for"},
			"host_id": {Type: "string", Description: "Host ID where the project is deployed. Use live_list_hosts to find available host IDs."},
		},
		Required:             []string{"project", "host_id"},
		AdditionalProperties: &boolFalse,
	})

	rollbackDeployParams = mustSchema(paramSchema{
		Type: "object",
		Properties: map[string]paramProperty{
			"project":     {Type: "string", Description: "Project name to roll back"},
			"commit_hash": {Type: "string", Description: "Version ID (full or short) to roll back to. Use list_deploy_versions to find available IDs."},
			"host_id":     {Type: "string", Description: "Host ID where the project is deployed. Use live_list_hosts to find available host IDs."},
		},
		Required:             []string{"project", "commit_hash", "host_id"},
		AdditionalProperties: &boolFalse,
	})

	removeDeployParams = mustSchema(paramSchema{
		Type: "object",
		Properties: map[string]paramProperty{
			"project":        {Type: "string", Description: "Project name to tear down"},
			"host_id":        {Type: "string", Description: "Host ID where the project is deployed. Use live_list_hosts to find available host IDs."},
			"remove_volumes": {Type: "boolean", Description: "Optional. If true, also delete project-labeled named volumes (destructive — user data is lost). Defaults to false."},
		},
		Required:             []string{"project", "host_id"},
		AdditionalProperties: &boolFalse,
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
		Description: `Required. expr-lang expression selecting which containers trigger the alert. Fields: name, id, image, state, health, host, labels.
Examples: name contains "nginx"; state == "running"; image matches "redis.*"; name contains "api" && health == "healthy".`,
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
			Name:           "list_hosts",
			Description:    "List all Docker hosts connected to Dozzle with their name, CPU cores, total memory, Docker version, and availability status.",
			ParametersJson: noParams,
			Scope:          pb.ToolScope_TOOL_SCOPE_INSTANCE,
			ReadOnly:       true,
		},
		{
			Name:           "find_containers",
			Description:    "Search for Docker containers by name, state, or health status. All parameters are optional. Returns container ID, name, image, state, health, and host. Use this before start/stop/restart actions to get the container ID and host.",
			ParametersJson: findContainerParams,
			Scope:          pb.ToolScope_TOOL_SCOPE_INSTANCE,
			ReadOnly:       true,
		},
		{
			Name:           "list_running_containers",
			Description:    "List all currently running Docker containers. Use find_containers instead if you need to filter by name or health status.",
			ParametersJson: noParams,
			Scope:          pb.ToolScope_TOOL_SCOPE_INSTANCE,
			ReadOnly:       true,
		},
		{
			Name:           "list_all_containers",
			Description:    "List all Docker containers including stopped, exited, and previously run containers.",
			ParametersJson: noParams,
			Scope:          pb.ToolScope_TOOL_SCOPE_INSTANCE,
			ReadOnly:       true,
		},
		{
			Name:           "get_running_container_stats",
			Description:    "Get real-time CPU and memory usage statistics for all currently running Docker containers. Returns current percentages and peak values over the last 5 minutes.",
			ParametersJson: noParams,
			Scope:          pb.ToolScope_TOOL_SCOPE_INSTANCE,
			ReadOnly:       true,
		},
		{
			Name:           "fetch_container_logs",
			Description:    "Fetch raw logs from a running Docker container. Requires container_id and host from find_containers. Optionally filter by time range, log level, text search, or regex pattern. Returns up to 100 matching log lines.",
			ParametersJson: fetchLogsParams,
			Scope:          pb.ToolScope_TOOL_SCOPE_CONTAINER,
			ReadOnly:       true,
		},
		{
			Name:           "stream_logs",
			Description:    "Stream live logs from a running Docker container in real time. Requires container_id and host_id from find_containers. Optionally filter by log level, text search, or regex pattern. Streams continuously until cancelled.",
			ParametersJson: streamLogsParams,
			Scope:          pb.ToolScope_TOOL_SCOPE_CONTAINER,
			ReadOnly:       true,
		},
		{
			Name:           "list_notifications",
			Description:    "List configured alert subscriptions on a Dozzle host. Use this to check whether an alert already exists before creating a new one with create_log_notification, create_metric_notification, or create_event_notification.",
			ParametersJson: listNotificationsParams,
			Scope:          pb.ToolScope_TOOL_SCOPE_INSTANCE,
			ReadOnly:       true,
		},
		{
			Name:           "inspect_container",
			Description:    "Get detailed configuration of a Docker container including environment variables, port mappings, mounts, restart policy, network mode, labels, and resource limits.",
			ParametersJson: targetedParams,
			Scope:          pb.ToolScope_TOOL_SCOPE_CONTAINER,
			ReadOnly:       true,
		},
	}

	if enableActions {
		tools = append(tools,
			&pb.ToolDefinition{
				Name:           "start_container",
				Description:    "Start a stopped Docker container",
				ParametersJson: targetedParams,
				Scope:          pb.ToolScope_TOOL_SCOPE_CONTAINER,
			},
			&pb.ToolDefinition{
				Name:           "stop_container",
				Description:    "Stop a running Docker container",
				ParametersJson: targetedParams,
				Scope:          pb.ToolScope_TOOL_SCOPE_CONTAINER,
			},
			&pb.ToolDefinition{
				Name:           "restart_container",
				Description:    "Restart a Docker container",
				ParametersJson: targetedParams,
				Scope:          pb.ToolScope_TOOL_SCOPE_CONTAINER,
			},
			&pb.ToolDefinition{
				Name:           "remove_container",
				Description:    "Remove a Docker container. The container must be stopped first — call stop_container if it is still running. Confirm with the user before removing, since the container is gone permanently (its config is kept only if it was created by a deploy_compose project).",
				ParametersJson: targetedParams,
				Scope:          pb.ToolScope_TOOL_SCOPE_CONTAINER,
			},
			&pb.ToolDefinition{
				Name:           "update_container",
				Description:    "Update a Docker container by pulling the latest version of its image and recreating it with the same configuration. If the image is already up to date, no recreation occurs. For swarm service containers, updates the service instead.",
				ParametersJson: targetedParams,
				Scope:          pb.ToolScope_TOOL_SCOPE_CONTAINER,
			},
			&pb.ToolDefinition{
				Name:           "deploy_compose",
				Description:    "Deploy a Docker Compose file. Creates or updates a project. Creates networks, volumes, pulls images, and starts containers in dependency order. Only supports pre-built images (no build step). Each deployment is versioned for history and rollback.",
				ParametersJson: deployComposeParams,
				Scope:          pb.ToolScope_TOOL_SCOPE_HOST,
			},
			&pb.ToolDefinition{
				Name:           "list_deploy_versions",
				Description:    "List the deployment version history for a project. Returns version IDs, timestamps, and messages. Use the version ID with rollback_deploy to revert to a previous configuration.",
				ParametersJson: listDeployVersionsParams,
				Scope:          pb.ToolScope_TOOL_SCOPE_HOST,
				ReadOnly:       true,
			},
			&pb.ToolDefinition{
				Name:           "rollback_deploy",
				Description:    "Roll back a project to a previous deployment version. Restores the compose configuration from the specified version and redeploys. Use list_deploy_versions to find available version IDs.",
				ParametersJson: rollbackDeployParams,
				Scope:          pb.ToolScope_TOOL_SCOPE_HOST,
			},
			&pb.ToolDefinition{
				Name:           "remove_deploy",
				Description:    "Tear down a deployed project: stops and removes its containers, removes project-labeled networks, and deletes the stored version history for the project. Named volumes are preserved by default — set remove_volumes=true to also delete them (destructive — user data is lost). Ask the user to confirm before setting remove_volumes=true.",
				ParametersJson: removeDeployParams,
				Scope:          pb.ToolScope_TOOL_SCOPE_HOST,
			},
			&pb.ToolDefinition{
				Name:           "create_log_notification",
				Description:    "Create an alert that fires when a container log line matches a filter. Requires a container_expression selecting which containers to watch and a log_expression matched against each log line. Alerts are delivered through the user's Dozzle Cloud channels.",
				ParametersJson: createLogNotificationParams,
				Scope:          pb.ToolScope_TOOL_SCOPE_INSTANCE,
			},
			&pb.ToolDefinition{
				Name:           "create_metric_notification",
				Description:    "Create an alert that fires when container CPU/memory usage crosses a threshold. Requires a container_expression selecting which containers to watch and a metric_expression evaluated against their stats. Alerts are delivered through the user's Dozzle Cloud channels.",
				ParametersJson: createMetricNotificationParams,
				Scope:          pb.ToolScope_TOOL_SCOPE_INSTANCE,
			},
			&pb.ToolDefinition{
				Name:           "create_event_notification",
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
// DeployManager may be nil in modes without a local docker daemon (e.g., k8s);
// deploy tools will then return a "not configured" error.
// NotificationService may be nil in modes without a notification manager
// (e.g., k8s); notification tools will then return a "not configured" error.
type ToolDeps struct {
	EnableActions       bool
	HostService         ToolHostService
	Labels              container.ContainerLabels
	DeployManager       *deploy.Manager
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

func executeTool(ctx context.Context, name string, argsJSON string, deps ToolDeps) (*pb.CallToolResponse, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	switch name {
	case "list_hosts":
		return executeListHosts(deps)
	case "find_containers":
		return executeFindContainers(argsJSON, deps)
	case "list_running_containers":
		return executeListRunningContainers(deps)
	case "list_all_containers":
		return executeListAllContainers(deps)
	case "get_running_container_stats":
		return executeGetRunningContainerStats(deps)
	case "fetch_container_logs":
		return executeFetchContainerLogs(ctx, argsJSON, deps)
	case "inspect_container":
		return executeInspectContainer(argsJSON, deps)
	case "list_notifications":
		return executeListNotifications(deps)
	case "start_container", "stop_container", "restart_container", "remove_container":
		if !deps.EnableActions {
			return nil, fmt.Errorf("container actions are not enabled")
		}
		return executeContainerAction(ctx, name, argsJSON, deps)
	case "update_container":
		if !deps.EnableActions {
			return nil, fmt.Errorf("container actions are not enabled")
		}
		return executeUpdateContainer(ctx, argsJSON, deps)
	case "deploy_compose":
		if !deps.EnableActions {
			return nil, fmt.Errorf("container actions are not enabled")
		}
		return executeDeployCompose(ctx, argsJSON, deps)
	case "list_deploy_versions":
		if !deps.EnableActions {
			return nil, fmt.Errorf("container actions are not enabled")
		}
		return executeListDeployVersions(ctx, argsJSON, deps)
	case "rollback_deploy":
		if !deps.EnableActions {
			return nil, fmt.Errorf("container actions are not enabled")
		}
		return executeRollbackDeploy(ctx, argsJSON, deps)
	case "remove_deploy":
		if !deps.EnableActions {
			return nil, fmt.Errorf("container actions are not enabled")
		}
		return executeRemoveDeploy(ctx, argsJSON, deps)
	case "create_log_notification":
		if !deps.EnableActions {
			return nil, fmt.Errorf("container actions are not enabled")
		}
		return executeCreateLogNotification(argsJSON, deps)
	case "create_metric_notification":
		if !deps.EnableActions {
			return nil, fmt.Errorf("container actions are not enabled")
		}
		return executeCreateMetricNotification(argsJSON, deps)
	case "create_event_notification":
		if !deps.EnableActions {
			return nil, fmt.Errorf("container actions are not enabled")
		}
		return executeCreateEventNotification(argsJSON, deps)
	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

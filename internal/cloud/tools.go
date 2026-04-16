package cloud

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/deploy"
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
		},
		{
			Name:           "find_containers",
			Description:    "Search for Docker containers by name, state, or health status. All parameters are optional. Returns container ID, name, image, state, health, and host. Use this before start/stop/restart actions to get the container ID and host.",
			ParametersJson: findContainerParams,
		},
		{
			Name:           "list_running_containers",
			Description:    "List all currently running Docker containers. Use find_containers instead if you need to filter by name or health status.",
			ParametersJson: noParams,
		},
		{
			Name:           "list_all_containers",
			Description:    "List all Docker containers including stopped, exited, and previously run containers.",
			ParametersJson: noParams,
		},
		{
			Name:           "get_running_container_stats",
			Description:    "Get real-time CPU and memory usage statistics for all currently running Docker containers. Returns current percentages and peak values over the last 5 minutes.",
			ParametersJson: noParams,
		},
		{
			Name:           "fetch_container_logs",
			Description:    "Fetch raw logs from a running Docker container. Requires container_id and host from find_containers. Optionally filter by time range, log level, text search, or regex pattern. Returns up to 100 matching log lines.",
			ParametersJson: fetchLogsParams,
		},
		{
			Name:           "stream_logs",
			Description:    "Stream live logs from a running Docker container in real time. Requires container_id and host_id from find_containers. Optionally filter by log level, text search, or regex pattern. Streams continuously until cancelled.",
			ParametersJson: streamLogsParams,
		},
		{
			Name:           "inspect_container",
			Description:    "Get detailed configuration of a Docker container including environment variables, port mappings, mounts, restart policy, network mode, labels, and resource limits.",
			ParametersJson: targetedParams,
		},
	}

	if enableActions {
		tools = append(tools,
			&pb.ToolDefinition{
				Name:           "start_container",
				Description:    "Start a stopped Docker container",
				ParametersJson: targetedParams,
			},
			&pb.ToolDefinition{
				Name:           "stop_container",
				Description:    "Stop a running Docker container",
				ParametersJson: targetedParams,
			},
			&pb.ToolDefinition{
				Name:           "restart_container",
				Description:    "Restart a Docker container",
				ParametersJson: targetedParams,
			},
			&pb.ToolDefinition{
				Name:           "update_container",
				Description:    "Update a Docker container by pulling the latest version of its image and recreating it with the same configuration. If the image is already up to date, no recreation occurs. For swarm service containers, updates the service instead.",
				ParametersJson: targetedParams,
			},
			&pb.ToolDefinition{
				Name:           "deploy_compose",
				Description:    "Deploy a Docker Compose file. Creates or updates a project. Creates networks, volumes, pulls images, and starts containers in dependency order. Only supports pre-built images (no build step). Each deployment is versioned for history and rollback.",
				ParametersJson: deployComposeParams,
			},
			&pb.ToolDefinition{
				Name:           "list_deploy_versions",
				Description:    "List the deployment version history for a project. Returns version IDs, timestamps, and messages. Use the version ID with rollback_deploy to revert to a previous configuration.",
				ParametersJson: listDeployVersionsParams,
			},
			&pb.ToolDefinition{
				Name:           "rollback_deploy",
				Description:    "Roll back a project to a previous deployment version. Restores the compose configuration from the specified version and redeploys. Use list_deploy_versions to find available version IDs.",
				ParametersJson: rollbackDeployParams,
			},
		)
	}

	return tools
}

// ToolDeps bundles the dependencies required to execute cloud tool calls.
// DeployManager may be nil in modes without a local docker daemon (e.g., k8s);
// deploy tools will then return a "not configured" error.
type ToolDeps struct {
	EnableActions bool
	HostService   ToolHostService
	Labels        container.ContainerLabels
	DeployManager *deploy.Manager
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
	case "start_container", "stop_container", "restart_container":
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
	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

package cloud

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/amir20/dozzle/internal/container"
	container_support "github.com/amir20/dozzle/internal/support/container"
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

// ContainerActioner can perform actions on a container
type ContainerActioner interface {
	Action(ctx context.Context, action container.ContainerAction) error
}

// ToolHostService is the subset of HostService needed by tool execution
type ToolHostService interface {
	ListAllContainers(labels container.ContainerLabels) ([]container.Container, []error)
	FindContainer(host string, id string, labels container.ContainerLabels) (ContainerActioner, error)
}

// HostServiceAdapter wraps the web.HostService to satisfy ToolHostService
type HostServiceAdapter struct {
	ListAllContainersFunc func(labels container.ContainerLabels) ([]container.Container, []error)
	FindContainerFunc     func(host string, id string, labels container.ContainerLabels) (*container_support.ContainerService, error)
}

func (a *HostServiceAdapter) ListAllContainers(labels container.ContainerLabels) ([]container.Container, []error) {
	return a.ListAllContainersFunc(labels)
}

func (a *HostServiceAdapter) FindContainer(host string, id string, labels container.ContainerLabels) (ContainerActioner, error) {
	cs, err := a.FindContainerFunc(host, id, labels)
	if err != nil {
		return nil, err
	}
	return cs, nil
}

type containerResult struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Image     string `json:"image"`
	State     string `json:"state"`
	Host      string `json:"host"`
	Created   string `json:"created"`
	StartedAt string `json:"startedAt"`
}

type actionResult struct {
	Success     bool   `json:"success"`
	ContainerID string `json:"containerId"`
	Action      string `json:"action"`
}

type containerActionArgs struct {
	ContainerID string `json:"container_id"`
}

// AvailableTools returns the list of tool definitions based on configuration.
// list_containers is always available. Action tools require enableActions.
func AvailableTools(enableActions bool) []openai.FunctionDefinition {
	tools := []openai.FunctionDefinition{
		{
			Name:        "list_containers",
			Description: "List all Docker containers with their current state, name, image, and host",
			Parameters: jsonschema.Definition{
				Type:       jsonschema.Object,
				Properties: map[string]jsonschema.Definition{},
			},
		},
	}

	if enableActions {
		tools = append(tools,
			openai.FunctionDefinition{
				Name:        "start_container",
				Description: "Start a stopped Docker container",
				Parameters: jsonschema.Definition{
					Type: jsonschema.Object,
					Properties: map[string]jsonschema.Definition{
						"container_id": {
							Type:        jsonschema.String,
							Description: "The container ID to start",
						},
					},
					Required: []string{"container_id"},
				},
			},
			openai.FunctionDefinition{
				Name:        "stop_container",
				Description: "Stop a running Docker container",
				Parameters: jsonschema.Definition{
					Type: jsonschema.Object,
					Properties: map[string]jsonschema.Definition{
						"container_id": {
							Type:        jsonschema.String,
							Description: "The container ID to stop",
						},
					},
					Required: []string{"container_id"},
				},
			},
			openai.FunctionDefinition{
				Name:        "restart_container",
				Description: "Restart a Docker container",
				Parameters: jsonschema.Definition{
					Type: jsonschema.Object,
					Properties: map[string]jsonschema.Definition{
						"container_id": {
							Type:        jsonschema.String,
							Description: "The container ID to restart",
						},
					},
					Required: []string{"container_id"},
				},
			},
		)
	}

	return tools
}

// ExecuteTool dispatches a tool call by name and returns JSON result
func ExecuteTool(ctx context.Context, name string, argsJSON string, hostService ToolHostService, labels container.ContainerLabels) (string, error) {
	switch name {
	case "list_containers":
		return executeListContainers(hostService, labels)
	case "start_container":
		return executeContainerAction(ctx, argsJSON, container.Start, hostService, labels)
	case "stop_container":
		return executeContainerAction(ctx, argsJSON, container.Stop, hostService, labels)
	case "restart_container":
		return executeContainerAction(ctx, argsJSON, container.Restart, hostService, labels)
	default:
		return "", fmt.Errorf("unknown tool: %s", name)
	}
}

func executeListContainers(hostService ToolHostService, labels container.ContainerLabels) (string, error) {
	containers, _ := hostService.ListAllContainers(labels)

	results := make([]containerResult, len(containers))
	for i, c := range containers {
		results[i] = containerResult{
			ID:        c.ID,
			Name:      c.Name,
			Image:     c.Image,
			State:     c.State,
			Host:      c.Host,
			Created:   c.Created.Format("2006-01-02T15:04:05Z"),
			StartedAt: c.StartedAt.Format("2006-01-02T15:04:05Z"),
		}
	}

	data, err := json.Marshal(results)
	if err != nil {
		return "", fmt.Errorf("failed to marshal containers: %w", err)
	}
	return string(data), nil
}

func executeContainerAction(ctx context.Context, argsJSON string, action container.ContainerAction, hostService ToolHostService, labels container.ContainerLabels) (string, error) {
	var args containerActionArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("failed to parse arguments: %w", err)
	}

	if args.ContainerID == "" {
		return "", fmt.Errorf("container_id is required")
	}

	// FindContainer searches across all hosts when host is empty
	actioner, err := hostService.FindContainer("", args.ContainerID, labels)
	if err != nil {
		return "", fmt.Errorf("container not found: %w", err)
	}

	if err := actioner.Action(ctx, action); err != nil {
		return "", fmt.Errorf("action failed: %w", err)
	}

	result := actionResult{
		Success:     true,
		ContainerID: args.ContainerID,
		Action:      string(action),
	}

	data, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}
	return string(data), nil
}

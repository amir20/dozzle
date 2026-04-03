package cloud

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/amir20/dozzle/internal/container"
	container_support "github.com/amir20/dozzle/internal/support/container"
	"github.com/rs/zerolog/log"
)

// ToolHostService is the subset of HostService needed by tool execution
type ToolHostService interface {
	ListAllContainers(labels container.ContainerLabels) ([]container.Container, []error)
	FindContainer(host string, id string, labels container.ContainerLabels) (*container_support.ContainerService, error)
}

// FunctionDefinition describes a tool that can be called by the cloud service.
type FunctionDefinition struct {
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Parameters  ParameterDefinition `json:"parameters"`
}

// ParameterDefinition describes the JSON Schema parameters for a tool.
type ParameterDefinition struct {
	Type       string                         `json:"type"`
	Properties map[string]PropertyDefinition  `json:"properties"`
	Required   []string                       `json:"required,omitempty"`
}

// PropertyDefinition describes a single property in a tool's parameters.
type PropertyDefinition struct {
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
}

type containerResult struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Image         string   `json:"image"`
	Command       string   `json:"command"`
	Created       string   `json:"created"`
	StartedAt     string   `json:"startedAt"`
	FinishedAt    string   `json:"finishedAt,omitempty"`
	State         string   `json:"state"`
	Health        string   `json:"health,omitempty"`
	Host          string   `json:"host,omitempty"`
	Group         string   `json:"group,omitempty"`
	CPUPercent    *float64 `json:"cpuPercent,omitempty"`
	MaxCPU5Min    *float64 `json:"maxCpu5Min,omitempty"`
	MemoryPercent *float64 `json:"memoryPercent,omitempty"`
	MaxMemory5Min *float64 `json:"maxMemory5Min,omitempty"`
}

var actionMap = map[string]container.ContainerAction{
	"start_container":   container.Start,
	"stop_container":    container.Stop,
	"restart_container": container.Restart,
}

type actionResult struct {
	Success     bool   `json:"success"`
	ContainerID string `json:"containerId"`
	Action      string `json:"action"`
}

type containerActionArgs struct {
	ContainerID string `json:"container_id"`
	Host        string `json:"host"`
}

// AvailableTools returns the list of tool definitions based on configuration.
// list_containers is always available. Action tools require enableActions.
func AvailableTools(enableActions bool) []FunctionDefinition {
	tools := []FunctionDefinition{
		{
			Name:        "find_containers",
			Description: "List all Docker containers with their current state, name, image, and host",
			Parameters: ParameterDefinition{
				Type:       "object",
				Properties: map[string]PropertyDefinition{},
			},
		},
	}

	if enableActions {
		actionParams := ParameterDefinition{
				Type: "object",
				Properties: map[string]PropertyDefinition{
					"container_id": {
						Type:        "string",
						Description: "The container ID",
					},
					"host": {
						Type:        "string",
						Description: "The host name where the container is running",
					},
				},
				Required: []string{"container_id", "host"},
			}
		tools = append(tools,
			FunctionDefinition{
				Name:        "start_container",
				Description: "Start a stopped Docker container",
				Parameters:  actionParams,
			},
			FunctionDefinition{
				Name:        "stop_container",
				Description: "Stop a running Docker container",
				Parameters:  actionParams,
			},
			FunctionDefinition{
				Name:        "restart_container",
				Description: "Restart a Docker container",
				Parameters:  actionParams,
			},
		)
	}

	return tools
}

// marshalTools serializes tool definitions to JSON strings for the gRPC response.
func marshalTools(enableActions bool) []string {
	tools := AvailableTools(enableActions)
	result := make([]string, 0, len(tools))
	for _, tool := range tools {
		data, err := json.Marshal(tool)
		if err != nil {
			log.Error().Err(err).Str("tool", tool.Name).Msg("failed to marshal tool definition")
			continue
		}
		result = append(result, string(data))
	}
	return result
}

// ExecuteTool dispatches a tool call by name and returns JSON result.
// enableActions must be true for action tools (start/stop/restart) to execute.
func ExecuteTool(ctx context.Context, name string, argsJSON string, enableActions bool, hostService ToolHostService, labels container.ContainerLabels) (string, error) {
	switch name {
	case "find_containers":
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return executeListContainers(hostService, labels)
	case "start_container", "stop_container", "restart_container":
		if !enableActions {
			return "", fmt.Errorf("container actions are not enabled")
		}
		return executeContainerAction(ctx, argsJSON, actionMap[name], hostService, labels)
	default:
		return "", fmt.Errorf("unknown tool: %s", name)
	}
}

func executeListContainers(hostService ToolHostService, labels container.ContainerLabels) (string, error) {
	containers, errs := hostService.ListAllContainers(labels)
	for _, err := range errs {
		if err != nil {
			log.Warn().Err(err).Msg("error listing containers from host")
		}
	}

	results := make([]containerResult, len(containers))
	for i, c := range containers {
		r := containerResult{
			ID:         c.ID,
			Name:       c.Name,
			Image:      c.Image,
			Command:    c.Command,
			Created:    c.Created.UTC().Format(time.RFC3339),
			StartedAt:  c.StartedAt.UTC().Format(time.RFC3339),
			FinishedAt: formatTimeOrEmpty(c.FinishedAt),
			State:      c.State,
			Health:     c.Health,
			Host:       c.Host,
			Group:      c.Group,
		}

		if c.Stats != nil && c.Stats.Len() > 0 {
			stats := c.Stats.Data()
			latest := stats[len(stats)-1]
			r.CPUPercent = &latest.CPUPercent
			r.MemoryPercent = &latest.MemoryPercent

			var maxCPU, maxMem float64
			for _, s := range stats {
				maxCPU = max(maxCPU, s.CPUPercent)
				maxMem = max(maxMem, s.MemoryPercent)
			}
			r.MaxCPU5Min = &maxCPU
			r.MaxMemory5Min = &maxMem
		}

		results[i] = r
	}

	data, err := json.Marshal(results)
	if err != nil {
		return "", fmt.Errorf("failed to marshal containers: %w", err)
	}
	return string(data), nil
}

func formatTimeOrEmpty(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func executeContainerAction(ctx context.Context, argsJSON string, action container.ContainerAction, hostService ToolHostService, labels container.ContainerLabels) (string, error) {
	var args containerActionArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("failed to parse arguments: %w", err)
	}

	if args.ContainerID == "" {
		return "", fmt.Errorf("container_id is required")
	}

	if args.Host == "" {
		return "", fmt.Errorf("host is required")
	}

	cs, err := hostService.FindContainer(args.Host, args.ContainerID, labels)
	if err != nil {
		return "", fmt.Errorf("container not found: %w", err)
	}

	if err := cs.Action(ctx, action); err != nil {
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

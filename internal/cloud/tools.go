package cloud

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/amir20/dozzle/internal/container"
	container_support "github.com/amir20/dozzle/internal/support/container"
	pb "github.com/amir20/dozzle/proto/cloud"
	"github.com/rs/zerolog/log"
)

// ToolHostService is the subset of HostService needed by tool execution
type ToolHostService interface {
	ListAllContainers(labels container.ContainerLabels) ([]container.Container, []error)
	FindContainer(host string, id string, labels container.ContainerLabels) (*container_support.ContainerService, error)
	Hosts() []container.Host
}

type containerActionArgs struct {
	ContainerID string `json:"container_id"`
	Host        string `json:"host"`
}

type findContainersArgs struct {
	Name   string `json:"name"`
	Image  string `json:"image"`
	State  string `json:"state"`
	Health string `json:"health"`
}

// AvailableTools returns the list of tool definitions based on configuration.
func AvailableTools(enableActions bool) []*pb.ToolDefinition {
	noParams := `{"type":"object","properties":{}}`

	findContainerParams := `{"type":"object","properties":{"name":{"type":"string","description":"Optional container name to search for (partial match supported)"},"image":{"type":"string","description":"Optional image name to filter by (partial match supported)"},"state":{"type":"string","description":"Optional state filter (e.g. running, exited, created)"},"health":{"type":"string","description":"Optional health status filter (e.g. healthy, unhealthy, none)"}}}`

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
	}

	if enableActions {
		actionParams := `{"type":"object","properties":{"container_id":{"type":"string","description":"The container ID"},"host":{"type":"string","description":"The host name where the container is running"}},"required":["container_id","host"]}`
		tools = append(tools,
			&pb.ToolDefinition{
				Name:           "start_container",
				Description:    "Start a stopped Docker container",
				ParametersJson: actionParams,
			},
			&pb.ToolDefinition{
				Name:           "stop_container",
				Description:    "Stop a running Docker container",
				ParametersJson: actionParams,
			},
			&pb.ToolDefinition{
				Name:           "restart_container",
				Description:    "Restart a Docker container",
				ParametersJson: actionParams,
			},
		)
	}

	return tools
}

// ExecuteTool dispatches a tool call by name and returns a proto CallToolResponse.
// enableActions must be true for action tools (start/stop/restart) to execute.
func ExecuteTool(ctx context.Context, name string, argsJSON string, enableActions bool, hostService ToolHostService, labels container.ContainerLabels) *pb.CallToolResponse {
	resp, err := executeTool(ctx, name, argsJSON, enableActions, hostService, labels)
	if err != nil {
		return &pb.CallToolResponse{
			Success: false,
			Error:   err.Error(),
		}
	}
	return resp
}

func executeTool(ctx context.Context, name string, argsJSON string, enableActions bool, hostService ToolHostService, labels container.ContainerLabels) (*pb.CallToolResponse, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	switch name {
	case "list_hosts":
		hosts := hostService.Hosts()
		result := make([]*pb.HostInfo, len(hosts))
		for i, h := range hosts {
			result[i] = &pb.HostInfo{
				Id:            h.ID,
				Name:          h.Name,
				NCpu:          int32(h.NCPU),
				MemTotal:      h.MemTotal,
				DockerVersion: h.DockerVersion,
				AgentVersion:  h.AgentVersion,
				Type:          h.Type,
				Available:     h.Available,
			}
		}
		return &pb.CallToolResponse{
			Success: true,
			Result:  &pb.CallToolResponse_ListHosts{ListHosts: &pb.ListHostsResult{Hosts: result}},
		}, nil

	case "find_containers":
		var args findContainersArgs
		if argsJSON != "" {
			if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
				return nil, fmt.Errorf("failed to parse arguments: %w", err)
			}
		}

		containers, errs := hostService.ListAllContainers(labels)
		logHostErrors(errs)

		result := make([]*pb.ContainerInfo, 0, len(containers))
		for _, c := range containers {
			if args.Name != "" && !containsIgnoreCase(c.Name, args.Name) {
				continue
			}
			if args.Image != "" && !containsIgnoreCase(c.Image, args.Image) {
				continue
			}
			if args.State != "" && !strings.EqualFold(c.State, args.State) {
				continue
			}
			if args.Health != "" && !strings.EqualFold(c.Health, args.Health) {
				continue
			}
			// Keep raw host ID so action tools can use it directly
			result = append(result, containerToProto(c, nil))
		}
		return &pb.CallToolResponse{
			Success: true,
			Result:  &pb.CallToolResponse_ListContainers{ListContainers: &pb.ListContainersResult{Containers: result}},
		}, nil

	case "list_running_containers":
		containers, errs := hostService.ListAllContainers(labels)
		logHostErrors(errs)
		hostNames := buildHostNameMap(hostService)

		result := make([]*pb.ContainerInfo, 0, len(containers))
		for _, c := range containers {
			if c.State != "running" {
				continue
			}
			result = append(result, containerToProto(c, hostNames))
		}
		return &pb.CallToolResponse{
			Success: true,
			Result:  &pb.CallToolResponse_ListContainers{ListContainers: &pb.ListContainersResult{Containers: result}},
		}, nil

	case "list_all_containers":
		containers, errs := hostService.ListAllContainers(labels)
		logHostErrors(errs)
		hostNames := buildHostNameMap(hostService)

		result := make([]*pb.ContainerInfo, 0, len(containers))
		for _, c := range containers {
			result = append(result, containerToProto(c, hostNames))
		}
		return &pb.CallToolResponse{
			Success: true,
			Result:  &pb.CallToolResponse_ListContainers{ListContainers: &pb.ListContainersResult{Containers: result}},
		}, nil

	case "get_running_container_stats":
		containers, errs := hostService.ListAllContainers(labels)
		logHostErrors(errs)
		hostNames := buildHostNameMap(hostService)

		result := make([]*pb.ContainerStatEntry, 0, len(containers))
		for _, c := range containers {
			if c.State != "running" {
				continue
			}
			if c.Stats == nil || c.Stats.Len() == 0 {
				continue
			}

			stats := c.Stats.Data()
			latest := stats[len(stats)-1]

			var maxCPU, maxMem float64
			for _, s := range stats {
				maxCPU = max(maxCPU, s.CPUPercent)
				maxMem = max(maxMem, s.MemoryPercent)
			}

			result = append(result, &pb.ContainerStatEntry{
				Id:             c.ID,
				Name:           c.Name,
				Host:           resolveHostName(c.Host, hostNames),
				CpuPercent:     latest.CPUPercent,
				MemoryPercent:  latest.MemoryPercent,
				MemoryUsage:    latest.MemoryUsage,
				MaxCpu_5Min:    maxCPU,
				MaxMemory_5Min: maxMem,
			})
		}
		return &pb.CallToolResponse{
			Success: true,
			Result:  &pb.CallToolResponse_ContainerStats{ContainerStats: &pb.ContainerStatsResult{Stats: result}},
		}, nil

	case "start_container", "stop_container", "restart_container":
		if !enableActions {
			return nil, fmt.Errorf("container actions are not enabled")
		}
		var args containerActionArgs
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return nil, fmt.Errorf("failed to parse arguments: %w", err)
		}

		action, err := resolveAction(name)
		if err != nil {
			return nil, err
		}

		actionResult, err := executeAction(ctx, args.Host, args.ContainerID, action, hostService, labels)
		if err != nil {
			return nil, err
		}
		return &pb.CallToolResponse{
			Success: true,
			Result:  &pb.CallToolResponse_Action{Action: actionResult},
		}, nil

	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

func resolveAction(name string) (container.ContainerAction, error) {
	switch name {
	case "start_container":
		return container.Start, nil
	case "stop_container":
		return container.Stop, nil
	case "restart_container":
		return container.Restart, nil
	default:
		return "", fmt.Errorf("unknown action: %s", name)
	}
}

func executeAction(ctx context.Context, host, containerID string, action container.ContainerAction, hostService ToolHostService, labels container.ContainerLabels) (*pb.ActionResult, error) {
	if containerID == "" {
		return nil, fmt.Errorf("container_id is required")
	}

	if host == "" {
		return nil, fmt.Errorf("host is required")
	}

	cs, err := hostService.FindContainer(host, containerID, labels)
	if err != nil {
		return nil, fmt.Errorf("container not found: %w", err)
	}

	if err := cs.Action(ctx, action); err != nil {
		return nil, fmt.Errorf("action failed: %w", err)
	}

	return &pb.ActionResult{
		Success:     true,
		ContainerId: containerID,
		Action:      string(action),
	}, nil
}

// buildHostNameMap creates a mapping from host ID to host name.
func buildHostNameMap(hostService ToolHostService) map[string]string {
	hosts := hostService.Hosts()
	m := make(map[string]string, len(hosts))
	for _, h := range hosts {
		m[h.ID] = h.Name
	}
	return m
}

// resolveHostName returns the host name for a given host ID, falling back to the ID itself.
func resolveHostName(hostID string, hostNames map[string]string) string {
	if name, ok := hostNames[hostID]; ok {
		return name
	}
	return hostID
}

func containerToProto(c container.Container, hostNames map[string]string) *pb.ContainerInfo {
	return &pb.ContainerInfo{
		Id:         c.ID,
		Name:       c.Name,
		Image:      c.Image,
		Command:    c.Command,
		Created:    c.Created.UTC().Format(time.RFC3339),
		StartedAt:  c.StartedAt.UTC().Format(time.RFC3339),
		FinishedAt: formatTimeOrEmpty(c.FinishedAt),
		State:      c.State,
		Health:     c.Health,
		Host:       resolveHostName(c.Host, hostNames),
		Group:      c.Group,
	}
}

func formatTimeOrEmpty(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func containsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

func logHostErrors(errs []error) {
	for _, err := range errs {
		if err != nil {
			log.Warn().Err(err).Msg("error listing containers from host")
		}
	}
}

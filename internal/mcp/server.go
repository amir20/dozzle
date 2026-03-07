package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/amir20/dozzle/internal/container"
	container_support "github.com/amir20/dozzle/internal/support/container"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// HostService is the subset of web.HostService needed by the MCP server.
type HostService interface {
	FindContainer(host string, id string, labels container.ContainerLabels) (*container_support.ContainerService, error)
	ListAllContainers(labels container.ContainerLabels) ([]container.Container, []error)
	Hosts() []container.Host
}

// Server wraps an MCP server that exposes Dozzle container operations as tools.
type Server struct {
	mcpServer   *server.MCPServer
	hostService HostService
	labels      container.ContainerLabels
}

// NewServer creates a new MCP server with Dozzle tools registered.
func NewServer(hostService HostService, labels container.ContainerLabels, version string) *Server {
	s := &Server{
		hostService: hostService,
		labels:      labels,
	}

	mcpServer := server.NewMCPServer(
		"dozzle",
		version,
		server.WithToolCapabilities(false),
		server.WithInstructions("Dozzle MCP server provides tools to list Docker containers, read container logs, and perform container actions (start/stop/restart)."),
	)

	s.mcpServer = mcpServer
	s.registerTools()

	return s
}

// ServeHTTP starts the MCP server as a Streamable HTTP server on the given address.
func (s *Server) ServeHTTP(addr string) error {
	httpServer := server.NewStreamableHTTPServer(s.mcpServer)
	return httpServer.Start(addr)
}

func (s *Server) registerTools() {
	s.mcpServer.AddTool(listContainersTool(), s.handleListContainers)
	s.mcpServer.AddTool(getContainerLogsTool(), s.handleGetContainerLogs)
	s.mcpServer.AddTool(containerActionTool(), s.handleContainerAction)
	s.mcpServer.AddTool(listHostsTool(), s.handleListHosts)
	s.mcpServer.AddTool(getContainerStatsTool(), s.handleGetContainerStats)
}

// --- Tool Definitions ---

func listContainersTool() mcp.Tool {
	return mcp.NewTool("list_containers",
		mcp.WithDescription("List all Docker containers across all hosts. Returns container ID, name, image, state, host, and other metadata."),
		mcp.WithString("state",
			mcp.Description("Filter by container state (running, exited, created, paused, dead). Leave empty for all."),
			mcp.Enum("running", "exited", "created", "paused", "dead", ""),
		),
		mcp.WithReadOnlyHintAnnotation(true),
	)
}

func getContainerLogsTool() mcp.Tool {
	return mcp.NewTool("get_container_logs",
		mcp.WithDescription("Fetch processed logs from a Docker container. Returns structured log entries with detected log levels, JSON parsing, and multi-line grouping. Each entry includes timestamp, level, stream (stdout/stderr), message type (single/complex/group), and the parsed message content."),
		mcp.WithString("host",
			mcp.Description("The host ID where the container is running. Use list_containers to find this."),
			mcp.Required(),
		),
		mcp.WithString("container_id",
			mcp.Description("The container ID (or short ID) to get logs from. Use list_containers to find this."),
			mcp.Required(),
		),
		mcp.WithNumber("since_minutes",
			mcp.Description("Fetch logs from the last N minutes. Defaults to 5."),
		),
		mcp.WithString("stream",
			mcp.Description("Which output stream to read: stdout, stderr, or all."),
			mcp.Enum("stdout", "stderr", "all"),
		),
		mcp.WithReadOnlyHintAnnotation(true),
	)
}

func containerActionTool() mcp.Tool {
	return mcp.NewTool("container_action",
		mcp.WithDescription("Perform an action on a Docker container: start, stop, or restart."),
		mcp.WithString("host",
			mcp.Description("The host ID where the container is running."),
			mcp.Required(),
		),
		mcp.WithString("container_id",
			mcp.Description("The container ID to act on."),
			mcp.Required(),
		),
		mcp.WithString("action",
			mcp.Description("The action to perform."),
			mcp.Required(),
			mcp.Enum("start", "stop", "restart"),
		),
		mcp.WithDestructiveHintAnnotation(true),
	)
}

func listHostsTool() mcp.Tool {
	return mcp.NewTool("list_hosts",
		mcp.WithDescription("List all Docker hosts connected to Dozzle."),
		mcp.WithReadOnlyHintAnnotation(true),
	)
}

func getContainerStatsTool() mcp.Tool {
	return mcp.NewTool("get_container_stats",
		mcp.WithDescription("Get CPU and memory usage stats for a Docker container. Returns the last ~5 minutes of stats history (up to 300 data points) with CPU percentage, memory percentage, and memory usage in bytes."),
		mcp.WithString("host",
			mcp.Description("The host ID where the container is running. Use list_containers to find this."),
			mcp.Required(),
		),
		mcp.WithString("container_id",
			mcp.Description("The container ID to get stats for. Use list_containers to find this."),
			mcp.Required(),
		),
		mcp.WithReadOnlyHintAnnotation(true),
	)
}

// --- Tool Handlers ---

func (s *Server) handleListContainers(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	stateFilter := mcp.ParseString(request, "state", "")

	containers, errs := s.hostService.ListAllContainers(s.labels)
	for _, err := range errs {
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("error listing containers: %v", err)), nil
		}
	}

	type containerInfo struct {
		ID      string            `json:"id"`
		Name    string            `json:"name"`
		Image   string            `json:"image"`
		State   string            `json:"state"`
		Health  string            `json:"health,omitempty"`
		Host    string            `json:"host"`
		Created time.Time         `json:"created"`
		Labels  map[string]string `json:"labels,omitempty"`
		Group   string            `json:"group,omitempty"`
	}

	var results []containerInfo
	for _, c := range containers {
		if stateFilter != "" && c.State != stateFilter {
			continue
		}
		results = append(results, containerInfo{
			ID:      c.ID,
			Name:    c.Name,
			Image:   c.Image,
			State:   c.State,
			Health:  c.Health,
			Host:    c.Host,
			Created: c.Created,
			Labels:  c.Labels,
			Group:   c.Group,
		})
	}

	data, err := json.Marshal(results)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal containers: %v", err)), nil
	}

	return mcp.NewToolResultText(string(data)), nil
}

func (s *Server) handleGetContainerLogs(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	host := mcp.ParseString(request, "host", "")
	containerID := mcp.ParseString(request, "container_id", "")
	sinceMinutes := mcp.ParseFloat64(request, "since_minutes", 5)
	stream := mcp.ParseString(request, "stream", "all")

	if host == "" || containerID == "" {
		return mcp.NewToolResultError("host and container_id are required"), nil
	}

	containerSvc, err := s.hostService.FindContainer(host, containerID, s.labels)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("container not found: %v", err)), nil
	}

	var stdType container.StdType
	switch stream {
	case "stdout":
		stdType = container.STDOUT
	case "stderr":
		stdType = container.STDERR
	default:
		stdType = container.STDALL
	}

	since := time.Now().Add(-time.Duration(sinceMinutes) * time.Minute)
	events, err := containerSvc.LogsBetweenDates(ctx, since, time.Now(), stdType)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to read logs: %v", err)), nil
	}

	type logEntry struct {
		Timestamp string `json:"timestamp"`
		Level     string `json:"level,omitempty"`
		Stream    string `json:"stream,omitempty"`
		Type      string `json:"type"`
		Message   any    `json:"message"`
	}

	var entries []logEntry
	totalSize := 0
	const maxSize = 1024 * 1024 // 1MB limit

	for event := range events {
		var msg any
		switch event.Type {
		case container.LogTypeGroup:
			if fragments, ok := event.Message.([]container.LogFragment); ok {
				lines := make([]string, len(fragments))
				for i, f := range fragments {
					lines[i] = f.Message
				}
				msg = lines
			} else {
				msg = event.RawMessage
			}
		case container.LogTypeComplex:
			msg = event.Message
		default:
			msg = event.RawMessage
		}

		entry := logEntry{
			Timestamp: time.UnixMilli(event.Timestamp).UTC().Format(time.RFC3339Nano),
			Level:     event.Level,
			Stream:    event.Stream,
			Type:      string(event.Type),
			Message:   msg,
		}

		line, err := json.Marshal(entry)
		if err != nil {
			continue
		}

		totalSize += len(line) + 1
		if totalSize > maxSize {
			break
		}

		entries = append(entries, entry)
	}

	if len(entries) == 0 {
		return mcp.NewToolResultText("(no logs in the specified time range)"), nil
	}

	var sb strings.Builder
	encoder := json.NewEncoder(&sb)
	for _, entry := range entries {
		encoder.Encode(entry)
	}

	return mcp.NewToolResultText(strings.TrimRight(sb.String(), "\n")), nil
}

func (s *Server) handleContainerAction(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	host := mcp.ParseString(request, "host", "")
	containerID := mcp.ParseString(request, "container_id", "")
	action := mcp.ParseString(request, "action", "")

	if host == "" || containerID == "" || action == "" {
		return mcp.NewToolResultError("host, container_id, and action are required"), nil
	}

	containerAction, err := container.ParseContainerAction(action)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid action: %v", err)), nil
	}

	containerSvc, err := s.hostService.FindContainer(host, containerID, s.labels)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("container not found: %v", err)), nil
	}

	if err := containerSvc.Action(ctx, containerAction); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("action failed: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Successfully performed '%s' on container %s", action, containerSvc.Container.Name)), nil
}

func (s *Server) handleListHosts(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	hosts := s.hostService.Hosts()

	type hostInfo struct {
		ID            string `json:"id"`
		Name          string `json:"name"`
		NCPU          int    `json:"nCPU"`
		MemTotal      int64  `json:"memTotal"`
		DockerVersion string `json:"dockerVersion"`
		Type          string `json:"type"`
		Available     bool   `json:"available"`
	}

	var results []hostInfo
	for _, h := range hosts {
		results = append(results, hostInfo{
			ID:            h.ID,
			Name:          h.Name,
			NCPU:          h.NCPU,
			MemTotal:      h.MemTotal,
			DockerVersion: h.DockerVersion,
			Type:          h.Type,
			Available:     h.Available,
		})
	}

	data, err := json.Marshal(results)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal hosts: %v", err)), nil
	}

	return mcp.NewToolResultText(string(data)), nil
}

func (s *Server) handleGetContainerStats(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	host := mcp.ParseString(request, "host", "")
	containerID := mcp.ParseString(request, "container_id", "")

	if host == "" || containerID == "" {
		return mcp.NewToolResultError("host and container_id are required"), nil
	}

	containerSvc, err := s.hostService.FindContainer(host, containerID, s.labels)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("container not found: %v", err)), nil
	}

	c := containerSvc.Container

	type statEntry struct {
		CPUPercent    float64 `json:"cpuPercent"`
		MemoryPercent float64 `json:"memoryPercent"`
		MemoryUsage   float64 `json:"memoryUsageBytes"`
	}

	type statsResponse struct {
		ContainerID   string      `json:"containerId"`
		ContainerName string      `json:"containerName"`
		MemoryLimit   uint64      `json:"memoryLimitBytes,omitempty"`
		CPULimit      float64     `json:"cpuLimit,omitempty"`
		DataPoints    int         `json:"dataPoints"`
		Stats         []statEntry `json:"stats"`
	}

	var entries []statEntry
	if c.Stats != nil {
		for _, stat := range c.Stats.Data() {
			entries = append(entries, statEntry{
				CPUPercent:    stat.CPUPercent,
				MemoryPercent: stat.MemoryPercent,
				MemoryUsage:   stat.MemoryUsage,
			})
		}
	}

	resp := statsResponse{
		ContainerID:   c.ID,
		ContainerName: c.Name,
		MemoryLimit:   c.MemoryLimit,
		CPULimit:      c.CPULimit,
		DataPoints:    len(entries),
		Stats:         entries,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal stats: %v", err)), nil
	}

	return mcp.NewToolResultText(string(data)), nil
}

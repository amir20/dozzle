package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/amir20/dozzle/internal/container"
	container_support "github.com/amir20/dozzle/internal/support/container"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rs/zerolog/log"
)

// HostService is the subset of web.HostService needed by the MCP server.
type HostService interface {
	FindContainer(host string, id string, labels container.ContainerLabels) (*container_support.ContainerService, error)
	ListAllContainers(labels container.ContainerLabels) ([]container.Container, []error)
	Hosts() []container.Host
}

// Server wraps an MCP server that exposes Dozzle container operations as tools.
type Server struct {
	mcpServer   *mcp.Server
	hostService HostService
	labels      container.ContainerLabels
}

// NewServer creates a new MCP server with Dozzle tools registered.
func NewServer(hostService HostService, labels container.ContainerLabels, version string) *Server {
	s := &Server{
		hostService: hostService,
		labels:      labels,
	}

	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    "dozzle",
		Version: version,
	}, &mcp.ServerOptions{
		Instructions: "Dozzle MCP server provides tools to list Docker containers, read container logs, and view container stats.",
	})

	s.mcpServer = mcpServer
	s.registerTools()

	return s
}

// Handler returns an http.Handler for the MCP streamable HTTP transport.
func (s *Server) Handler() http.Handler {
	return mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return s.mcpServer
	}, nil)
}

// --- Tool Parameter Types ---

type listContainersParams struct {
	State *string `json:"state,omitempty" jsonschema:"Filter by container state (running, exited, created, paused, dead). Leave empty for all."`
}

type getContainerLogsParams struct {
	Host         string  `json:"host" jsonschema:"The host ID where the container is running. Use list_containers to find this."`
	ContainerID  string  `json:"container_id" jsonschema:"The container ID (or short ID) to get logs from. Use list_containers to find this."`
	SinceMinutes *int    `json:"since_minutes,omitempty" jsonschema:"Fetch logs from the last N minutes. Defaults to 5."`
	Stream       *string `json:"stream,omitempty" jsonschema:"Which output stream to read: stdout, stderr, or all. Defaults to all."`
}

type getContainerStatsParams struct {
	Host        string `json:"host" jsonschema:"The host ID where the container is running. Use list_containers to find this."`
	ContainerID string `json:"container_id" jsonschema:"The container ID to get stats for. Use list_containers to find this."`
}

func (s *Server) registerTools() {
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "list_containers",
		Description: "List all Docker containers across all hosts. Returns container ID, name, image, state, host, and other metadata.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, s.handleListContainers)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "get_container_logs",
		Description: "Fetch processed logs from a Docker container. Returns structured log entries with detected log levels, JSON parsing, and multi-line grouping.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, s.handleGetContainerLogs)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "list_hosts",
		Description: "List all Docker hosts connected to Dozzle.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, s.handleListHosts)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "get_container_stats",
		Description: "Get CPU and memory usage stats for a Docker container. Returns the last ~5 minutes of stats history with CPU percentage, memory percentage, and memory usage in bytes.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, s.handleGetContainerStats)
}

// --- Tool Handlers ---

func (s *Server) handleListContainers(ctx context.Context, _ *mcp.CallToolRequest, params *listContainersParams) (*mcp.CallToolResult, any, error) {
	containers, errs := s.hostService.ListAllContainers(s.labels)
	for _, err := range errs {
		if err != nil {
			log.Warn().Err(err).Msg("partial failure listing containers from a host")
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

	stateFilter := ""
	if params.State != nil {
		stateFilter = *params.State
	}

	results := []containerInfo{}
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
		return nil, nil, fmt.Errorf("failed to marshal containers: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
	}, nil, nil
}

func (s *Server) handleGetContainerLogs(ctx context.Context, _ *mcp.CallToolRequest, params *getContainerLogsParams) (*mcp.CallToolResult, any, error) {
	if params.Host == "" || params.ContainerID == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "host and container_id are required"}},
			IsError: true,
		}, nil, nil
	}

	containerSvc, err := s.hostService.FindContainer(params.Host, params.ContainerID, s.labels)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("container not found: %v", err)}},
			IsError: true,
		}, nil, nil
	}

	stream := ""
	if params.Stream != nil {
		stream = *params.Stream
	}
	var stdType container.StdType
	switch stream {
	case "", "all":
		stdType = container.STDALL
	case "stdout":
		stdType = container.STDOUT
	case "stderr":
		stdType = container.STDERR
	default:
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("invalid stream %q: must be stdout, stderr, or all", stream)}},
			IsError: true,
		}, nil, nil
	}

	sinceMinutes := 5
	if params.SinceMinutes != nil && *params.SinceMinutes > 0 {
		sinceMinutes = *params.SinceMinutes
	}

	logCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	since := time.Now().Add(-time.Duration(sinceMinutes) * time.Minute)
	events, err := containerSvc.LogsBetweenDates(logCtx, since, time.Now(), stdType)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("failed to read logs: %v", err)}},
			IsError: true,
		}, nil, nil
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
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "(no logs in the specified time range)"}},
		}, nil, nil
	}

	var sb strings.Builder
	encoder := json.NewEncoder(&sb)
	for _, entry := range entries {
		if err := encoder.Encode(entry); err != nil {
			return nil, nil, fmt.Errorf("failed to encode log entry: %w", err)
		}
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: strings.TrimRight(sb.String(), "\n")}},
	}, nil, nil
}

func (s *Server) handleListHosts(ctx context.Context, _ *mcp.CallToolRequest, _ *struct{}) (*mcp.CallToolResult, any, error) {
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

	results := []hostInfo{}
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
		return nil, nil, fmt.Errorf("failed to marshal hosts: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
	}, nil, nil
}

func (s *Server) handleGetContainerStats(ctx context.Context, _ *mcp.CallToolRequest, params *getContainerStatsParams) (*mcp.CallToolResult, any, error) {
	if params.Host == "" || params.ContainerID == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "host and container_id are required"}},
			IsError: true,
		}, nil, nil
	}

	containerSvc, err := s.hostService.FindContainer(params.Host, params.ContainerID, s.labels)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("container not found: %v", err)}},
			IsError: true,
		}, nil, nil
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

	entries := []statEntry{}
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
		return nil, nil, fmt.Errorf("failed to marshal stats: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
	}, nil, nil
}

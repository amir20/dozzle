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

type searchContainerLogsParams struct {
	Host          string  `json:"host" jsonschema:"The host ID where the container is running. Use list_containers to find this."`
	ContainerID   string  `json:"container_id" jsonschema:"The container ID (or short ID) to search logs from. Use list_containers to find this."`
	Query         string  `json:"query" jsonschema:"The search string to look for in log messages. Case-insensitive by default."`
	SinceMinutes  *int    `json:"since_minutes,omitempty" jsonschema:"Search logs from the last N minutes. Defaults to 5."`
	Stream        *string `json:"stream,omitempty" jsonschema:"Which output stream to search: stdout, stderr, or all. Defaults to all."`
	CaseSensitive *bool   `json:"case_sensitive,omitempty" jsonschema:"Whether to perform a case-sensitive search. Defaults to false."`
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
		Name:        "search_container_logs",
		Description: "Search container logs for a keyword or phrase. Returns only matching log entries, making it efficient for finding specific errors or events without downloading large volumes of logs.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, s.handleSearchContainerLogs)

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

const maxLogSize = 1024 * 1024 // 1MB limit

// mcpLogEntry is the JSON shape returned by the log tools.
type mcpLogEntry struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level,omitempty"`
	Stream    string `json:"stream,omitempty"`
	Type      string `json:"type"`
	Message   any    `json:"message"`
}

func errorResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
		IsError: true,
	}
}

func textResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}
}

// parseStream resolves the optional stream parameter into a StdType. It returns
// a non-nil error result when the value is invalid.
func parseStream(stream *string) (container.StdType, *mcp.CallToolResult) {
	value := ""
	if stream != nil {
		value = *stream
	}
	switch value {
	case "", "all":
		return container.STDALL, nil
	case "stdout":
		return container.STDOUT, nil
	case "stderr":
		return container.STDERR, nil
	default:
		return container.STDALL, errorResult(fmt.Sprintf("invalid stream %q: must be stdout, stderr, or all", value))
	}
}

// fetchLogs finds the container and returns its log events for the last
// sinceMinutes (defaulting to 5). On a user-facing failure it returns a non-nil
// error result; otherwise the caller must call cancel when done.
func (s *Server) fetchLogs(ctx context.Context, host, containerID string, stream *string, sinceMinutes *int) (<-chan *container.LogEvent, context.CancelFunc, *mcp.CallToolResult) {
	containerSvc, err := s.hostService.FindContainer(host, containerID, s.labels)
	if err != nil {
		return nil, nil, errorResult(fmt.Sprintf("container not found: %v", err))
	}

	stdType, errResult := parseStream(stream)
	if errResult != nil {
		return nil, nil, errResult
	}

	minutes := 5
	if sinceMinutes != nil && *sinceMinutes > 0 {
		minutes = *sinceMinutes
	}

	logCtx, cancel := context.WithCancel(ctx)
	since := time.Now().Add(-time.Duration(minutes) * time.Minute)
	events, err := containerSvc.LogsBetweenDates(logCtx, since, time.Now(), stdType)
	if err != nil {
		cancel()
		return nil, nil, errorResult(fmt.Sprintf("failed to read logs: %v", err))
	}

	return events, cancel, nil
}

// eventMessage extracts the JSON-encodable message payload for a log event.
func eventMessage(event *container.LogEvent) any {
	switch event.Type {
	case container.LogTypeGroup:
		if fragments, ok := event.Message.([]container.LogFragment); ok {
			lines := make([]string, len(fragments))
			for i, f := range fragments {
				lines[i] = f.Message
			}
			return lines
		}
		return event.RawMessage
	case container.LogTypeComplex:
		return event.Message
	default:
		return event.RawMessage
	}
}

func newLogEntry(event *container.LogEvent) mcpLogEntry {
	return mcpLogEntry{
		Timestamp: time.UnixMilli(event.Timestamp).UTC().Format(time.RFC3339Nano),
		Level:     event.Level,
		Stream:    event.Stream,
		Type:      string(event.Type),
		Message:   eventMessage(event),
	}
}

// collectLogEntries drains events into JSON-encodable entries. keep, when
// non-nil, filters which entries are included. Collection stops once the encoded
// size would exceed maxLogSize, in which case truncated is true. scanned counts
// every event pulled from the channel, matched or not.
func collectLogEntries(events <-chan *container.LogEvent, keep func(mcpLogEntry) bool) (entries []mcpLogEntry, scanned int, truncated bool) {
	totalSize := 0
	for event := range events {
		scanned++
		entry := newLogEntry(event)
		if keep != nil && !keep(entry) {
			continue
		}

		line, err := json.Marshal(entry)
		if err != nil {
			continue
		}

		totalSize += len(line) + 1
		if totalSize > maxLogSize {
			truncated = true
			break
		}

		entries = append(entries, entry)
	}
	return entries, scanned, truncated
}

// encodeLogEntries renders entries as newline-delimited JSON.
func encodeLogEntries(entries []mcpLogEntry) (string, error) {
	var sb strings.Builder
	encoder := json.NewEncoder(&sb)
	for _, entry := range entries {
		if err := encoder.Encode(entry); err != nil {
			return "", fmt.Errorf("failed to encode log entry: %w", err)
		}
	}
	return strings.TrimRight(sb.String(), "\n"), nil
}

func (s *Server) handleGetContainerLogs(ctx context.Context, _ *mcp.CallToolRequest, params *getContainerLogsParams) (*mcp.CallToolResult, any, error) {
	if params.Host == "" || params.ContainerID == "" {
		return errorResult("host and container_id are required"), nil, nil
	}

	events, cancel, errResult := s.fetchLogs(ctx, params.Host, params.ContainerID, params.Stream, params.SinceMinutes)
	if errResult != nil {
		return errResult, nil, nil
	}
	defer cancel()

	entries, _, _ := collectLogEntries(events, nil)
	if len(entries) == 0 {
		return textResult("(no logs in the specified time range)"), nil, nil
	}

	text, err := encodeLogEntries(entries)
	if err != nil {
		return nil, nil, err
	}
	return textResult(text), nil, nil
}

func (s *Server) handleSearchContainerLogs(ctx context.Context, _ *mcp.CallToolRequest, params *searchContainerLogsParams) (*mcp.CallToolResult, any, error) {
	if params.Host == "" || params.ContainerID == "" || params.Query == "" {
		return errorResult("host, container_id, and query are required"), nil, nil
	}

	caseSensitive := params.CaseSensitive != nil && *params.CaseSensitive
	query := params.Query
	if !caseSensitive {
		query = strings.ToLower(query)
	}

	events, cancel, errResult := s.fetchLogs(ctx, params.Host, params.ContainerID, params.Stream, params.SinceMinutes)
	if errResult != nil {
		return errResult, nil, nil
	}
	defer cancel()

	keep := func(entry mcpLogEntry) bool {
		haystack := messageToSearchString(entry.Message)
		if !caseSensitive {
			haystack = strings.ToLower(haystack)
		}
		return strings.Contains(haystack, query)
	}

	entries, scanned, truncated := collectLogEntries(events, keep)
	if len(entries) == 0 {
		return textResult(fmt.Sprintf("(no matches for %q in %d log entries scanned)", params.Query, scanned)), nil, nil
	}

	body, err := encodeLogEntries(entries)
	if err != nil {
		return nil, nil, err
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Found %d matches for %q (scanned %d entries):\n", len(entries), params.Query, scanned)
	sb.WriteString(body)
	if truncated {
		sb.WriteString("\n(results truncated at 1MB; narrow your query or time range to see more)")
	}

	return textResult(sb.String()), nil, nil
}

func messageToSearchString(msg any) string {
	switch v := msg.(type) {
	case string:
		return v
	case []string:
		return strings.Join(v, "\n")
	default:
		return fmt.Sprintf("%v", v)
	}
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

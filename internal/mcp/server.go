package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/amir20/dozzle/internal/auth"
	"github.com/amir20/dozzle/internal/cloud"
	"github.com/amir20/dozzle/internal/container"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rs/zerolog/log"
)

// HostService is the subset of web.HostService needed by the MCP server.
type HostService interface {
	FindContainer(host string, id string, labels container.ContainerLabels) (*container.ContainerService, error)
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

// resolveLabels returns the container label filter to apply for the requesting
// user. When authentication is enabled the user's own filter is carried on the
// request context; using it (instead of the server-global labels) keeps MCP
// reads scoped to the same containers the user can see through the normal log
// and stats paths. Falls back to the server-global labels when no per-user
// filter is present (e.g. auth disabled).
func (s *Server) resolveLabels(ctx context.Context) container.ContainerLabels {
	if user := auth.UserFromContext(ctx); user != nil && user.ContainerLabels.Exists() {
		return user.ContainerLabels
	}
	return s.labels
}

// Handler returns an http.Handler for the MCP streamable HTTP transport.
//
// It is stateless because the SDK only serves the 2026-07-28 protocol (and its
// server/discover probe) without sessions. Every tool here is a plain
// request/response, so nothing needs a session or an SSE stream.
func (s *Server) Handler() http.Handler {
	return mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return s.mcpServer
	}, &mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true})
}

// --- Tool Parameter Types ---

type listContainersParams struct {
	State         *string `json:"state,omitempty" jsonschema:"Filter by container state (running, exited, created, paused, dead). Leave empty for all."`
	IncludeLabels *bool   `json:"include_labels,omitempty" jsonschema:"Include each container's Docker labels. Labels can be large, so they are left out by default."`
}

type getContainerLogsParams struct {
	Host         string  `json:"host,omitempty" jsonschema:"The host ID or name where the container is running. Optional: leave it out to search every host."`
	ContainerID  string  `json:"container_id" jsonschema:"The container ID, short ID or name to get logs from. Use list_containers to find this."`
	SinceMinutes *int    `json:"since_minutes,omitempty" jsonschema:"Read the N minutes before end (or now). Defaults to 5. Ignored when start is set."`
	Start        *string `json:"start,omitempty" jsonschema:"Start of an absolute time range, RFC 3339 (e.g. 2026-10-28T04:20:00Z)."`
	End          *string `json:"end,omitempty" jsonschema:"End of the time range, RFC 3339. Defaults to now."`
	Level        *string `json:"level,omitempty" jsonschema:"Minimum level to return: trace, debug, info, warn, error or fatal. Entries below it, or with no detected level, are left out."`
	Limit        *int    `json:"limit,omitempty" jsonschema:"Return at most the newest N entries."`
	Stream       *string `json:"stream,omitempty" jsonschema:"Which output stream to read: stdout, stderr, or all. Defaults to all."`
}

type searchContainerLogsParams struct {
	Host          string  `json:"host,omitempty" jsonschema:"The host ID or name where the container is running. Optional: leave it out to search every host."`
	ContainerID   string  `json:"container_id" jsonschema:"The container ID, short ID or name to search logs from. Use list_containers to find this."`
	Query         string  `json:"query" jsonschema:"The search string to look for in log messages. Case-insensitive by default."`
	SinceMinutes  *int    `json:"since_minutes,omitempty" jsonschema:"Read the N minutes before end (or now). Defaults to 5. Ignored when start is set."`
	Start         *string `json:"start,omitempty" jsonschema:"Start of an absolute time range, RFC 3339 (e.g. 2026-10-28T04:20:00Z)."`
	End           *string `json:"end,omitempty" jsonschema:"End of the time range, RFC 3339. Defaults to now."`
	Level         *string `json:"level,omitempty" jsonschema:"Minimum level to return: trace, debug, info, warn, error or fatal. Entries below it, or with no detected level, are left out."`
	Limit         *int    `json:"limit,omitempty" jsonschema:"Return at most the newest N entries."`
	Stream        *string `json:"stream,omitempty" jsonschema:"Which output stream to search: stdout, stderr, or all. Defaults to all."`
	CaseSensitive *bool   `json:"case_sensitive,omitempty" jsonschema:"Whether to perform a case-sensitive search. Defaults to false."`
}

type getContainerStatsParams struct {
	Host        string `json:"host,omitempty" jsonschema:"The host ID or name where the container is running. Optional: leave it out to search every host."`
	ContainerID string `json:"container_id" jsonschema:"The container ID, short ID or name to get stats for. Use list_containers to find this."`
}

func (s *Server) registerTools() {
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "list_containers",
		Description: "List all Docker containers across all hosts. Returns container ID, name, image, state, host, and created time. Labels are only included with include_labels.",
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
	containers, errs := s.hostService.ListAllContainers(s.resolveLabels(ctx))
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

	includeLabels := params.IncludeLabels != nil && *params.IncludeLabels

	stateFilter := ""
	if params.State != nil {
		stateFilter = *params.State
	}

	results := []containerInfo{}
	for _, c := range containers {
		if stateFilter != "" && c.State != stateFilter {
			continue
		}
		info := containerInfo{
			ID:      c.ID,
			Name:    c.Name,
			Image:   c.Image,
			State:   c.State,
			Health:  c.Health,
			Host:    c.Host,
			Created: c.Created,
			Group:   c.Group,
		}
		if includeLabels {
			info.Labels = c.Labels
		}
		results = append(results, info)
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

// maxLogsScanned stops a read of a wide time range on a chatty container. The
// answer only keeps the newest 1MB, so without it every line in the window is
// streamed and encoded just to be dropped. See readLogs.
const maxLogsScanned = 100_000

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

// parseTimeRange turns since_minutes, start and end into the window to read.
// since_minutes (default 5) counts back from end, so "the 10 minutes before
// 04:40" needs only end; start overrides it.
func parseTimeRange(sinceMinutes *int, start, end *string) (time.Time, time.Time, *mcp.CallToolResult) {
	to := time.Now()
	if end != nil && *end != "" {
		t, err := time.Parse(time.RFC3339, *end)
		if err != nil {
			return time.Time{}, time.Time{}, errorResult(fmt.Sprintf("invalid end %q: must be RFC 3339, like 2026-10-28T04:40:00Z", *end))
		}
		to = t
	}

	if start != nil && *start != "" {
		from, err := time.Parse(time.RFC3339, *start)
		if err != nil {
			return time.Time{}, time.Time{}, errorResult(fmt.Sprintf("invalid start %q: must be RFC 3339, like 2026-10-28T04:20:00Z", *start))
		}
		if !from.Before(to) {
			return time.Time{}, time.Time{}, errorResult("start must be before end")
		}
		return from, to, nil
	}

	minutes := 5
	if sinceMinutes != nil && *sinceMinutes > 0 {
		minutes = *sinceMinutes
	}
	return to.Add(-time.Duration(minutes) * time.Minute), to, nil
}

// levelRank orders the levels logparse detects, lowest first.
var levelRank = map[string]int{"trace": 0, "debug": 1, "info": 2, "warn": 3, "error": 4, "fatal": 5}

// parseMinLevel resolves the optional level argument into a keep filter, or nil
// when no level was given. Entries with no detected level never pass a filter.
func parseMinLevel(level *string) (func(mcpLogEntry) bool, *mcp.CallToolResult) {
	if level == nil || *level == "" {
		return nil, nil
	}
	minRank, ok := levelRank[strings.ToLower(*level)]
	if !ok {
		return nil, errorResult(fmt.Sprintf("invalid level %q: must be trace, debug, info, warn, error or fatal", *level))
	}
	return func(entry mcpLogEntry) bool {
		rank, ok := levelRank[entry.Level]
		return ok && rank >= minRank
	}, nil
}

// findContainer resolves a container reference (id, short id or name) and an
// optional host (id or name) with the same resolver the cloud read tools use,
// so a name keeps working after a redeploy hands the container a new id. note
// is non-empty when a name matched several containers and one was picked.
func (s *Server) findContainer(ctx context.Context, host, ref string) (*container.ContainerService, string, *mcp.CallToolResult) {
	labels := s.resolveLabels(ctx)
	hostID, containerID, note, err := cloud.ResolveContainerRead(s.hostService, labels, ref, host)
	if errors.Is(err, cloud.ErrNoContainerMatch) {
		return nil, "", errorResult(fmt.Sprintf("container not found: no container matching %q; call list_containers to see available containers", ref))
	}
	if err != nil {
		return nil, "", errorResult(fmt.Sprintf("container not found: %v", err))
	}

	containerSvc, err := s.hostService.FindContainer(hostID, containerID, labels)
	if err != nil {
		return nil, "", errorResult(fmt.Sprintf("container not found: %v", err))
	}
	return containerSvc, note, nil
}

// fetchLogs finds the container and returns its log events between from and
// to, plus the resolver's note. On a user-facing
// failure it returns a non-nil error result; otherwise the caller must call
// cancel when done.
func (s *Server) fetchLogs(ctx context.Context, host, ref string, stream *string, from, to time.Time) (<-chan *container.LogEvent, string, context.CancelFunc, *mcp.CallToolResult) {
	stdType, errResult := parseStream(stream)
	if errResult != nil {
		return nil, "", nil, errResult
	}

	containerSvc, note, errResult := s.findContainer(ctx, host, ref)
	if errResult != nil {
		return nil, "", nil, errResult
	}

	logCtx, cancel := context.WithCancel(ctx)
	events, err := containerSvc.LogsBetweenDates(logCtx, from, to, stdType)
	if err != nil {
		cancel()
		return nil, "", nil, errorResult(fmt.Sprintf("failed to read logs: %v", err))
	}

	return events, note, cancel, nil
}

// withNote puts the resolver's note on its own line ahead of text.
func withNote(note, text string) string {
	if note == "" {
		return text
	}
	return note + "\n" + text
}

// eventMessage extracts the JSON-encodable message payload for a log event.
// ANSI colour codes are stripped: the UI renders them, but here they only cost
// space and break substring search across a colour change.
func eventMessage(event *container.LogEvent) any {
	switch event.Type {
	case container.LogTypeGroup:
		if fragments, ok := event.Message.([]container.LogFragment); ok {
			lines := make([]string, len(fragments))
			for i, f := range fragments {
				lines[i] = container.StripANSI(f.Message)
			}
			return lines
		}
		return container.StripANSI(event.RawMessage)
	case container.LogTypeComplex:
		return event.Message
	default:
		return container.StripANSI(event.RawMessage)
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
// non-nil, filters which entries are included. It keeps the newest entries:
// once the encoded size passes maxLogSize, or the count passes limit (when
// limit > 0), the oldest are dropped. scanned counts every event pulled from
// the channel and matched every entry keep accepted, so matched > len(entries)
// means the answer was cut. Reading stops after maxLogsScanned events, and
// stoppedAt is then the time of the first event left unread.
func collectLogEntries(events <-chan *container.LogEvent, keep func(mcpLogEntry) bool, limit int) (entries []mcpLogEntry, scanned, matched int, stoppedAt time.Time) {
	var sizes []int
	totalSize := 0
	for event := range events {
		if scanned == maxLogsScanned {
			stoppedAt = time.UnixMilli(event.Timestamp)
			break
		}
		scanned++
		entry := newLogEntry(event)
		if keep != nil && !keep(entry) {
			continue
		}

		line, err := json.Marshal(entry)
		if err != nil {
			continue
		}

		matched++
		entries = append(entries, entry)
		sizes = append(sizes, len(line)+1)
		totalSize += len(line) + 1
		for totalSize > maxLogSize || (limit > 0 && len(entries) > limit) {
			totalSize -= sizes[0]
			entries, sizes = entries[1:], sizes[1:]
		}
	}
	return entries, scanned, matched, stoppedAt
}

// logRead is what readLogs got from a range.
type logRead struct {
	entries          []mcpLogEntry
	scanned, matched int
	// tailFrom is set when the range was too big to read whole, and only the
	// part from tailFrom to its end was read.
	tailFrom time.Time
	// stoppedAt is set when even that part was cut short.
	stoppedAt time.Time
}

// readLogs collects the entries in [from, to]. Logs stream oldest first, so a
// range that holds more than maxLogsScanned entries is read again over its
// tail, as long as the span the first read got through. The answer is then
// still the newest entries, which is what a question about a container is
// almost always after.
func (s *Server) readLogs(ctx context.Context, host, ref string, stream *string, from, to time.Time, keep func(mcpLogEntry) bool, limit int) (logRead, string, *mcp.CallToolResult) {
	read := func(from time.Time) (logRead, string, *mcp.CallToolResult) {
		events, note, cancel, errResult := s.fetchLogs(ctx, host, ref, stream, from, to)
		if errResult != nil {
			return logRead{}, "", errResult
		}
		defer cancel()
		var r logRead
		r.entries, r.scanned, r.matched, r.stoppedAt = collectLogEntries(events, keep, limit)
		return r, note, nil
	}

	r, note, errResult := read(from)
	if errResult != nil || r.stoppedAt.IsZero() {
		return r, note, errResult
	}

	span := max(r.stoppedAt.Sub(from), time.Second)
	tailFrom := to.Add(-span)
	r, _, errResult = read(tailFrom)
	r.tailFrom = tailFrom
	return r, note, errResult
}

// notes are the lines that follow the entries: how much was cut, and how much
// of the range was read.
func (r logRead) notes() []string {
	var notes []string
	if !r.stoppedAt.IsZero() {
		// The entries are the oldest of the tail, not the newest.
		notes = append(notes, fmt.Sprintf("(the range holds more than %d entries; read %d from %s, and it continues from %s, so narrow it)",
			maxLogsScanned, r.scanned, r.tailFrom.UTC().Format(time.RFC3339), r.stoppedAt.UTC().Format(time.RFC3339)))
		return notes
	}
	if cut := truncationNote(len(r.entries), r.matched); cut != "" {
		notes = append(notes, cut)
	}
	if !r.tailFrom.IsZero() {
		notes = append(notes, fmt.Sprintf("(the range holds more than %d entries, so only the newest part, from %s, was read; set start and end to read earlier)",
			maxLogsScanned, r.tailFrom.UTC().Format(time.RFC3339)))
	}
	return notes
}

// truncationNote says how much of the answer was cut, or "" when nothing was.
func truncationNote(shown, matched int) string {
	if shown >= matched {
		return ""
	}
	return fmt.Sprintf("(showing the newest %d of %d entries; narrow the time range, or use limit or level)", shown, matched)
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
	if params.ContainerID == "" {
		return errorResult("container_id is required"), nil, nil
	}

	from, to, errResult := parseTimeRange(params.SinceMinutes, params.Start, params.End)
	if errResult != nil {
		return errResult, nil, nil
	}
	keep, errResult := parseMinLevel(params.Level)
	if errResult != nil {
		return errResult, nil, nil
	}

	r, note, errResult := s.readLogs(ctx, params.Host, params.ContainerID, params.Stream, from, to, keep, intValue(params.Limit))
	if errResult != nil {
		return errResult, nil, nil
	}

	if len(r.entries) == 0 {
		lines := append([]string{"(no logs in the specified time range)"}, r.notes()...)
		return textResult(withNote(note, strings.Join(lines, "\n"))), nil, nil
	}

	text, err := encodeLogEntries(r.entries)
	if err != nil {
		return nil, nil, err
	}
	for _, line := range r.notes() {
		text += "\n" + line
	}
	return textResult(withNote(note, text)), nil, nil
}

func (s *Server) handleSearchContainerLogs(ctx context.Context, _ *mcp.CallToolRequest, params *searchContainerLogsParams) (*mcp.CallToolResult, any, error) {
	if params.ContainerID == "" || params.Query == "" {
		return errorResult("container_id and query are required"), nil, nil
	}

	caseSensitive := params.CaseSensitive != nil && *params.CaseSensitive
	query := params.Query
	if !caseSensitive {
		query = strings.ToLower(query)
	}

	from, to, errResult := parseTimeRange(params.SinceMinutes, params.Start, params.End)
	if errResult != nil {
		return errResult, nil, nil
	}
	levelOK, errResult := parseMinLevel(params.Level)
	if errResult != nil {
		return errResult, nil, nil
	}

	keep := func(entry mcpLogEntry) bool {
		if levelOK != nil && !levelOK(entry) {
			return false
		}
		haystack := messageToSearchString(entry.Message)
		if !caseSensitive {
			haystack = strings.ToLower(haystack)
		}
		return strings.Contains(haystack, query)
	}

	r, note, errResult := s.readLogs(ctx, params.Host, params.ContainerID, params.Stream, from, to, keep, intValue(params.Limit))
	if errResult != nil {
		return errResult, nil, nil
	}

	if len(r.entries) == 0 {
		lines := append([]string{fmt.Sprintf("(no matches for %q in %d log entries scanned)", params.Query, r.scanned)}, r.notes()...)
		return textResult(withNote(note, strings.Join(lines, "\n"))), nil, nil
	}

	body, err := encodeLogEntries(r.entries)
	if err != nil {
		return nil, nil, err
	}

	var sb strings.Builder
	if note != "" {
		sb.WriteString(note + "\n")
	}
	fmt.Fprintf(&sb, "Found %d matches for %q (scanned %d entries):\n", r.matched, params.Query, r.scanned)
	sb.WriteString(body)
	for _, line := range r.notes() {
		sb.WriteString("\n" + line)
	}

	return textResult(sb.String()), nil, nil
}

func intValue(v *int) int {
	if v == nil {
		return 0
	}
	return *v
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
	if params.ContainerID == "" {
		return errorResult("container_id is required"), nil, nil
	}

	containerSvc, note, errResult := s.findContainer(ctx, params.Host, params.ContainerID)
	if errResult != nil {
		return errResult, nil, nil
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
		Host          string      `json:"host"`
		Note          string      `json:"note,omitempty"`
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
		Host:          c.Host,
		Note:          note,
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

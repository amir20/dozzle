package mcp

import (
	"context"
	"fmt"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/auth"
	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/utils"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockHostService struct {
	containers []container.Container
	hosts      []container.Host
	findErr    error
	listErrs   []error
	logEvents  []*container.LogEvent
	logErr     error

	// gotLabels records the last label filter passed to a lookup so tests can
	// assert the requesting user's filter is applied instead of the global one.
	gotLabels container.ContainerLabels
}

type stubClientService struct {
	container.ClientService
	events []*container.LogEvent
	err    error
}

func (s *stubClientService) LogsBetweenDates(ctx context.Context, _ container.Container, _ time.Time, _ time.Time, _ container.StdType) (<-chan *container.LogEvent, error) {
	if s.err != nil {
		return nil, s.err
	}
	ch := make(chan *container.LogEvent)
	go func() {
		defer close(ch)
		for _, e := range s.events {
			select {
			case ch <- e:
			case <-ctx.Done():
				return
			}
		}
	}()
	return ch, nil
}

func (s *stubClientService) LogHistogram(context.Context, container.Container, time.Time, time.Time, time.Duration) (container.LogHistogram, error) {
	return container.LogHistogram{}, nil
}

func (s *stubClientService) RawLogs(context.Context, container.Container, time.Time, time.Time, container.StdType) (io.ReadCloser, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockHostService) FindContainer(host string, id string, labels container.ContainerLabels) (*container.ContainerService, error) {
	m.gotLabels = labels
	if m.findErr != nil {
		return nil, m.findErr
	}
	for _, c := range m.containers {
		if c.ID == id && c.Host == host {
			stub := &stubClientService{events: m.logEvents, err: m.logErr}
			return container.NewContainerService(stub, c), nil
		}
	}
	return nil, assert.AnError
}

func (m *mockHostService) ListAllContainers(labels container.ContainerLabels) ([]container.Container, []error) {
	m.gotLabels = labels
	return m.containers, m.listErrs
}

// Hosts returns m.hosts, or one host per distinct container host when the test
// set none, since the name resolver checks a supplied host against this list.
func (m *mockHostService) Hosts() []container.Host {
	if m.hosts != nil {
		return m.hosts
	}
	hosts := []container.Host{}
	seen := map[string]bool{}
	for _, c := range m.containers {
		if !seen[c.Host] {
			seen[c.Host] = true
			hosts = append(hosts, container.Host{ID: c.Host, Name: c.Host})
		}
	}
	return hosts
}

func TestReadToolsUseRequestingUsersFilter(t *testing.T) {
	svc := &mockHostService{
		containers: []container.Container{{ID: "abc123", Name: "web", Host: "local"}},
	}
	// Server-global filter. A restricted user must NOT read through this.
	global := container.ContainerLabels{"com.example.scope": {"admin"}}
	s := NewServer(svc, global, "test")

	userLabels := container.ContainerLabels{"com.example.scope": {"tenant-a"}}
	ctx := auth.WithUser(context.Background(), auth.User{ContainerLabels: userLabels})

	// list_containers scopes to the user's filter.
	_, _, err := s.handleListContainers(ctx, nil, &listContainersParams{})
	require.NoError(t, err)
	assert.Equal(t, userLabels, svc.gotLabels, "list_containers should use the requesting user's filter")

	// get_container_stats (FindContainer path) scopes to the user's filter.
	svc.gotLabels = nil
	_, _, err = s.handleGetContainerStats(ctx, nil, &getContainerStatsParams{Host: "local", ContainerID: "abc123"})
	require.NoError(t, err)
	assert.Equal(t, userLabels, svc.gotLabels, "get_container_stats should use the requesting user's filter")

	// With no user on the context (e.g. auth disabled) it falls back to the global filter.
	svc.gotLabels = nil
	_, _, err = s.handleListContainers(context.Background(), nil, &listContainersParams{})
	require.NoError(t, err)
	assert.Equal(t, global, svc.gotLabels, "falls back to server-global filter when unauthenticated")
}

func TestListContainers(t *testing.T) {
	svc := &mockHostService{
		containers: []container.Container{
			{ID: "abc123", Name: "web", Image: "nginx:latest", State: "running", Host: "local"},
			{ID: "def456", Name: "db", Image: "postgres:15", State: "exited", Host: "local"},
		},
	}

	s := NewServer(svc, nil, "test")

	ctx := context.Background()
	ct, st := mcp.NewInMemoryTransports()

	_, err := s.mcpServer.Connect(ctx, st, nil)
	require.NoError(t, err)

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.0.1"}, nil)
	session, err := client.Connect(ctx, ct, nil)
	require.NoError(t, err)
	defer session.Close()

	// Test listing all containers
	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "list_containers",
		Arguments: map[string]any{},
	})
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Len(t, result.Content, 1)
	text := result.Content[0].(*mcp.TextContent).Text
	assert.Contains(t, text, "abc123")
	assert.Contains(t, text, "def456")

	// Test filtering by state
	result, err = session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "list_containers",
		Arguments: map[string]any{"state": "running"},
	})
	require.NoError(t, err)
	assert.False(t, result.IsError)
	text = result.Content[0].(*mcp.TextContent).Text
	assert.Contains(t, text, "abc123")
	assert.NotContains(t, text, "def456")
}

func TestListContainersPartialFailure(t *testing.T) {
	svc := &mockHostService{
		containers: []container.Container{
			{ID: "abc123", Name: "web", Image: "nginx:latest", State: "running", Host: "host1"},
		},
		listErrs: []error{nil, fmt.Errorf("host2 unreachable")},
	}

	s := NewServer(svc, nil, "test")

	ctx := context.Background()
	ct, st := mcp.NewInMemoryTransports()

	_, err := s.mcpServer.Connect(ctx, st, nil)
	require.NoError(t, err)

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.0.1"}, nil)
	session, err := client.Connect(ctx, ct, nil)
	require.NoError(t, err)
	defer session.Close()

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "list_containers",
		Arguments: map[string]any{},
	})
	require.NoError(t, err)
	assert.False(t, result.IsError)
	text := result.Content[0].(*mcp.TextContent).Text
	assert.Contains(t, text, "abc123")
}

func TestListHosts(t *testing.T) {
	svc := &mockHostService{
		hosts: []container.Host{
			{ID: "local", Name: "localhost", NCPU: 4, MemTotal: 8000000000, DockerVersion: "24.0", Type: "local", Available: true},
		},
	}

	s := NewServer(svc, nil, "test")

	ctx := context.Background()
	ct, st := mcp.NewInMemoryTransports()

	_, err := s.mcpServer.Connect(ctx, st, nil)
	require.NoError(t, err)

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.0.1"}, nil)
	session, err := client.Connect(ctx, ct, nil)
	require.NoError(t, err)
	defer session.Close()

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "list_hosts",
		Arguments: map[string]any{},
	})
	require.NoError(t, err)
	assert.False(t, result.IsError)
	text := result.Content[0].(*mcp.TextContent).Text
	assert.Contains(t, text, "localhost")
	assert.Contains(t, text, "24.0")
}

func TestGetContainerStats(t *testing.T) {
	stats := utils.RingBufferFrom(300, []container.ContainerStat{
		{ID: "abc123", CPUPercent: 25.5, MemoryPercent: 50.0, MemoryUsage: 1024000},
		{ID: "abc123", CPUPercent: 30.0, MemoryPercent: 55.0, MemoryUsage: 1100000},
	})

	svc := &mockHostService{
		containers: []container.Container{
			{ID: "abc123", Name: "web", Host: "local", Stats: stats, MemoryLimit: 2048000, CPULimit: 2.0},
		},
	}

	s := NewServer(svc, nil, "test")

	ctx := context.Background()
	ct, st := mcp.NewInMemoryTransports()

	_, err := s.mcpServer.Connect(ctx, st, nil)
	require.NoError(t, err)

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.0.1"}, nil)
	session, err := client.Connect(ctx, ct, nil)
	require.NoError(t, err)
	defer session.Close()

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "get_container_stats",
		Arguments: map[string]any{"host": "local", "container_id": "abc123"},
	})
	require.NoError(t, err)
	assert.False(t, result.IsError)
	text := result.Content[0].(*mcp.TextContent).Text
	assert.Contains(t, text, "25.5")
	assert.Contains(t, text, "2048000")
}

func TestGetContainerStatsNotFound(t *testing.T) {
	svc := &mockHostService{
		containers: []container.Container{},
	}

	s := NewServer(svc, nil, "test")

	ctx := context.Background()
	ct, st := mcp.NewInMemoryTransports()

	_, err := s.mcpServer.Connect(ctx, st, nil)
	require.NoError(t, err)

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.0.1"}, nil)
	session, err := client.Connect(ctx, ct, nil)
	require.NoError(t, err)
	defer session.Close()

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "get_container_stats",
		Arguments: map[string]any{"host": "local", "container_id": "nonexistent"},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestGetContainerLogsRequiredParams(t *testing.T) {
	svc := &mockHostService{}

	s := NewServer(svc, nil, "test")

	ctx := context.Background()
	ct, st := mcp.NewInMemoryTransports()

	_, err := s.mcpServer.Connect(ctx, st, nil)
	require.NoError(t, err)

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.0.1"}, nil)
	session, err := client.Connect(ctx, ct, nil)
	require.NoError(t, err)
	defer session.Close()

	// Missing required params
	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "get_container_logs",
		Arguments: map[string]any{},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestNewServerRegistersTools(t *testing.T) {
	svc := &mockHostService{}
	s := NewServer(svc, nil, "v1.0.0")

	ctx := context.Background()
	ct, st := mcp.NewInMemoryTransports()

	_, err := s.mcpServer.Connect(ctx, st, nil)
	require.NoError(t, err)

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.0.1"}, nil)
	session, err := client.Connect(ctx, ct, nil)
	require.NoError(t, err)
	defer session.Close()

	tools, err := session.ListTools(ctx, nil)
	require.NoError(t, err)

	toolNames := make([]string, len(tools.Tools))
	for i, tool := range tools.Tools {
		toolNames[i] = tool.Name
	}

	assert.Contains(t, toolNames, "list_containers")
	assert.Contains(t, toolNames, "get_container_logs")
	assert.Contains(t, toolNames, "search_container_logs")
	assert.Contains(t, toolNames, "list_hosts")
	assert.Contains(t, toolNames, "get_container_stats")
	assert.Len(t, tools.Tools, 5)
}

// Clients on the 2026-07-28 protocol (ChatGPT's tunnel among them) probe with
// server/discover and send every request without a session. The SDK only
// serves that version over a stateless handler.
func TestHandlerServesLatestProtocol(t *testing.T) {
	s := NewServer(&mockHostService{}, nil, "v1.0.0")
	srv := httptest.NewServer(s.Handler())
	defer srv.Close()

	ctx := context.Background()
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.0.1"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: srv.URL}, nil)
	require.NoError(t, err)
	defer session.Close()

	assert.Equal(t, "2026-07-28", session.InitializeResult().ProtocolVersion)

	tools, err := session.ListTools(ctx, nil)
	require.NoError(t, err)
	assert.Len(t, tools.Tools, 5)
}

func TestGetContainerLogs(t *testing.T) {
	now := time.Now()
	svc := &mockHostService{
		containers: []container.Container{
			{ID: "abc123", Name: "web", Host: "local"},
		},
		logEvents: []*container.LogEvent{
			{Timestamp: now.UnixMilli(), Level: "info", Stream: "stdout", Type: container.LogTypeSingle, RawMessage: "hello"},
			{Timestamp: now.UnixMilli(), Level: "error", Stream: "stderr", Type: container.LogTypeSingle, RawMessage: "boom"},
		},
	}

	s := NewServer(svc, nil, "test")

	ctx := context.Background()
	ct, st := mcp.NewInMemoryTransports()

	_, err := s.mcpServer.Connect(ctx, st, nil)
	require.NoError(t, err)

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.0.1"}, nil)
	session, err := client.Connect(ctx, ct, nil)
	require.NoError(t, err)
	defer session.Close()

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "get_container_logs",
		Arguments: map[string]any{"host": "local", "container_id": "abc123"},
	})
	require.NoError(t, err)
	assert.False(t, result.IsError)
	text := result.Content[0].(*mcp.TextContent).Text
	assert.Contains(t, text, "hello")
	assert.Contains(t, text, "boom")
}

func TestGetContainerLogsInvalidStream(t *testing.T) {
	svc := &mockHostService{
		containers: []container.Container{
			{ID: "abc123", Name: "web", Host: "local"},
		},
	}

	s := NewServer(svc, nil, "test")

	ctx := context.Background()
	ct, st := mcp.NewInMemoryTransports()

	_, err := s.mcpServer.Connect(ctx, st, nil)
	require.NoError(t, err)

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.0.1"}, nil)
	session, err := client.Connect(ctx, ct, nil)
	require.NoError(t, err)
	defer session.Close()

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "get_container_logs",
		Arguments: map[string]any{"host": "local", "container_id": "abc123", "stream": "bogus"},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
	text := result.Content[0].(*mcp.TextContent).Text
	assert.Contains(t, text, "invalid stream")
}

func TestSearchContainerLogs(t *testing.T) {
	now := time.Now()
	svc := &mockHostService{
		containers: []container.Container{
			{ID: "abc123", Name: "web", Host: "local"},
		},
		logEvents: []*container.LogEvent{
			{Timestamp: now.UnixMilli(), Level: "info", Stream: "stdout", Type: container.LogTypeSingle, RawMessage: "Request received for /home"},
			{Timestamp: now.UnixMilli(), Level: "error", Stream: "stderr", Type: container.LogTypeSingle, RawMessage: "Failed to process payment"},
			{Timestamp: now.UnixMilli(), Level: "info", Stream: "stdout", Type: container.LogTypeSingle, RawMessage: "Request received for /about"},
			{Timestamp: now.UnixMilli(), Level: "error", Stream: "stderr", Type: container.LogTypeSingle, RawMessage: "payment gateway timeout"},
		},
	}

	s := NewServer(svc, nil, "test")

	ctx := context.Background()
	ct, st := mcp.NewInMemoryTransports()

	_, err := s.mcpServer.Connect(ctx, st, nil)
	require.NoError(t, err)

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.0.1"}, nil)
	session, err := client.Connect(ctx, ct, nil)
	require.NoError(t, err)
	defer session.Close()

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "search_container_logs",
		Arguments: map[string]any{"host": "local", "container_id": "abc123", "query": "payment"},
	})
	require.NoError(t, err)
	assert.False(t, result.IsError)
	text := result.Content[0].(*mcp.TextContent).Text
	assert.Contains(t, text, "Failed to process payment")
	assert.Contains(t, text, "payment gateway timeout")
	assert.NotContains(t, text, "Request received for /home")
	assert.Contains(t, text, "Found 2 matches")
}

func TestSearchContainerLogsCaseSensitive(t *testing.T) {
	now := time.Now()
	svc := &mockHostService{
		containers: []container.Container{
			{ID: "abc123", Name: "web", Host: "local"},
		},
		logEvents: []*container.LogEvent{
			{Timestamp: now.UnixMilli(), Level: "info", Stream: "stdout", Type: container.LogTypeSingle, RawMessage: "Error: something went wrong"},
			{Timestamp: now.UnixMilli(), Level: "info", Stream: "stdout", Type: container.LogTypeSingle, RawMessage: "error: lowercase message"},
		},
	}

	s := NewServer(svc, nil, "test")

	ctx := context.Background()
	ct, st := mcp.NewInMemoryTransports()

	_, err := s.mcpServer.Connect(ctx, st, nil)
	require.NoError(t, err)

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.0.1"}, nil)
	session, err := client.Connect(ctx, ct, nil)
	require.NoError(t, err)
	defer session.Close()

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "search_container_logs",
		Arguments: map[string]any{"host": "local", "container_id": "abc123", "query": "Error", "case_sensitive": true},
	})
	require.NoError(t, err)
	assert.False(t, result.IsError)
	text := result.Content[0].(*mcp.TextContent).Text
	assert.Contains(t, text, "Error: something went wrong")
	assert.NotContains(t, text, "error: lowercase message")
	assert.Contains(t, text, "Found 1 match")
}

func TestSearchContainerLogsNoMatches(t *testing.T) {
	now := time.Now()
	svc := &mockHostService{
		containers: []container.Container{
			{ID: "abc123", Name: "web", Host: "local"},
		},
		logEvents: []*container.LogEvent{
			{Timestamp: now.UnixMilli(), Level: "info", Stream: "stdout", Type: container.LogTypeSingle, RawMessage: "everything is fine"},
			{Timestamp: now.UnixMilli(), Level: "info", Stream: "stdout", Type: container.LogTypeSingle, RawMessage: "all good here"},
		},
	}

	s := NewServer(svc, nil, "test")

	ctx := context.Background()
	ct, st := mcp.NewInMemoryTransports()

	_, err := s.mcpServer.Connect(ctx, st, nil)
	require.NoError(t, err)

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.0.1"}, nil)
	session, err := client.Connect(ctx, ct, nil)
	require.NoError(t, err)
	defer session.Close()

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "search_container_logs",
		Arguments: map[string]any{"host": "local", "container_id": "abc123", "query": "error"},
	})
	require.NoError(t, err)
	assert.False(t, result.IsError)
	text := result.Content[0].(*mcp.TextContent).Text
	assert.Contains(t, text, "no matches")
	assert.Contains(t, text, "2 log entries scanned")
}

func TestSearchContainerLogsRequiredParams(t *testing.T) {
	svc := &mockHostService{}

	s := NewServer(svc, nil, "test")

	ctx := context.Background()
	ct, st := mcp.NewInMemoryTransports()

	_, err := s.mcpServer.Connect(ctx, st, nil)
	require.NoError(t, err)

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.0.1"}, nil)
	session, err := client.Connect(ctx, ct, nil)
	require.NoError(t, err)
	defer session.Close()

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "search_container_logs",
		Arguments: map[string]any{},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestSearchContainerLogsTruncates(t *testing.T) {
	now := time.Now()
	big := strings.Repeat("match ", 200) // ~1.2KB per entry
	events := make([]*container.LogEvent, 2000)
	for i := range events {
		events[i] = &container.LogEvent{Timestamp: now.UnixMilli(), Level: "info", Stream: "stdout", Type: container.LogTypeSingle, RawMessage: big}
	}
	svc := &mockHostService{
		containers: []container.Container{{ID: "abc123", Name: "web", Host: "local"}},
		logEvents:  events,
	}

	s := NewServer(svc, nil, "test")

	ctx := context.Background()
	ct, st := mcp.NewInMemoryTransports()

	_, err := s.mcpServer.Connect(ctx, st, nil)
	require.NoError(t, err)

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.0.1"}, nil)
	session, err := client.Connect(ctx, ct, nil)
	require.NoError(t, err)
	defer session.Close()

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "search_container_logs",
		Arguments: map[string]any{"host": "local", "container_id": "abc123", "query": "match"},
	})
	require.NoError(t, err)
	assert.False(t, result.IsError)
	text := result.Content[0].(*mcp.TextContent).Text
	assert.Contains(t, text, "showing the newest")
}

func connectTestClient(t *testing.T, s *Server) *mcp.ClientSession {
	t.Helper()
	ctx := context.Background()
	ct, st := mcp.NewInMemoryTransports()
	_, err := s.mcpServer.Connect(ctx, st, nil)
	require.NoError(t, err)
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.0.1"}, nil)
	session, err := client.Connect(ctx, ct, nil)
	require.NoError(t, err)
	t.Cleanup(func() { session.Close() })
	return session
}

func callText(t *testing.T, session *mcp.ClientSession, name string, args map[string]any) (string, bool) {
	t.Helper()
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	require.NoError(t, err)
	return result.Content[0].(*mcp.TextContent).Text, result.IsError
}

func TestContainerToolsAcceptNameWithoutHost(t *testing.T) {
	svc := &mockHostService{
		containers: []container.Container{
			{ID: "abc123", Name: "web", State: "running", Host: "local"},
			{ID: "def456", Name: "db", State: "running", Host: "local"},
		},
		logEvents: []*container.LogEvent{
			{Type: container.LogTypeSingle, RawMessage: "hello from web", Timestamp: 1},
		},
	}
	session := connectTestClient(t, NewServer(svc, nil, "test"))

	text, isErr := callText(t, session, "get_container_logs", map[string]any{"container_id": "web"})
	assert.False(t, isErr, text)
	assert.Contains(t, text, "hello from web")

	text, isErr = callText(t, session, "search_container_logs", map[string]any{"container_id": "web", "query": "hello"})
	assert.False(t, isErr, text)
	assert.Contains(t, text, "Found 1 match")

	text, isErr = callText(t, session, "get_container_stats", map[string]any{"container_id": "db", "host": "local"})
	assert.False(t, isErr, text)
	assert.Contains(t, text, `"containerId":"def456"`)

	text, isErr = callText(t, session, "get_container_stats", map[string]any{"container_id": "nope"})
	assert.True(t, isErr)
	assert.Contains(t, text, "list_containers")
	assert.NotContains(t, text, "find_containers")
}

func TestContainerToolsNoteAmbiguousName(t *testing.T) {
	svc := &mockHostService{
		containers: []container.Container{
			{ID: "aaa111", Name: "api-1", State: "exited", Host: "local", FinishedAt: time.Unix(100, 0)},
			{ID: "bbb222", Name: "api-2", State: "exited", Host: "local", FinishedAt: time.Unix(200, 0)},
		},
	}
	session := connectTestClient(t, NewServer(svc, nil, "test"))

	text, isErr := callText(t, session, "get_container_stats", map[string]any{"container_id": "api"})
	assert.False(t, isErr, text)
	assert.Contains(t, text, `"containerId":"bbb222"`)
	assert.Contains(t, text, "api-1")
}

func TestListContainersOmitsLabelsByDefault(t *testing.T) {
	svc := &mockHostService{
		containers: []container.Container{
			{ID: "abc123", Name: "web", Host: "local", Labels: map[string]string{"traefik.http.routers.web.rule": "Host(`x`)"}},
		},
	}
	session := connectTestClient(t, NewServer(svc, nil, "test"))

	text, _ := callText(t, session, "list_containers", map[string]any{})
	assert.NotContains(t, text, "traefik")

	text, _ = callText(t, session, "list_containers", map[string]any{"include_labels": true})
	assert.Contains(t, text, "traefik")
}

func TestGetContainerLogsKeepsNewestWithLimitAndLevel(t *testing.T) {
	svc := &mockHostService{
		containers: []container.Container{{ID: "abc123", Name: "web", Host: "local"}},
		logEvents: []*container.LogEvent{
			{Type: container.LogTypeSingle, RawMessage: "first", Level: "error", Timestamp: 1},
			{Type: container.LogTypeSingle, RawMessage: "second", Level: "info", Timestamp: 2},
			{Type: container.LogTypeSingle, RawMessage: "third", Level: "warn", Timestamp: 3},
			{Type: container.LogTypeSingle, RawMessage: "\x1b[31mfourth\x1b[0m", Level: "error", Timestamp: 4},
		},
	}
	session := connectTestClient(t, NewServer(svc, nil, "test"))

	text, isErr := callText(t, session, "get_container_logs", map[string]any{"container_id": "web", "limit": 2})
	assert.False(t, isErr, text)
	assert.NotContains(t, text, "first")
	assert.Contains(t, text, "third")
	assert.Contains(t, text, `"message":"fourth"`, "ANSI codes are stripped")
	assert.Contains(t, text, "showing the newest 2 of 4 entries")

	text, isErr = callText(t, session, "get_container_logs", map[string]any{"container_id": "web", "level": "warn"})
	assert.False(t, isErr, text)
	assert.NotContains(t, text, "second")
	assert.Contains(t, text, "first")
	assert.Contains(t, text, "third")
	assert.NotContains(t, text, "showing the newest")

	text, isErr = callText(t, session, "get_container_logs", map[string]any{"container_id": "web", "level": "loud"})
	assert.True(t, isErr)
	assert.Contains(t, text, "invalid level")
}

func TestSearchMatchesAcrossColourChange(t *testing.T) {
	svc := &mockHostService{
		containers: []container.Container{{ID: "abc123", Name: "web", Host: "local"}},
		logEvents: []*container.LogEvent{
			{Type: container.LogTypeSingle, RawMessage: "payment \x1b[31mfailed\x1b[0m", Timestamp: 1},
		},
	}
	session := connectTestClient(t, NewServer(svc, nil, "test"))

	text, isErr := callText(t, session, "search_container_logs", map[string]any{"container_id": "web", "query": "payment failed"})
	assert.False(t, isErr, text)
	assert.Contains(t, text, "Found 1 match")
}

func TestParseTimeRange(t *testing.T) {
	str := func(s string) *string { return &s }
	ten := 10

	from, to, errResult := parseTimeRange(nil, str("2026-10-28T04:20:00Z"), str("2026-10-28T04:40:00Z"))
	require.Nil(t, errResult)
	assert.Equal(t, time.Date(2026, 10, 28, 4, 20, 0, 0, time.UTC), from.UTC())
	assert.Equal(t, time.Date(2026, 10, 28, 4, 40, 0, 0, time.UTC), to.UTC())

	from, to, errResult = parseTimeRange(&ten, nil, str("2026-10-28T04:40:00Z"))
	require.Nil(t, errResult)
	assert.Equal(t, 10*time.Minute, to.Sub(from))

	from, to, errResult = parseTimeRange(nil, nil, nil)
	require.Nil(t, errResult)
	assert.Equal(t, 5*time.Minute, to.Sub(from))

	_, _, errResult = parseTimeRange(nil, str("2026-10-28T04:40:00Z"), str("2026-10-28T04:20:00Z"))
	assert.NotNil(t, errResult)

	_, _, errResult = parseTimeRange(nil, str("yesterday"), nil)
	assert.NotNil(t, errResult)
}

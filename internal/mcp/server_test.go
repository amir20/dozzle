package mcp

import (
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/container"
	container_support "github.com/amir20/dozzle/internal/support/container"
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
}

type stubClientService struct {
	container_support.ClientService
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

func (s *stubClientService) RawLogs(context.Context, container.Container, time.Time, time.Time, container.StdType) (io.ReadCloser, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockHostService) FindContainer(host string, id string, labels container.ContainerLabels) (*container_support.ContainerService, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	for _, c := range m.containers {
		if c.ID == id && c.Host == host {
			stub := &stubClientService{events: m.logEvents, err: m.logErr}
			return container_support.NewContainerService(stub, c), nil
		}
	}
	return nil, assert.AnError
}

func (m *mockHostService) ListAllContainers(labels container.ContainerLabels) ([]container.Container, []error) {
	return m.containers, m.listErrs
}

func (m *mockHostService) Hosts() []container.Host {
	return m.hosts
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
	assert.Contains(t, toolNames, "list_hosts")
	assert.Contains(t, toolNames, "get_container_stats")
	assert.Len(t, tools.Tools, 4)
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

package cloud

import (
	"context"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/container"
	container_support "github.com/amir20/dozzle/internal/support/container"
	pb "github.com/amir20/dozzle/proto/cloud"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestParseStreamArgs_Valid(t *testing.T) {
	args, re, err := parseStreamArgs(`{"container_id":"abc","host_id":"host1","level":"error","regex":"foo.*bar"}`)
	assert.NoError(t, err)
	assert.Equal(t, "abc", args.ContainerID)
	assert.Equal(t, "host1", args.Host)
	assert.Equal(t, "error", args.Level)
	assert.NotNil(t, re)
}

func TestParseStreamArgs_MissingRequired(t *testing.T) {
	_, _, err := parseStreamArgs(`{"container_id":"abc"}`)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "container_id and host_id are required")
}

func TestParseStreamArgs_InvalidJSON(t *testing.T) {
	_, _, err := parseStreamArgs(`{invalid`)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse arguments")
}

func TestParseStreamArgs_InvalidRegex(t *testing.T) {
	_, _, err := parseStreamArgs(`{"container_id":"abc","host_id":"h","regex":"[invalid"}`)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid regex pattern")
}

func TestParseStreamArgs_NoRegex(t *testing.T) {
	args, re, err := parseStreamArgs(`{"container_id":"abc","host_id":"h"}`)
	assert.NoError(t, err)
	assert.Equal(t, "abc", args.ContainerID)
	assert.Nil(t, re)
}

func TestMatchesFilters_NoFilters(t *testing.T) {
	event := &container.LogEvent{RawMessage: "hello world", Level: "info"}
	args := &fetchLogsArgs{}
	msg, ok := matchesFilters(event, args, nil)
	assert.True(t, ok)
	assert.Equal(t, "hello world", msg)
}

func TestMatchesFilters_LevelMismatch(t *testing.T) {
	event := &container.LogEvent{RawMessage: "hello", Level: "info"}
	args := &fetchLogsArgs{Level: "error"}
	_, ok := matchesFilters(event, args, nil)
	assert.False(t, ok)
}

func TestMatchesFilters_LevelMatch(t *testing.T) {
	event := &container.LogEvent{RawMessage: "hello", Level: "ERROR"}
	args := &fetchLogsArgs{Level: "error"}
	msg, ok := matchesFilters(event, args, nil)
	assert.True(t, ok)
	assert.Equal(t, "hello", msg)
}

func TestMatchesFilters_QueryMatch(t *testing.T) {
	event := &container.LogEvent{RawMessage: "Hello World", Level: "info"}
	args := &fetchLogsArgs{Query: "hello"}
	msg, ok := matchesFilters(event, args, nil)
	assert.True(t, ok)
	assert.Equal(t, "Hello World", msg)
}

func TestMatchesFilters_QueryMismatch(t *testing.T) {
	event := &container.LogEvent{RawMessage: "Hello World", Level: "info"}
	args := &fetchLogsArgs{Query: "missing"}
	_, ok := matchesFilters(event, args, nil)
	assert.False(t, ok)
}

func TestMatchesFilters_RegexMatch(t *testing.T) {
	event := &container.LogEvent{RawMessage: "error: connection refused", Level: "error"}
	args := &fetchLogsArgs{}
	re := regexp.MustCompile(`error:.*refused`)
	msg, ok := matchesFilters(event, args, re)
	assert.True(t, ok)
	assert.Equal(t, "error: connection refused", msg)
}

func TestMatchesFilters_RegexMismatch(t *testing.T) {
	event := &container.LogEvent{RawMessage: "info: all good", Level: "info"}
	args := &fetchLogsArgs{}
	re := regexp.MustCompile(`error:.*refused`)
	_, ok := matchesFilters(event, args, re)
	assert.False(t, ok)
}

func TestMatchesFilters_FallbackToMessage(t *testing.T) {
	event := &container.LogEvent{RawMessage: "", Message: "fallback msg", Level: "info"}
	args := &fetchLogsArgs{}
	msg, ok := matchesFilters(event, args, nil)
	assert.True(t, ok)
	assert.Equal(t, "fallback msg", msg)
}

func TestMatchesFilters_InverseRegex(t *testing.T) {
	re := regexp.MustCompile(`error`)

	tests := []struct {
		name    string
		message string
		level   string
		args    fetchLogsArgs
		re      *regexp.Regexp
		wantOk  bool
	}{
		{
			name:    "normal mode: regex matches",
			message: "error: connection refused",
			level:   "info",
			args:    fetchLogsArgs{},
			re:      re,
			wantOk:  true,
		},
		{
			name:    "normal mode: regex no match",
			message: "info: all good",
			level:   "info",
			args:    fetchLogsArgs{},
			re:      re,
			wantOk:  false,
		},
		{
			name:    "inverse mode: regex matches is excluded",
			message: "error: connection refused",
			level:   "info",
			args:    fetchLogsArgs{Inverse: true},
			re:      re,
			wantOk:  false,
		},
		{
			name:    "inverse mode: regex no match is included",
			message: "info: all good",
			level:   "info",
			args:    fetchLogsArgs{Inverse: true},
			re:      re,
			wantOk:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := &container.LogEvent{RawMessage: tt.message, Level: tt.level}
			_, ok := matchesFilters(event, &tt.args, tt.re)
			assert.Equal(t, tt.wantOk, ok)
		})
	}
}

func TestMatchesFilters_InverseQuery(t *testing.T) {
	tests := []struct {
		name    string
		message string
		args    fetchLogsArgs
		wantOk  bool
	}{
		{
			name:    "normal query match",
			message: "hello world",
			args:    fetchLogsArgs{Query: "hello"},
			wantOk:  true,
		},
		{
			name:    "normal query no match",
			message: "foo bar",
			args:    fetchLogsArgs{Query: "hello"},
			wantOk:  false,
		},
		{
			name:    "inverse query: match is excluded",
			message: "hello world",
			args:    fetchLogsArgs{Query: "hello", Inverse: true},
			wantOk:  false,
		},
		{
			name:    "inverse query: no match is included",
			message: "foo bar",
			args:    fetchLogsArgs{Query: "hello", Inverse: true},
			wantOk:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := &container.LogEvent{RawMessage: tt.message, Level: "info"}
			_, ok := matchesFilters(event, &tt.args, nil)
			assert.Equal(t, tt.wantOk, ok)
		})
	}
}

// StreamMockClientService extends MockClientService to allow controllable StreamLogs behavior
type StreamMockClientService struct {
	MockClientService
	streamFunc func(ctx context.Context, c container.Container, from time.Time, stdTypes container.StdType, events chan<- *container.LogEvent) error
}

func (m *StreamMockClientService) StreamLogs(ctx context.Context, c container.Container, from time.Time, stdTypes container.StdType, events chan<- *container.LogEvent) error {
	if m.streamFunc != nil {
		return m.streamFunc(ctx, c, from, stdTypes, events)
	}
	return nil
}

func TestExecuteStreamLogs_BasicFlow(t *testing.T) {
	mockClient := &StreamMockClientService{}
	mockClient.streamFunc = func(ctx context.Context, _ container.Container, _ time.Time, _ container.StdType, events chan<- *container.LogEvent) error {
		events <- &container.LogEvent{RawMessage: "line 1", Level: "info", Stream: "stdout", Timestamp: 1000}
		events <- &container.LogEvent{RawMessage: "line 2", Level: "info", Stream: "stdout", Timestamp: 2000}
		return nil
	}

	cs := container_support.NewContainerService(mockClient, container.Container{ID: "abc123", Name: "test-container"})
	mockHost := &MockHostService{}
	mockHost.On("FindContainer", "host1", "abc123", container.ContainerLabels(nil)).Return(cs, nil)

	var mu sync.Mutex
	var responses []*pb.ToolResponse
	send := func(resp *pb.ToolResponse) error {
		mu.Lock()
		defer mu.Unlock()
		responses = append(responses, resp)
		return nil
	}

	argsJSON := `{"container_id":"abc123","host_id":"host1"}`
	err := executeStreamLogs(context.Background(), "req1", argsJSON, ToolDeps{HostService: mockHost}, send)
	assert.NoError(t, err)

	mu.Lock()
	defer mu.Unlock()

	// Should have at least one response with end_stream=true
	assert.NotEmpty(t, responses)
	lastResp := responses[len(responses)-1]
	callTool := lastResp.GetCallTool()
	assert.NotNil(t, callTool)
	assert.True(t, callTool.EndStream)
}

func TestExecuteStreamLogs_WithLevelFilter(t *testing.T) {
	mockClient := &StreamMockClientService{}
	mockClient.streamFunc = func(ctx context.Context, _ container.Container, _ time.Time, _ container.StdType, events chan<- *container.LogEvent) error {
		events <- &container.LogEvent{RawMessage: "info msg", Level: "info", Stream: "stdout", Timestamp: 1000}
		events <- &container.LogEvent{RawMessage: "error msg", Level: "error", Stream: "stderr", Timestamp: 2000}
		events <- &container.LogEvent{RawMessage: "another info", Level: "info", Stream: "stdout", Timestamp: 3000}
		return nil
	}

	cs := container_support.NewContainerService(mockClient, container.Container{ID: "abc123", Name: "test-container"})
	mockHost := &MockHostService{}
	mockHost.On("FindContainer", "host1", "abc123", container.ContainerLabels(nil)).Return(cs, nil)

	var mu sync.Mutex
	var responses []*pb.ToolResponse
	send := func(resp *pb.ToolResponse) error {
		mu.Lock()
		defer mu.Unlock()
		responses = append(responses, resp)
		return nil
	}

	argsJSON := `{"container_id":"abc123","host_id":"host1","level":"error"}`
	err := executeStreamLogs(context.Background(), "req1", argsJSON, ToolDeps{HostService: mockHost}, send)
	assert.NoError(t, err)

	mu.Lock()
	defer mu.Unlock()

	// Collect all entries across all responses
	var allEntries []*pb.LogEntry
	for _, resp := range responses {
		ct := resp.GetCallTool()
		if ct != nil && ct.GetFetchLogs() != nil {
			allEntries = append(allEntries, ct.GetFetchLogs().Entries...)
		}
	}

	assert.Len(t, allEntries, 1)
	assert.Equal(t, "error msg", allEntries[0].Message)
}

func TestExecuteStreamLogs_CancelContext(t *testing.T) {
	mockClient := &StreamMockClientService{}
	mockClient.streamFunc = func(ctx context.Context, _ container.Container, _ time.Time, _ container.StdType, events chan<- *container.LogEvent) error {
		// Block until context is cancelled
		<-ctx.Done()
		return ctx.Err()
	}

	cs := container_support.NewContainerService(mockClient, container.Container{ID: "abc123", Name: "test-container"})
	mockHost := &MockHostService{}
	mockHost.On("FindContainer", "host1", "abc123", container.ContainerLabels(nil)).Return(cs, nil)

	var mu sync.Mutex
	var responses []*pb.ToolResponse
	send := func(resp *pb.ToolResponse) error {
		mu.Lock()
		defer mu.Unlock()
		responses = append(responses, resp)
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- executeStreamLogs(ctx, "req1", `{"container_id":"abc123","host_id":"host1"}`, ToolDeps{HostService: mockHost}, send)
	}()

	// Give it a moment to start, then cancel
	time.Sleep(50 * time.Millisecond)
	cancel()

	err := <-done
	assert.ErrorIs(t, err, context.Canceled)

	mu.Lock()
	defer mu.Unlock()

	// Should have sent end_stream
	if len(responses) > 0 {
		lastResp := responses[len(responses)-1]
		callTool := lastResp.GetCallTool()
		assert.True(t, callTool.EndStream)
	}
}

func TestExecuteStreamLogs_InvalidArgs(t *testing.T) {
	mockHost := &MockHostService{}
	send := func(resp *pb.ToolResponse) error { return nil }

	err := executeStreamLogs(context.Background(), "req1", `{invalid`, ToolDeps{HostService: mockHost}, send)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse arguments")
}

func TestExecuteStreamLogs_ContainerNotFound(t *testing.T) {
	mockHost := &MockHostService{}
	mockHost.On("FindContainer", "host1", "missing", container.ContainerLabels(nil)).Return(nil, assert.AnError)

	send := func(resp *pb.ToolResponse) error { return nil }

	err := executeStreamLogs(context.Background(), "req1", `{"container_id":"missing","host_id":"host1"}`, ToolDeps{HostService: mockHost}, send)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "container not found")
}

func TestExecuteStreamLogs_BatchingAt50(t *testing.T) {
	mockClient := &StreamMockClientService{}
	mockClient.streamFunc = func(ctx context.Context, _ container.Container, _ time.Time, _ container.StdType, events chan<- *container.LogEvent) error {
		for i := range 60 {
			events <- &container.LogEvent{RawMessage: "msg", Level: "info", Stream: "stdout", Timestamp: int64(i)}
		}
		return nil
	}

	cs := container_support.NewContainerService(mockClient, container.Container{ID: "abc123", Name: "test-container"})
	mockHost := &MockHostService{}
	mockHost.On("FindContainer", mock.Anything, mock.Anything, mock.Anything).Return(cs, nil)

	var mu sync.Mutex
	var responses []*pb.ToolResponse
	send := func(resp *pb.ToolResponse) error {
		mu.Lock()
		defer mu.Unlock()
		responses = append(responses, resp)
		return nil
	}

	err := executeStreamLogs(context.Background(), "req1", `{"container_id":"abc123","host_id":"host1"}`, ToolDeps{HostService: mockHost}, send)
	assert.NoError(t, err)

	mu.Lock()
	defer mu.Unlock()

	// First batch should have 50 entries (batch at batchSize=50)
	assert.GreaterOrEqual(t, len(responses), 2, "should have at least 2 responses (batch + end_stream)")
	firstBatch := responses[0].GetCallTool().GetFetchLogs()
	assert.Len(t, firstBatch.Entries, 50)
	assert.False(t, responses[0].GetCallTool().EndStream)
}

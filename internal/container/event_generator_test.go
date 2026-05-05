package container

import (
	"context"
	"io"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

func TestEventGenerator_Events_tty(t *testing.T) {
	input := "example input"

	g := NewEventGenerator(context.Background(), makeFakeReader(input, STDOUT), Container{Tty: true})
	event := <-g.Events

	require.NotNil(t, event, "Expected event to not be nil, but got nil")
	assert.Equal(t, input, event.Message)
	assert.Equal(t, LogTypeSingle, event.Type)
}

func TestEventGenerator_Events_non_tty(t *testing.T) {
	input := "example input"

	g := NewEventGenerator(context.Background(), makeFakeReader(input, STDOUT), Container{Tty: false})
	event := <-g.Events

	require.NotNil(t, event, "Expected event to not be nil, but got nil")
	assert.Equal(t, input, event.Message)
	assert.Equal(t, LogTypeSingle, event.Type)
}

func TestEventGenerator_Events_non_tty_close_channel(t *testing.T) {
	input := "example input"

	g := NewEventGenerator(context.Background(), makeFakeReader(input, STDOUT), Container{Tty: false})
	<-g.Events
	_, ok := <-g.Events

	assert.False(t, ok, "Expected channel to be closed")
}

func TestEventGenerator_Events_routines_done(t *testing.T) {
	input := "example input"

	g := NewEventGenerator(context.Background(), makeFakeReader(input, STDOUT), Container{Tty: false})
	<-g.Events
	assert.False(t, waitTimeout(&g.wg, 1*time.Second), "Expected routines to be done")
}

type mockLogReader struct {
	messages []string
	types    []StdType
	i        int
}

func (m *mockLogReader) Read() (string, StdType, error) {
	if m.i >= len(m.messages) {
		return "", 0, io.EOF
	}
	m.i++
	return m.messages[m.i-1], m.types[m.i-1], nil
}

func makeFakeReader(message string, stream StdType) LogReader {
	return &mockLogReader{
		messages: []string{message},
		types:    []StdType{stream},
	}
}

func waitTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()
	select {
	case <-c:
		return false // completed normally
	case <-time.After(timeout):
		return true // timed out
	}
}

func Test_createEvent(t *testing.T) {
	data := orderedmap.New[string, any]()
	data.Set("xyz", "value")
	data.Set("abc", "value2")
	type args struct {
		message string
	}
	tests := []struct {
		name string
		args args
		want *LogEvent
	}{
		{
			name: "empty message",
			args: args{
				message: "",
			},
			want: &LogEvent{
				Message: "",
			},
		}, {
			name: "simple json message",
			args: args{
				message: "2020-05-13T18:55:37.772853839Z {\"xyz\": \"value\", \"abc\": \"value2\"}",
			},
			want: &LogEvent{
				Message: data,
			},
		},
		{
			name: "invalid json message",
			args: args{
				message: "2020-05-13T18:55:37.772853839Z {\"key\"}",
			},
			want: &LogEvent{
				Message: "{\"key\"}",
			},
		},
		{
			name: "invalid json message",
			args: args{
				message: "2020-05-13T18:55:37.772853839Z 123",
			},
			want: &LogEvent{
				Message: "123",
			},
		},
		{
			name: "invalid logfmt message",
			args: args{
				message: "2020-05-13T18:55:37.772853839Z sample text with=equal sign",
			},
			want: &LogEvent{
				Message: "sample text with=equal sign",
			},
		},
		{
			name: "null message",
			args: args{
				message: "2020-05-13T18:55:37.772853839Z null",
			},
			want: &LogEvent{
				Message: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := createEvent(tt.args.message, STDOUT); !reflect.DeepEqual(got.Message, tt.want.Message) {
				t.Errorf("createEvent() = %v, want %v", got.Message, tt.want.Message)
			}
		})
	}
}

func TestEventGenerator_ComplexLog(t *testing.T) {
	input := "2020-05-13T18:55:37.772853839Z {\"level\": \"info\", \"message\": \"test\"}"

	g := NewEventGenerator(context.Background(), makeFakeReader(input, STDOUT), Container{Tty: false})
	event := <-g.Events

	require.NotNil(t, event, "Expected event to not be nil")
	assert.Equal(t, LogTypeComplex, event.Type)
	_, isMap := event.Message.(*orderedmap.OrderedMap[string, any])
	assert.True(t, isMap, "Expected Message to be an ordered map")
}

func TestEventGenerator_GroupedSimpleLogs(t *testing.T) {
	// Create messages with same timestamp (close enough to group) where first has level
	baseTime := "2020-05-13T18:55:37.772853839Z"
	messages := []string{
		baseTime + " ERROR: Something went wrong",
		baseTime + " at line 42",
		baseTime + " in function foo",
	}

	reader := &mockLogReader{
		messages: messages,
		types:    []StdType{STDERR, STDERR, STDERR},
	}

	g := NewEventGenerator(context.Background(), reader, Container{Tty: false})
	event := <-g.Events

	require.NotNil(t, event, "Expected event to not be nil")
	assert.Equal(t, LogTypeGroup, event.Type)

	fragments, ok := event.Message.([]LogFragment)
	require.True(t, ok, "Expected Message to be []LogFragment")
	assert.Len(t, fragments, 3)
	assert.Equal(t, "ERROR: Something went wrong", fragments[0].Message)
	assert.Equal(t, "at line 42", fragments[1].Message)
	assert.Equal(t, "in function foo", fragments[2].Message)
}

func TestEventGenerator_SingleSimpleLog(t *testing.T) {
	input := "2020-05-13T18:55:37.772853839Z INFO: Single log message"

	g := NewEventGenerator(context.Background(), makeFakeReader(input, STDOUT), Container{Tty: false})
	event := <-g.Events

	require.NotNil(t, event, "Expected event to not be nil")
	assert.Equal(t, LogTypeSingle, event.Type)
	assert.Equal(t, "INFO: Single log message", event.Message)
}

func TestEventGenerator_MixedLogs(t *testing.T) {
	// Mix of complex and simple logs
	messages := []string{
		"2020-05-13T18:55:37.772853839Z {\"level\": \"info\"}",
		"2020-05-13T18:55:38.772853839Z WARN: warning message",
	}

	reader := &mockLogReader{
		messages: messages,
		types:    []StdType{STDOUT, STDOUT},
	}

	g := NewEventGenerator(context.Background(), reader, Container{Tty: false})

	// First event should be complex
	event1 := <-g.Events
	require.NotNil(t, event1)
	assert.Equal(t, LogTypeComplex, event1.Type)

	// Second event should be single simple
	event2 := <-g.Events
	require.NotNil(t, event2)
	assert.Equal(t, LogTypeSingle, event2.Type)
}

// Tests for orphan skipping: leading levelless lines ARE skipped when container
// started well before (simulating a load-more / scroll-to-top fetch).
func TestEventGenerator_OrphanSkipped_FollowedByLeveledLog(t *testing.T) {
	baseTime := "2020-05-13T18:55:37.772853839Z"
	// Container started hours before the logs — this is a mid-stream fetch.
	containerStart := time.Date(2020, 5, 13, 10, 0, 0, 0, time.UTC)
	messages := []string{
		baseTime + " at line 42",        // orphan (no level, container started long ago)
		baseTime + " in function foo",   // orphan
		baseTime + " ERROR: Next error", // real entry with level
		baseTime + " at line 99",        // continuation of real entry
	}

	reader := &mockLogReader{
		messages: messages,
		types:    []StdType{STDERR, STDERR, STDERR, STDERR},
	}

	g := NewEventGenerator(context.Background(), reader, Container{Tty: false, StartedAt: containerStart})
	event := <-g.Events

	require.NotNil(t, event)
	assert.Equal(t, LogTypeGroup, event.Type)
	fragments, ok := event.Message.([]LogFragment)
	require.True(t, ok)
	assert.Len(t, fragments, 2)
	assert.Equal(t, "ERROR: Next error", fragments[0].Message)
	assert.Equal(t, "at line 99", fragments[1].Message)
}

func TestEventGenerator_OrphanSkipped_FollowedByComplexLog(t *testing.T) {
	baseTime := "2020-05-13T18:55:37.772853839Z"
	containerStart := time.Date(2020, 5, 13, 10, 0, 0, 0, time.UTC)
	messages := []string{
		baseTime + " at line 42",
		baseTime + " in function foo",
		baseTime + " {\"level\": \"info\", \"message\": \"test\"}",
	}

	reader := &mockLogReader{
		messages: messages,
		types:    []StdType{STDERR, STDERR, STDOUT},
	}

	g := NewEventGenerator(context.Background(), reader, Container{Tty: false, StartedAt: containerStart})
	event := <-g.Events

	require.NotNil(t, event)
	assert.Equal(t, LogTypeComplex, event.Type)
}

// When the first log is near the container start, nothing can precede it — no orphan skipping.
func TestEventGenerator_OrphanNotSkipped_NearContainerStart(t *testing.T) {
	baseTime := "2020-05-13T18:55:37.772853839Z"
	// Container started at the same time as the first log.
	containerStart := time.Date(2020, 5, 13, 18, 55, 37, 772853839, time.UTC)
	messages := []string{
		baseTime + " at line 42",
		baseTime + " in function foo",
		baseTime + " ERROR: Next error",
		baseTime + " at line 99",
	}

	reader := &mockLogReader{
		messages: messages,
		types:    []StdType{STDERR, STDERR, STDERR, STDERR},
	}

	g := NewEventGenerator(context.Background(), reader, Container{Tty: false, StartedAt: containerStart})

	// Leading lines emitted as singles since we're at the container start
	event1 := <-g.Events
	require.NotNil(t, event1)
	assert.Equal(t, LogTypeSingle, event1.Type)
	assert.Equal(t, "at line 42", event1.Message)

	event2 := <-g.Events
	require.NotNil(t, event2)
	assert.Equal(t, LogTypeSingle, event2.Type)
	assert.Equal(t, "in function foo", event2.Message)

	// Then the real grouped entry
	event3 := <-g.Events
	require.NotNil(t, event3)
	assert.Equal(t, LogTypeGroup, event3.Type)
	fragments, ok := event3.Message.([]LogFragment)
	require.True(t, ok)
	assert.Len(t, fragments, 2)
	assert.Equal(t, "ERROR: Next error", fragments[0].Message)
	assert.Equal(t, "at line 99", fragments[1].Message)
}

// Tests for orphan NOT skipped: leading levelless lines are emitted when no real logs follow.
func TestEventGenerator_OrphanNotSkipped_AllLevellessLines(t *testing.T) {
	baseTime := "2020-05-13T18:55:37.772853839Z"
	messages := []string{
		baseTime + " at line 42",
		baseTime + " in function foo",
		baseTime + " in function bar",
	}

	reader := &mockLogReader{
		messages: messages,
		types:    []StdType{STDERR, STDERR, STDERR},
	}

	g := NewEventGenerator(context.Background(), reader, Container{Tty: false})

	var events []*LogEvent
	for event := range g.Events {
		events = append(events, event)
	}

	assert.Len(t, events, 3)
	for _, event := range events {
		assert.Equal(t, LogTypeSingle, event.Type)
	}
}

func TestEventGenerator_OrphanNotSkipped_TimestampGapBreaksOrphanDetection(t *testing.T) {
	// Lines far apart in time — first is buffered as orphan candidate but the
	// gap breaks the chain. Both must be emitted: a single isolated levelless
	// line is not an orphan continuation, and dropping it loses real user
	// content (e.g. postgres "checkpoint starting: time" is the first event
	// of every 5-min historical window).
	containerStart := time.Date(2020, 5, 13, 10, 0, 0, 0, time.UTC)
	messages := []string{
		"2020-05-13T18:55:37.000Z some log without level",
		"2020-05-13T18:55:38.000Z another log without level",
	}

	reader := &mockLogReader{
		messages: messages,
		types:    []StdType{STDOUT, STDOUT},
	}

	g := NewEventGenerator(context.Background(), reader, Container{Tty: false, StartedAt: containerStart})

	event1 := <-g.Events
	require.NotNil(t, event1)
	assert.Equal(t, LogTypeSingle, event1.Type)
	assert.Equal(t, "some log without level", event1.Message)

	event2 := <-g.Events
	require.NotNil(t, event2)
	assert.Equal(t, LogTypeSingle, event2.Type)
	assert.Equal(t, "another log without level", event2.Message)
}

func TestEventGenerator_OrphanNotSkipped_NoTimestamp(t *testing.T) {
	// Lines without timestamps (e.g., tty/raw input) are never treated as orphans.
	input := "some raw output"

	g := NewEventGenerator(context.Background(), makeFakeReader(input, STDOUT), Container{Tty: true})
	event := <-g.Events

	require.NotNil(t, event)
	assert.Equal(t, input, event.Message)
	assert.Equal(t, LogTypeSingle, event.Type)
}

func TestEventGenerator_NoGroupingWhenTimestampGap(t *testing.T) {
	// Messages with different timestamps (too far apart to group)
	messages := []string{
		"2020-05-13T18:55:37.000Z ERROR: First error",
		"2020-05-13T18:55:38.000Z continuation line",
	}

	reader := &mockLogReader{
		messages: messages,
		types:    []StdType{STDERR, STDERR},
	}

	g := NewEventGenerator(context.Background(), reader, Container{Tty: false})

	// Should get two separate events (not grouped due to timestamp gap)
	event1 := <-g.Events
	require.NotNil(t, event1)
	assert.Equal(t, LogTypeSingle, event1.Type)

	event2 := <-g.Events
	require.NotNil(t, event2)
	assert.Equal(t, LogTypeSingle, event2.Type)
}

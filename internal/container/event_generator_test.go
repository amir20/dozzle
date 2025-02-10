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
}

func TestEventGenerator_Events_non_tty(t *testing.T) {
	input := "example input"

	g := NewEventGenerator(context.Background(), makeFakeReader(input, STDOUT), Container{Tty: false})
	event := <-g.Events

	require.NotNil(t, event, "Expected event to not be nil, but got nil")
	assert.Equal(t, input, event.Message)
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

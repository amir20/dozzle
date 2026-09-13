package container

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTimestampPrefixLen(t *testing.T) {
	docker := time.Date(2026, 9, 13, 22, 28, 56, 412_000_000, time.UTC).UnixMilli()

	tests := []struct {
		name    string
		message string
		rest    string // what the line should read as once the prefix is hidden; "" means no prefix
	}{
		{"zerolog colored", "\x1b[90m2026-09-13T22:28:56Z\x1b[0m \x1b[32mINF\x1b[0m Starting server", "\x1b[32mINF\x1b[0m Starting server"},
		{"plain RFC3339", "2026-09-13T22:28:56Z INF Starting server", "INF Starting server"},
		{"RFC3339Nano", "2026-09-13T22:28:56.412345678Z hello", "hello"},
		{"space and comma fraction", "2026-09-13 22:28:56,412 INFO hello", "INFO hello"},
		{"bracketed", "[2026-09-13 22:28:56.412] [info] hello", "[info] hello"},
		{"bracketed no space", "[2026-09-13T22:28:56Z]INFO hello", "INFO hello"},
		{"slashes", "2026/09/13 22:28:56 [error] nginx", "[error] nginx"},
		{"offset with colon", "2026-09-13T15:28:56-07:00 hello", "hello"},
		{"offset without colon", "2026-09-14T03:58:56.1+0530 hello", "hello"},
		{"local time no zone", "2026-09-13 18:28:56 hello", "hello"},
		{"local time half hour zone", "2026-09-14 03:58:57 hello", "hello"},
		{"tab after", "2026-09-13T22:28:56Z\thello", "hello"},
		{"reset inside bracket", "\x1b[2m[2026-09-13T22:28:56Z\x1b[0m] hello", "hello"},

		{"mismatched zoned time", "2026-09-13T22:20:00Z INF old event", ""},
		{"zoned offset disagrees", "2026-09-13T22:28:56+01:00 hello", ""},
		{"local time not on a quarter hour", "2026-09-13 18:21:56 hello", ""},
		{"local time too far off", "2026-09-14 14:28:56 hello", ""},
		{"starts with digits", "200 OK GET /index.html", ""},
		{"date only", "2026-09-13 hello", ""},
		{"color runs past timestamp", "\x1b[31m2026-09-13T22:28:56Z ERROR boom\x1b[0m", ""},
		{"color set again after reset is kept", "\x1b[90m2026-09-13T22:28:56Z\x1b[0m\x1b[31m ERR", ""},
		{"nothing after timestamp", "2026-09-13T22:28:56Z   ", ""},
		{"glued to text", "2026-09-13T22:28:56Zhello", ""},
		{"unclosed bracket", "[2026-09-13T22:28:56Z hello", ""},
		{"invalid month", "2026-13-13T22:28:56Z hello", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := timestampPrefixLen(tt.message, docker)
			if tt.rest == "" {
				assert.Zero(t, n)
				return
			}
			assert.Equal(t, tt.rest, tt.message[n:])
		})
	}

	t.Run("no docker timestamp", func(t *testing.T) {
		assert.Zero(t, timestampPrefixLen("2026-09-13T22:28:56Z hello", 0))
	})
}

func TestCreateEventTimestampPrefix(t *testing.T) {
	event := createEvent("2026-09-13T22:28:56.412Z \x1b[90m2026-09-13T22:28:56Z\x1b[0m INF ready", STDOUT)
	assert.Equal(t, "\x1b[90m2026-09-13T22:28:56Z\x1b[0m INF ready", event.Message, "message must stay intact")
	assert.Equal(t, "INF ready", event.Message.(string)[event.TimestampPrefix:])

	json := createEvent(`2026-09-13T22:28:56.412Z {"time":"2026-09-13T22:28:56Z","msg":"ready"}`, STDOUT)
	assert.Equal(t, LogTypeComplex, json.Type)
	assert.Zero(t, json.TimestampPrefix)
}

func TestEventGenerator_GroupKeepsTimestampPrefix(t *testing.T) {
	reader := &mockLogReader{
		messages: []string{
			"2026-09-13T22:28:56.412Z 2026-09-13T22:28:56Z ERR request failed",
			"2026-09-13T22:28:56.413Z   at handler.go:42",
			"2026-09-13T22:28:56.414Z 2026-09-13T22:28:56Z   at main.go:7",
		},
		types: []StdType{STDERR, STDERR, STDERR},
	}

	g := NewEventGenerator(context.Background(), reader, Container{Tty: false})
	event := <-g.Events

	require.NotNil(t, event)
	require.Equal(t, LogTypeGroup, event.Type)
	fragments := event.Message.([]LogFragment)
	require.Len(t, fragments, 3)
	assert.Equal(t, "ERR request failed", fragments[0].Message[fragments[0].TimestampPrefix:])
	assert.Zero(t, fragments[1].TimestampPrefix)
	assert.Equal(t, "at main.go:7", fragments[2].Message[fragments[2].TimestampPrefix:])
}

func BenchmarkTimestampPrefixLen(b *testing.B) {
	docker := time.Date(2026, 9, 13, 22, 28, 56, 0, time.UTC).UnixMilli()
	lines := []string{
		"\x1b[90m2026-09-13T22:28:56Z\x1b[0m \x1b[32mINF\x1b[0m Starting server",
		"GET /healthz 200 1ms",
	}
	b.ReportAllocs()
	for b.Loop() {
		for _, l := range lines {
			timestampPrefixLen(l, docker)
		}
	}
}

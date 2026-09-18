package logparse

import (
	"encoding/json"
	"testing"

	"github.com/amir20/dozzle/internal/container"
	"github.com/stretchr/testify/assert"
)

func Test_parsePrefixedJSON(t *testing.T) {
	const dockerTs = "2026-09-18T22:46:26.123456789Z "
	tests := []struct {
		name    string
		message string
		want    string // JSON of the parsed map, "" when it must not parse
	}{
		{
			name:    "zap console",
			message: "INFO\tfinished call\t{\"grpc.code\": \"OK\", \"grpc.time_ms\": \"0.158\"}",
			want:    `{"level":"info","msg":"finished call","grpc.code":"OK","grpc.time_ms":"0.158"}`,
		},
		{
			name:    "zap console with logger and caller",
			message: "WARN\tgrpc\tserver/server.go:42\tslow call\t{\"ms\": 900}",
			want:    `{"level":"warn","msg":"grpc server/server.go:42 slow call","ms":900}`,
		},
		{
			name:    "zap console with matching app timestamp",
			message: "2026-09-18T22:46:26.123Z\tERROR\tboom\t{\"err\": \"x\"}",
			want:    `{"level":"error","msg":"boom","err":"x"}`,
		},
		{
			name:    "no level in prefix",
			message: "request done {\"status\": 200}",
			want:    `{"msg":"request done","status":200}`,
		},
		{
			name:    "space separated prefix still yields a level",
			message: "INFO request done {\"status\": 200}",
			want:    `{"level":"info","msg":"INFO request done","status":200}`,
		},
		{
			name:    "object keys win over the prefix",
			message: "INFO\tprefix text\t{\"level\": \"debug\", \"msg\": \"inner\"}",
			want:    `{"level":"debug","msg":"inner"}`,
		},
		{name: "plain json is not prefixed", message: "{\"a\": 1}"},
		{name: "json not at end of line", message: "INFO sent {\"a\": 1} upstream"},
		{name: "invalid tail", message: "INFO done {\"a\"}"},
		{name: "brace glued to text", message: "payload={\"a\": 1}"},
		{name: "braces earlier in the prefix", message: "INFO {user} logged in {\"a\": 1}"},
		{name: "array tail", message: "INFO ids [1, 2]"},
		{name: "prefix too long", message: string(make([]byte, 300)) + " {\"a\": 1}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := createEvent(dockerTs+tt.message, container.STDOUT)
			data, ok := parsePrefixedJSON(tt.message, event.Timestamp)
			if tt.want == "" {
				assert.False(t, ok)
				return
			}
			assert.True(t, ok)
			got, err := json.Marshal(data)
			assert.NoError(t, err)
			assert.JSONEq(t, tt.want, string(got))
			// JSONEq ignores order; the prefix fields must lead.
			assert.Equal(t, tt.want, string(got))
		})
	}
}

func Test_createEvent_prefixedJSON(t *testing.T) {
	line := "INFO\tfinished call\t{\"server\": \"grpc\", \"grpc.code\": \"OK\"}"
	event := createEvent("2026-09-18T22:46:26.123456789Z "+line, container.STDOUT)

	assert.Equal(t, container.LogTypeComplex, event.Type)
	assert.JSONEq(t, `{"level":"info","msg":"finished call","server":"grpc","grpc.code":"OK"}`, event.RawMessage)
	assert.Equal(t, "info", guessLogLevel(event))
}

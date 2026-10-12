package agent

import (
	"testing"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/container/agent/pb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComplexMessageRoundTripKeepsOrder(t *testing.T) {
	fields, err := container.ParseFields([]byte(`{"z":1,"nested":{"b":"x","a":[true,null]},"a":"s"}`))
	require.NoError(t, err)

	event := logEventToPb(&container.LogEvent{Message: fields, Type: container.LogTypeComplex})
	var complexMessage pb.ComplexMessage
	require.NoError(t, event.Message.UnmarshalTo(&complexMessage))

	assert.Equal(t, fields, jsonBytesToFields(complexMessage.Data))
}

func TestComplexMessageFromLogfmt(t *testing.T) {
	fields := container.Fields[string]{{Key: "level", Value: "info"}, {Key: "msg", Value: "hi"}}

	event := logEventToPb(&container.LogEvent{Message: fields, Type: container.LogTypeComplex})
	var complexMessage pb.ComplexMessage
	require.NoError(t, event.Message.UnmarshalTo(&complexMessage))

	assert.Equal(t, container.Fields[any]{{Key: "level", Value: "info"}, {Key: "msg", Value: "hi"}}, jsonBytesToFields(complexMessage.Data))
}

// Agents built before container.Fields encoded with json.Encoder, which ends
// with a newline; a newer server must still read them.
func TestJSONBytesToFieldsFromOlderAgent(t *testing.T) {
	assert.Equal(t,
		container.Fields[any]{{Key: "b", Value: 1.0}, {Key: "a", Value: "x"}},
		jsonBytesToFields([]byte("{\"b\":1,\"a\":\"x\"}\n")),
	)
}

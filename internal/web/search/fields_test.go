package search

import (
	"testing"

	"github.com/amir20/dozzle/internal/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Nested objects decode as container.Fields, so search and escaping must reach
// into them, and neither may reorder keys.
func TestNestedFieldsAreSearchedAndEscaped(t *testing.T) {
	fields, err := container.ParseFields([]byte(`{"z":"top","user":{"name":"<b>needle</b>","id":7},"a":1}`))
	require.NoError(t, err)
	event := &container.LogEvent{Type: container.LogTypeComplex, Message: fields}

	re, err := ParseRegex("needle")
	require.NoError(t, err)
	require.True(t, Search(re, event))

	EscapeHTMLValues(event)

	out, err := event.Message.(container.Fields[any]).MarshalJSON()
	require.NoError(t, err)
	assert.Equal(t, `{"z":"top","user":{"name":"&lt;b&gt;<mark>needle</mark>&lt;/b&gt;","id":7},"a":1}`, string(out))
}

func TestFieldsStringIsSearchedAndEscaped(t *testing.T) {
	event := &container.LogEvent{
		Type:    container.LogTypeComplex,
		Message: container.Fields[string]{{Key: "msg", Value: "<i>needle</i>"}, {Key: "level", Value: "info"}},
	}
	re, err := ParseRegex("needle")
	require.NoError(t, err)
	require.True(t, Search(re, event))
	EscapeHTMLValues(event)

	out, err := event.Message.(container.Fields[string]).MarshalJSON()
	require.NoError(t, err)
	assert.Equal(t, `{"msg":"&lt;i&gt;<mark>needle</mark>&lt;/i&gt;","level":"info"}`, string(out))
}

func TestNumberFieldIsMarked(t *testing.T) {
	fields, err := container.ParseFields([]byte(`{"status":404,"msg":"ok"}`))
	require.NoError(t, err)
	event := &container.LogEvent{Type: container.LogTypeComplex, Message: fields}
	re, err := ParseRegex("404")
	require.NoError(t, err)
	require.True(t, Search(re, event))
	status, _ := event.Message.(container.Fields[any]).Get("status")
	assert.Equal(t, MarkerStart+"404"+MarkerEnd, status)
}

package docker

import (
	"bufio"
	"io"
	"strings"
	"testing"

	"github.com/magiconair/properties/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEventIterator(t *testing.T) {
	input := "example input"
	reader := bufio.NewReader(strings.NewReader(input))

	generator := NewEventIterator(reader)
	require.NotNil(t, generator, "Expected generator to not be nil, but got nil")
}

func TestEventGenerator_Next(t *testing.T) {
	input := "example input"
	reader := bufio.NewReader(strings.NewReader(input))

	generator := NewEventIterator(reader)

	event, err := generator.Next()
	require.NoError(t, err, "Expected no error, but got: %v", err)
	require.NotNil(t, event, "Expected event to not be nil, but got nil")
}

func TestEventGenerator_LastError(t *testing.T) {
	input := "example input"
	reader := bufio.NewReader(strings.NewReader(input))

	generator := NewEventIterator(reader)

	require.Nil(t, generator.LastError(), "Expected LastError to return nil, but got: %v", generator.LastError())

	generator.Next()

	// expert error to be EOF
	assert.Equal(t, generator.LastError(), io.EOF, "Expected LastError to return EOF, but got: %v", generator.LastError().Error())
}

func TestEventGenerator_Peek(t *testing.T) {
	input := "example input"
	reader := bufio.NewReader(strings.NewReader(input))

	generator := NewEventIterator(reader)

	event := generator.Peek()

	require.NotNil(t, event, "Expected event to not be nil, but got nil")
	assert.Equal(t, event.Message, input, "Expected event message to be %s, but got: %s", input, event.Message.(string))
}

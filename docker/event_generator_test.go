package docker

import (
	"bufio"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEventGenerator_Events(t *testing.T) {
	input := "example input"
	reader := bufio.NewReader(strings.NewReader(input))

	events, _ := NewEventGenerator(reader, true)
	event := <-events

	require.NotNil(t, event, "Expected event to not be nil, but got nil")
}

package k8s

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRead_lastLineWithoutNewline(t *testing.T) {
	input := "2024-01-01T00:00:00.000000000Z first\n2024-01-01T00:00:01.000000000Z last line without newline"
	reader := NewLogReader(io.NopCloser(strings.NewReader(input)))

	message, _, err := reader.Read()
	require.NoError(t, err)
	assert.Equal(t, "2024-01-01T00:00:00.000000000Z first\n", message)

	message, _, err = reader.Read()
	assert.Equal(t, io.EOF, err)
	assert.Equal(t, "2024-01-01T00:00:01.000000000Z last line without newline", message)
}

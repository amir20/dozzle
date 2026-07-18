package docker

import (
	"bytes"
	"encoding/binary"
	"io"
	"strings"
	"testing"

	"github.com/amir20/dozzle/internal/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeFrame(buf *bytes.Buffer, streamType byte, payload string) {
	header := make([]byte, 8)
	header[0] = streamType
	binary.BigEndian.PutUint32(header[4:], uint32(len(payload)))
	buf.Write(header)
	buf.WriteString(payload)
}

func TestRead_multiplexed(t *testing.T) {
	buf := &bytes.Buffer{}
	writeFrame(buf, 1, "2024-01-01T00:00:00.000000000Z out\n")
	writeFrame(buf, 2, "2024-01-01T00:00:01.000000000Z err\n")

	reader := NewLogReader(buf, false)

	message, std, err := reader.Read()
	require.NoError(t, err)
	assert.Equal(t, "2024-01-01T00:00:00.000000000Z out\n", message)
	assert.Equal(t, container.STDOUT, std)

	message, std, err = reader.Read()
	require.NoError(t, err)
	assert.Equal(t, "2024-01-01T00:00:01.000000000Z err\n", message)
	assert.Equal(t, container.STDERR, std)

	_, _, err = reader.Read()
	assert.Equal(t, io.EOF, err)
}

func TestRead_lastLineWithoutNewline(t *testing.T) {
	buf := &bytes.Buffer{}
	writeFrame(buf, 1, "2024-01-01T00:00:00.000000000Z first\n")
	writeFrame(buf, 1, "2024-01-01T00:00:01.000000000Z last line without newline")

	reader := NewLogReader(buf, false)

	message, _, err := reader.Read()
	require.NoError(t, err)
	assert.Equal(t, "2024-01-01T00:00:00.000000000Z first\n", message)

	message, std, err := reader.Read()
	assert.Equal(t, io.EOF, err)
	assert.Equal(t, container.STDOUT, std)
	assert.Equal(t, "2024-01-01T00:00:01.000000000Z last line without newline", message)
}

func TestRead_lastLineWithoutNewlineTTY(t *testing.T) {
	input := "2024-01-01T00:00:00.000000000Z first\n2024-01-01T00:00:01.000000000Z last line without newline"
	reader := NewLogReader(strings.NewReader(input), true)

	message, _, err := reader.Read()
	require.NoError(t, err)
	assert.Equal(t, "2024-01-01T00:00:00.000000000Z first\n", message)

	message, _, err = reader.Read()
	assert.Equal(t, io.EOF, err)
	assert.Equal(t, "2024-01-01T00:00:01.000000000Z last line without newline", message)
}

func TestRead_continuedFrames(t *testing.T) {
	buf := &bytes.Buffer{}
	writeFrame(buf, 1, "2024-01-01T00:00:00.000000000Z part one ")
	writeFrame(buf, 1, "2024-01-01T00:00:00.000000000Z and part two\n")

	reader := NewLogReader(buf, false)

	message, _, err := reader.Read()
	require.NoError(t, err)
	assert.Equal(t, "2024-01-01T00:00:00.000000000Z part one and part two\n", message)
}

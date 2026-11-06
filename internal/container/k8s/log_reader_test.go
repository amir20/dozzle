package k8s

import (
	"io"
	"runtime"
	"strings"
	"testing"

	"github.com/amir20/dozzle/internal/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// hugeLine serves size bytes with no newline, then tail, and records how much heap
// the reader is holding once it has been handed all of them.
type hugeLine struct {
	remaining int
	tail      io.Reader
	base      uint64
	held      uint64
}

func (h *hugeLine) Read(p []byte) (int, error) {
	if h.remaining == 0 {
		if h.held == 0 {
			h.held = heapInUse()
		}
		return h.tail.Read(p)
	}
	n := min(len(p), h.remaining)
	for i := range n {
		p[i] = 'a'
	}
	h.remaining -= n
	return n, nil
}

func heapInUse() uint64 {
	runtime.GC()
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	return stats.HeapAlloc
}

func TestRead_hugeLineStopsGrowingAtTheCap(t *testing.T) {
	source := &hugeLine{remaining: 16 * container.MaxLogLineBytes, tail: strings.NewReader("end\nnext\n"), base: heapInUse()}
	reader := NewLogReader(io.NopCloser(source))

	message, _, err := reader.Read()
	require.NoError(t, err)
	assert.Less(t, source.held, source.base+4*container.MaxLogLineBytes, "heap held while reading one long line")
	assert.LessOrEqual(t, len(message), container.MaxLogLineBytes+len(container.LogTruncationSuffix)+1)
	assert.True(t, strings.HasSuffix(message, container.LogTruncationSuffix+"\n"))

	message, _, err = reader.Read()
	require.NoError(t, err)
	assert.Equal(t, "next\n", message)
}

func TestRead_hugeLastLineWithoutNewline(t *testing.T) {
	input := strings.Repeat("a", 2*container.MaxLogLineBytes)
	reader := NewLogReader(io.NopCloser(strings.NewReader(input)))

	message, _, err := reader.Read()
	assert.Equal(t, io.EOF, err)
	assert.True(t, strings.HasSuffix(message, container.LogTruncationSuffix))
}

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

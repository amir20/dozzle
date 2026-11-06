package k8s

import (
	"bufio"
	"bytes"
	"errors"
	"io"

	"github.com/amir20/dozzle/internal/container"
)

type LogReader struct {
	reader *bufio.Reader
}

func NewLogReader(reader io.ReadCloser) *LogReader {
	return &LogReader{
		reader: bufio.NewReader(reader),
	}
}

func (r *LogReader) Read() (string, container.StdType, error) {
	var line []byte
	for {
		chunk, err := r.reader.ReadSlice('\n')
		// Past the cap, stop growing but keep draining to the newline, so the next
		// Read still starts at the beginning of a line. Growing until past the cap, not
		// up to it, is what makes TruncateLogLine mark a line that lost its end.
		if len(line) <= container.MaxLogLineBytes {
			line = append(line, chunk...)
		} else if bytes.HasSuffix(chunk, []byte("\n")) {
			line = append(line, '\n')
		}
		if errors.Is(err, bufio.ErrBufferFull) {
			continue
		}
		// A final line without a trailing newline arrives together with EOF.
		// Return the partial line with the error instead of dropping it; the
		// event generator emits the message before handling the error.
		return container.TruncateLogLine(string(line)), container.STDOUT, err
	}
}

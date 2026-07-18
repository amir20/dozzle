package k8s

import (
	"bufio"
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
	// A final line without a trailing newline arrives together with EOF.
	// Return the partial line with the error instead of dropping it; the
	// event generator emits the message before handling the error.
	line, err := r.reader.ReadString('\n')
	return line, container.STDOUT, err
}

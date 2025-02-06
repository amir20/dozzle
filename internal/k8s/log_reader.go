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
	line, err := r.reader.ReadString('\n')
	if err != nil {
		return "", 0, err
	}

	return line, container.STDOUT, nil
}

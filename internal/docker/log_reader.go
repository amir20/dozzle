package docker

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"strings"
	"sync"

	"github.com/amir20/dozzle/internal/container"
)

var ErrBadHeader = errors.New("bad header")

type StdType int

const (
	stdout StdType = iota
	stderr
)

type LogReader struct {
	reader *bufio.Reader
	tty    bool
	pool   *sync.Pool
}

func NewLogReader(r io.Reader, tty bool) *LogReader {
	return &LogReader{
		reader: bufio.NewReader(r),
		tty:    tty,
		pool: &sync.Pool{
			New: func() any {
				return bytes.NewBuffer(make([]byte, 0, 4096))
			},
		},
	}
}

func (d *LogReader) Read() (string, container.StdType, error) {
	message, stdType, err := d.readEvent()

	var std container.StdType
	switch stdType {
	case stdout:
		std = container.STDOUT
	case stderr:
		std = container.STDERR
	}

	// A final line without a trailing newline (e.g. a container's last write
	// before exiting) arrives together with EOF. Return the partial message
	// with the error instead of dropping it; the event generator emits the
	// message before handling the error.
	if err != nil {
		return message, std, err
	}

	for !strings.HasSuffix(message, "\n") {
		tail, _, err := d.readEvent()
		if err != nil {
			return message, std, err
		}

		_, after, _ := strings.Cut(tail, " ")
		message += after
	}

	return message, std, nil
}

func (d *LogReader) readEvent() (string, StdType, error) {
	header := []byte{0, 0, 0, 0, 0, 0, 0, 0}
	buffer := d.pool.Get().(*bytes.Buffer)
	buffer.Reset()
	defer d.pool.Put(buffer)

	var streamType StdType = stdout

	if d.tty {
		message, err := d.reader.ReadString('\n')
		if err != nil {
			return message, streamType, err
		}
		return message, streamType, nil
	} else {
		n, err := io.ReadFull(d.reader, header)
		if err != nil {
			return "", streamType, err
		}
		if n != 8 {
			message, _ := d.reader.ReadString('\n')
			return message, streamType, ErrBadHeader
		}

		switch header[0] {
		case 1:
			streamType = stdout
		case 2:
			streamType = stderr
		}

		count := binary.BigEndian.Uint32(header[4:])
		if count == 0 {
			return "", streamType, nil
		}

		_, err = io.CopyN(buffer, d.reader, int64(count))
		if err != nil {
			return "", streamType, err
		}
		return buffer.String(), streamType, nil
	}
}

package docker

import (
	"bytes"
	"encoding/binary"
	"io"
)

type logReader struct {
	readerCloser io.ReadCloser
	tty          bool
	lastHeader   []byte
	buffer       bytes.Buffer
}

func newLogReader(reader io.ReadCloser, tty bool) io.ReadCloser {
	return &logReader{
		reader,
		tty,
		make([]byte, 8),
		bytes.Buffer{},
	}
}

func (r *logReader) Read(p []byte) (n int, err error) {
	if r.tty {
		return r.readerCloser.Read(p)
	} else {
		if r.buffer.Len() > 0 {
			return r.buffer.Read(p)
		} else {
			r.buffer.Reset()
			_, err := r.readerCloser.Read(r.lastHeader)
			if err != nil {
				return -1, err
			}
			count := binary.BigEndian.Uint32(r.lastHeader[4:])
			_, err = io.CopyN(&r.buffer, r.readerCloser, int64(count))
			if err != nil {
				return -1, err
			}
			return r.buffer.Read(p)
		}
	}
}

func (r *logReader) Close() error {
	return r.readerCloser.Close()
}

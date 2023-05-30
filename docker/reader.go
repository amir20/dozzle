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
	label        bool
}

func newLogReader(reader io.ReadCloser, tty bool, labelStd bool) io.ReadCloser {
	return &logReader{
		reader,
		tty,
		make([]byte, 8),
		bytes.Buffer{},
		labelStd,
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
				return 0, err
			}
			if r.label {
				std := r.lastHeader[0] // https://github.com/rancher/docker/blob/master/pkg/stdcopy/stdcopy.go#L94

				if std == 1 {
					r.buffer.WriteString("OUT")
				}
				if std == 2 {
					r.buffer.WriteString("ERR")
				}
			}
			count := binary.BigEndian.Uint32(r.lastHeader[4:])
			_, err = io.CopyN(&r.buffer, r.readerCloser, int64(count))
			if err != nil {
				return 0, err
			}
			return r.buffer.Read(p)
		}
	}
}

func (r *logReader) Close() error {
	return r.readerCloser.Close()
}

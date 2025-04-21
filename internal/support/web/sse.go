package support_web

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"net/http"
)

type SSEWriter struct {
	w io.Writer
	f http.Flusher
}

type HasId interface {
	MessageId() int64
}

func NewSSEWriter(ctx context.Context, w http.ResponseWriter, r *http.Request) (*SSEWriter, error) {
	if _, ok := w.(http.Flusher); !ok {
		return nil, http.ErrNotSupported
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-transform")
	w.Header().Add("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	var writer io.Writer = w
	if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		w.Header().Set("Content-Encoding", "gzip")
		writer = gzip.NewWriter(w)
	}

	sse := &SSEWriter{
		w: writer,
		f: w.(http.Flusher),
	}

	return sse, nil
}

func (s *SSEWriter) Write(data []byte) (int, error) {
	written, err := s.w.Write(data)
	if err != nil {
		return written, err
	}

	_, err = s.w.Write([]byte("\n\n"))
	if err != nil {
		return written, err
	}

	if f, ok := s.w.(*gzip.Writer); ok {
		err := f.Flush()
		if err != nil {
			return written, err
		}
	}

	s.f.Flush()

	return written, nil
}

func (s *SSEWriter) Ping() error {
	_, err := s.Write([]byte(":ping "))
	return err
}

func (s *SSEWriter) Close() {
	if closer, ok := s.w.(io.Closer); ok && s.w != nil {
		closer.Close()
	}
}

func (s *SSEWriter) Message(data any) error {
	encoded, err := json.Marshal(data)

	if err != nil {
		return err
	}

	buffer := bytes.Buffer{}

	buffer.WriteString("data: ")
	buffer.Write(encoded)
	buffer.WriteString("\n")

	if f, ok := data.(HasId); ok {
		if f.MessageId() > 0 {
			buffer.WriteString(fmt.Sprintf("id: %d\n", f.MessageId()))
		}
	}

	_, err = buffer.WriteTo(s)
	return err
}

func (s *SSEWriter) Event(event string, data any) error {
	encoded, err := json.Marshal(data)

	if err != nil {
		return err
	}

	buffer := bytes.Buffer{}
	buffer.WriteString("event: " + event + "\n")
	buffer.WriteString("data: ")
	buffer.Write(encoded)
	buffer.WriteString("\n")

	_, err = buffer.WriteTo(s)
	return err
}

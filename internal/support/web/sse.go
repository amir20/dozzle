package support_web

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"net/http"
)

type SSEWriter struct {
	f http.Flusher
	w http.ResponseWriter
}

type HasId interface {
	MessageId() int64
}

func NewSSEWriter(ctx context.Context, w http.ResponseWriter) (*SSEWriter, error) {
	f, ok := w.(http.Flusher)

	if !ok {
		return nil, http.ErrNotSupported
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-transform")
	w.Header().Add("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	sse := &SSEWriter{
		f: f,
		w: w,
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

	s.f.Flush()

	return written, nil
}

func (s *SSEWriter) Ping() error {
	_, err := s.Write([]byte(":ping "))
	return err
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

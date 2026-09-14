package sse

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"net/http"

	"github.com/rs/zerolog/log"
)

// writeTimeout bounds a single write to an SSE client. A client that goes away without
// closing the socket (a sleeping laptop, a NAT that forgot the flow) stays writable until
// its send buffer fills, and the write then blocks until the kernel gives up retransmitting,
// which takes minutes. For that whole window the handler stops reading its event channel
// and the container store drops every event aimed at it. A deadline turns the wedge into an
// error the handler returns on, which cancels the request context and unsubscribes it.
const writeTimeout = 10 * time.Second

type Writer struct {
	w io.Writer
	f http.Flusher
	// nil when the ResponseWriter cannot set deadlines, in which case writes block as before
	rc *http.ResponseController
}

type HasId interface {
	MessageId() int64
}

func NewWriter(ctx context.Context, w http.ResponseWriter, r *http.Request) (*Writer, error) {
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

	sse := &Writer{
		w: writer,
		f: w.(http.Flusher),
	}

	// probe once rather than per write: a wrapped ResponseWriter that does not implement
	// SetWriteDeadline reports it the same way every time
	rc := http.NewResponseController(w)
	if err := rc.SetWriteDeadline(time.Time{}); err == nil {
		sse.rc = rc
	} else {
		log.Debug().Err(err).Msg("sse stream cannot set write deadlines, a dead client will block writes")
	}

	return sse, nil
}

func (s *Writer) Write(data []byte) (int, error) {
	if s.rc != nil {
		// covers the payload, the gzip flush and the http flush below
		s.rc.SetWriteDeadline(time.Now().Add(writeTimeout))
	}

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

// Retry sets how long the browser waits before reconnecting a dropped stream. Browsers
// pick their own default otherwise (Chrome ~3s, Firefox ~5s), and it is reset on every
// reconnect, so it is sent again at the top of each stream.
func (s *Writer) Retry(d time.Duration) error {
	_, err := s.Write([]byte(fmt.Sprintf("retry: %d", d.Milliseconds())))
	return err
}

func (s *Writer) Ping() error {
	_, err := s.Write([]byte(":ping "))
	return err
}

func (s *Writer) Close() {
	if closer, ok := s.w.(io.Closer); ok && s.w != nil {
		closer.Close()
	}
}

func (s *Writer) Message(data any) error {
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

func (s *Writer) Event(event string, data any) error {
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

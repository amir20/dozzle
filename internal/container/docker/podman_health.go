package docker

import (
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"strings"

	"github.com/moby/moby/api/types"
)

// healthEventCompat rewrites Podman's health event into Docker's encoding.
//
// Docker bakes the result into the action itself:
//
//	{"Action": "health_status: healthy", ...}
//
// Podman's Docker-compatible endpoint sends the bare action and carries the
// result in a field of its own, beside the embedded Docker message
// (pkg/domain/entities/types/events.go):
//
//	{"Action": "health_status", ..., "HealthStatus": "healthy"}
//
// It is not in Actor.Attributes, and events.Message has no field for it, so the
// client drops it while decoding and nothing downstream ever sees a health
// transition. Rewriting the bytes before the client reads them keeps that one
// difference here, so the store, the SSE stream, the notification rules and the
// agent all go on matching a single encoding.
func healthEventCompat(resp *http.Response) {
	if resp.Request == nil || resp.StatusCode != http.StatusOK {
		return
	}
	if !strings.HasSuffix(resp.Request.URL.Path, "/events") {
		return
	}

	// A JSON sequence separates records with a control byte that the decoder
	// below would choke on. Podman never answers with one, so leave that stream
	// alone rather than risk garbling Docker's.
	mediaType, _, _ := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if types.MediaType(mediaType) == types.MediaTypeJSONSequence {
		return
	}

	resp.Body = rewriteEvents(resp.Body)
}

// rewriteEvents streams body through dockerHealthShape. Records are handed on one
// at a time, as they arrive: an event stream is live, and buffering it would hold
// every event back behind the next one.
func rewriteEvents(body io.ReadCloser) io.ReadCloser {
	reader, writer := io.Pipe()

	go func() {
		decoder := json.NewDecoder(body)
		encoder := json.NewEncoder(writer)
		for {
			var record json.RawMessage
			if err := decoder.Decode(&record); err != nil {
				writer.CloseWithError(err)
				return
			}
			if err := encoder.Encode(dockerHealthShape(record)); err != nil {
				writer.CloseWithError(err)
				return
			}
		}
	}()

	return &rewrittenBody{reader: reader, source: body}
}

type rewrittenBody struct {
	reader *io.PipeReader
	source io.ReadCloser
}

func (b *rewrittenBody) Read(p []byte) (int, error) {
	return b.reader.Read(p)
}

// Close ends both halves: the pipe, so a blocked write in the goroutine returns,
// and the response body it was reading from.
func (b *rewrittenBody) Close() error {
	b.reader.Close()
	return b.source.Close()
}

// dockerHealthShape rewrites a Podman health event into Docker's encoding and
// leaves every other record exactly as it came in.
func dockerHealthShape(record json.RawMessage) json.RawMessage {
	var event struct {
		Action       string
		HealthStatus string
	}
	if err := json.Unmarshal(record, &event); err != nil {
		return record
	}
	if event.Action != "health_status" || event.HealthStatus == "" {
		return record
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(record, &fields); err != nil {
		return record
	}
	action, err := json.Marshal("health_status: " + event.HealthStatus)
	if err != nil {
		return record
	}
	// Unmarshal matched the key without regard to case, so replace the one that
	// is actually there instead of assuming Docker's spelling.
	for key := range fields {
		if strings.EqualFold(key, "Action") {
			fields[key] = action
			break
		}
	}

	rewritten, err := json.Marshal(fields)
	if err != nil {
		return record
	}
	return rewritten
}

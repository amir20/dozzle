package logparse

import (
	"encoding/json"
	"strings"

	"github.com/amir20/dozzle/internal/container"
)

// maxJSONPrefixLen caps the text allowed before a trailing JSON object. Console
// encoders put a level, a logger name, a caller and a message there; anything
// longer is prose that happens to end in JSON.
const maxJSONPrefixLen = 256

// parsePrefixedJSON parses the "text, then a JSON object" shape console encoders
// print, e.g. zap's development encoder:
//
//	INFO	finished call	{"grpc.code": "OK", "grpc.time_ms": "0.158"}
//
// The object must start at the first '{' on the line, follow whitespace, and run
// to the end of the line. The prefix is split on tabs: a field that is exactly a
// level becomes "level" (otherwise the prefix is guessed like a plain line), a
// timestamp matching dockerTs is dropped, and the rest is joined into "msg".
// Keys the object already has always win.
func parsePrefixedJSON(message string, dockerTs int64) (container.Fields[any], bool) {
	if len(message) < 3 || message[len(message)-1] != '}' {
		return nil, false
	}
	start := strings.IndexByte(message, '{')
	if start < 1 || start > maxJSONPrefixLen || (message[start-1] != ' ' && message[start-1] != '\t') {
		return nil, false
	}
	prefix := strings.TrimSpace(message[:start])
	prefix = prefix[timestampPrefixLen(prefix, dockerTs):]
	if prefix == "" {
		return nil, false
	}

	tail := []byte(message[start:])
	if !json.Valid(tail) {
		return nil, false
	}
	fields, err := container.ParseFields(tail)
	if err != nil || fields == nil {
		return nil, false
	}

	var level string
	var words []string
	for field := range strings.SplitSeq(prefix, "\t") {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		if level == "" && len(words) == 0 {
			if l := normalizeLogLevel(field); l != "unknown" {
				level = l
				continue
			}
		}
		words = append(words, field)
	}
	if level == "" {
		if l := guessFromString(prefix); l != "unknown" {
			level = l
		}
	}

	data := make(container.Fields[any], 0, len(fields)+2)
	if _, ok := fields.Get("level"); level != "" && !ok {
		data.Set("level", level)
	}
	if _, ok := fields.Get("msg"); len(words) > 0 && !ok {
		data.Set("msg", strings.Join(words, " "))
	}
	return append(data, fields...), true
}

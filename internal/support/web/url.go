package support_web

import (
	"regexp"

	"github.com/amir20/dozzle/internal/container"
)

const (
	URLMarkerStart = "\uE002"
	URLMarkerEnd   = "\uE003"
)

// Standard URL regex pattern to match http/https URLs
var urlRegex = regexp.MustCompile(`(https?:\/\/[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b(?:[-a-zA-Z0-9()@:%_\+.~#?&//=]*))`)

// MarkURLs marks URLs in the logEvent message with special markers
func MarkURLs(logEvent *container.LogEvent) bool {
	matcher := NewPatternMatcher(urlRegex, URLMarkerStart, URLMarkerEnd)
	return matcher.MarkInLogEvent(logEvent)
}

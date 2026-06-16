package support_web

import (
	"regexp"

	"github.com/amir20/dozzle/internal/container"
)

const (
	URLMarkerStart = "\uE002"
	URLMarkerEnd   = "\uE003"
)

var (
	searchMarkerChars = regexp.QuoteMeta(MarkerStart + MarkerEnd)
	urlHostChars      = "[-a-zA-Z0-9@:%._+~#=" + searchMarkerChars + "]"
	urlTLDChars       = "[a-zA-Z0-9()" + searchMarkerChars + "]"
	urlPathChars      = "[-a-zA-Z0-9()@:%_+.~#?&/=" + searchMarkerChars + "]"
	urlTailChars      = "[-a-zA-Z0-9@%_+~#?&/=" + searchMarkerChars + "]"
	urlHostRegex      = `(?:` + urlHostChars + `{1,256}\.` + urlTLDChars + `{1,6}|localhost(?::[0-9]+)?)`
)

// Standard URL regex pattern to match http/https URLs
var urlRegex = regexp.MustCompile(`(https?://` + urlHostRegex + urlPathChars + `*/?(?:` + urlTailChars + `|\b))`)

// MarkURLs marks URLs in the logEvent message with special markers
func MarkURLs(logEvent *container.LogEvent) bool {
	matcher := NewPatternMatcher(urlRegex, URLMarkerStart, URLMarkerEnd)
	return matcher.MarkInLogEvent(logEvent)
}

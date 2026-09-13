package support_web

import "regexp"

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

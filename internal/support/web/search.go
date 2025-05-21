package support_web

import (
	"regexp"
	"strings"

	"github.com/amir20/dozzle/internal/container"
)

const (
	MarkerStart = "\uE000"
	MarkerEnd   = "\uE001"
)

func ParseRegex(search string) (*regexp.Regexp, error) {
	return CreateRegex(search, search == strings.ToLower(search))
}

func Search(re *regexp.Regexp, logEvent *container.LogEvent) bool {
	matcher := NewPatternMatcher(re, MarkerStart, MarkerEnd)
	return matcher.MarkInLogEvent(logEvent)
}

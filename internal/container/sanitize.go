package container

import "strings"

// Highlight markers that Dozzle injects into log messages for HTML rendering
// (see internal/support/web: MarkerStart/End = U+E000/U+E001 for search hits,
// URLMarkerStart/End = U+E002/U+E003 for URLs). They live in the Unicode
// private-use area and must never appear in copied or downloaded plain-text
// logs, where they show up as invisible garbage characters.
const (
	markerSearchStart = ""
	markerSearchEnd   = ""
	markerURLStart    = ""
	markerURLEnd      = ""
)

var highlightMarkerStripper = strings.NewReplacer(
	markerSearchStart, "",
	markerSearchEnd, "",
	markerURLStart, "",
	markerURLEnd, "",
)

// StripHighlightMarkers removes the internal search/URL highlight markers that
// Dozzle injects for HTML rendering. Paths that emit raw log text (clipboard
// copy, downloads) must call it so the markers don't leak into the output.
func StripHighlightMarkers(s string) string {
	return highlightMarkerStripper.Replace(s)
}

// PlainText renders a log event as plain text for clipboard copy, stripping
// ANSI escape sequences and the internal highlight markers. Grouped events
// carry their lines in fragments and have an empty RawMessage, so they are
// expanded one line per fragment; otherwise every grouped multi-line entry
// collapses to a single blank line on copy.
func (e *LogEvent) PlainText() string {
	if e.Type == LogTypeGroup {
		if fragments, ok := e.Message.([]LogFragment); ok {
			lines := make([]string, len(fragments))
			for i, fragment := range fragments {
				lines[i] = StripHighlightMarkers(StripANSI(fragment.Message))
			}
			return strings.Join(lines, "\n")
		}
	}
	return StripHighlightMarkers(StripANSI(e.RawMessage))
}

package container

import "strings"

// PlainText renders a log event as plain text for clipboard copy, stripping
// ANSI escape sequences. Grouped events carry their lines in fragments and
// have an empty RawMessage, so they are expanded one line per fragment;
// otherwise every grouped multi-line entry collapses to a single blank line on
// copy.
func (e *LogEvent) PlainText() string {
	if e.Type == LogTypeGroup {
		if fragments, ok := e.Message.([]LogFragment); ok {
			lines := make([]string, len(fragments))
			for i, fragment := range fragments {
				lines[i] = StripANSI(fragment.Message)
			}
			return strings.Join(lines, "\n")
		}
	}
	return StripANSI(e.RawMessage)
}

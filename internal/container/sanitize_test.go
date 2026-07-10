package container

import "testing"

func TestLogEventPlainText(t *testing.T) {
	tests := []struct {
		name     string
		event    *LogEvent
		expected string
	}{
		{
			name:     "single event returns ANSI-stripped raw message",
			event:    &LogEvent{Type: LogTypeSingle, RawMessage: "\x1b[31mred\x1b[0m text"},
			expected: "red text",
		},
		{
			name:     "complex event returns raw message",
			event:    &LogEvent{Type: LogTypeComplex, RawMessage: `{"level":"info"}`},
			expected: `{"level":"info"}`,
		},
		{
			name: "grouped event expands every fragment line (otherwise lost on copy)",
			event: &LogEvent{
				Type: LogTypeGroup,
				Message: []LogFragment{
					{Message: "Job (17) starting"},
					{Message: "Job (17) done in 0.006s"},
					{Message: "Job (17) completed"},
				},
			},
			expected: "Job (17) starting\nJob (17) done in 0.006s\nJob (17) completed",
		},
		{
			name: "grouped fragments are ANSI-stripped",
			event: &LogEvent{
				Type:    LogTypeGroup,
				Message: []LogFragment{{Message: "\x1b[31mred\x1b[0m"}, {Message: "plain"}},
			},
			expected: "red\nplain",
		},
		{
			// A regex filter marks search hits with U+E000/U+E001 before the
			// event is rendered; those internal markers must not leak into the
			// copied text.
			name: "grouped fragments strip search-highlight markers",
			event: &LogEvent{
				Type: LogTypeGroup,
				Message: []LogFragment{
					{Message: "  at com.\ue000example\ue001.Main(Main.java:10)"},
					{Message: "  at com.\ue000example\ue001.Foo(Foo.java:20)"},
				},
			},
			expected: "  at com.example.Main(Main.java:10)\n  at com.example.Foo(Foo.java:20)",
		},
		{
			name:     "single event strips search-highlight markers",
			event:    &LogEvent{Type: LogTypeSingle, RawMessage: "found \ue000error\ue001 here"},
			expected: "found error here",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.event.PlainText(); got != tt.expected {
				t.Errorf("PlainText() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestStripHighlightMarkers(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"no markers", "plain text", "plain text"},
		{"search markers", "a \ue000b\ue001 c", "a b c"},
		{"url markers", "see \ue002http://example.com\ue003 now", "see http://example.com now"},
		{"mixed search and url markers", "\ue002http://x/\ue000hit\ue001\ue003", "http://x/hit"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := StripHighlightMarkers(tt.input); got != tt.expected {
				t.Errorf("StripHighlightMarkers(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

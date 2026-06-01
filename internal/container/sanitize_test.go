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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.event.PlainText(); got != tt.expected {
				t.Errorf("PlainText() = %q, want %q", got, tt.expected)
			}
		})
	}
}

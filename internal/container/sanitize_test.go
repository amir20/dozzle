package container

import "testing"

func TestSanitizeForPlainText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"strips NUL bytes that truncate clipboard text on Windows", "before\x00after", "beforeafter"},
		{"strips other C0 control bytes", "a\x01b\x07c\x1fd", "abcd"},
		{"preserves tab, newline and carriage return", "a\tb\nc\r\nd", "a\tb\nc\r\nd"},
		{"removes ANSI escape sequences", "\x1b[31mred\x1b[0m text", "red text"},
		{"leaves plain text untouched", "plain log line", "plain log line"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SanitizeForPlainText(tt.input); got != tt.expected {
				t.Errorf("SanitizeForPlainText(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

package notification

import (
	"testing"
)

// These cases are ported from Dozzle Cloud's api/internal/pattern package and
// must keep passing identically on both sides. See pattern.go.

func TestTokenShape(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Short tokens (≤2 chars) preserved
		{"short token - single char", "a", "a"},
		{"short token - two chars", "ab", "ab"},
		{"short token - number", "12", "12"},

		// Numeric tokens
		{"pure number", "12345", "<n>"},
		{"port number", "8080", "<n>"},

		// Timestamp-like tokens
		{"date format", "1/22/2026", "<ts>"},
		{"time format", "10:30:45", "<ts>"},
		{"ip address", "192.168.1.1", "<ts>"},
		{"datetime", "2024-01-15", "<ts>"},

		// ID-like tokens
		{"uuid", "550e8400-e29b-41d4-a716-446655440000", "<id>"},
		{"short hash", "abc123def", "<id>"},
		{"container id", "web-server-1234abcd", "<id>"},

		// Long values
		{"very long string", "this-is-a-very-long-string-that-exceeds-thirty-two-characters", "<v>"},

		// Regular words preserved
		{"word", "error", "error"},
		{"uppercase word", "ERROR", "ERROR"},
		{"mixed case", "Warning", "Warning"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tokenShape(tt.input)
			if result != tt.expected {
				t.Errorf("tokenShape(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExtractPattern(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "simple message",
			input:    "Connection refused",
			expected: "Connection refused",
		},
		{
			name:     "message with port",
			input:    "Listening on port 8080",
			expected: "Listening on port <n>",
		},
		{
			name:     "message with ip and port",
			input:    "Connected to 192.168.1.100:5432",
			expected: "Connected to <ts>",
		},
		{
			name:     "log with timestamp",
			input:    "[2024-01-15 10:30:45] Error occurred",
			expected: "[ <ts> <ts> ] Error occurred",
		},
		{
			name:     "key-value pairs",
			input:    "user=admin action=login status=success",
			expected: "user = admin action = login status = success",
		},
		{
			name:     "message with container id",
			input:    "Container abc123def456 started",
			expected: "Container <id> started",
		},
		{
			name:     "bracketed content",
			input:    "[INFO] Starting service",
			expected: "[ INFO ] Starting service",
		},
		{
			name:     "parenthetical content",
			input:    "Error (code 500) occurred",
			expected: "Error ( code <n> ) occurred",
		},
		{
			name:     "comma separated values",
			input:    "hosts: server1, server2, server3",
			expected: "hosts: server1 , server2 , server3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractPattern(tt.input)
			if result != tt.expected {
				t.Errorf("ExtractPattern(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExtractPattern_SimilarLogsProduceSamePattern(t *testing.T) {
	// These log messages should all produce the same pattern
	logs := []string{
		"Connection refused to database at 10.0.0.5:5432",
		"Connection refused to database at 192.168.1.1:5432",
		"Connection refused to database at 172.16.0.100:5432",
	}

	var patterns []string
	for _, log := range logs {
		patterns = append(patterns, ExtractPattern(log))
	}

	for i := 1; i < len(patterns); i++ {
		if patterns[i] != patterns[0] {
			t.Errorf("Expected same pattern for similar logs\nLog 0: %q -> %q\nLog %d: %q -> %q",
				logs[0], patterns[0], i, logs[i], patterns[i])
		}
	}
}

func TestExtractPattern_DifferentErrorsProduceDifferentPatterns(t *testing.T) {
	// Two failures from the same container must not collapse into one pattern,
	// or the cooldown would suppress the second one and Cloud would never see it.
	a := ExtractPattern("FATAL: could not extend file: No space left on device")
	b := ExtractPattern("FATAL: connection refused")

	if a == b {
		t.Errorf("expected distinct patterns, both were %q", a)
	}
	if HashPattern(a) == HashPattern(b) {
		t.Error("expected distinct pattern hashes")
	}
}

func TestHashPattern(t *testing.T) {
	if got := HashPattern(""); got != "" {
		t.Errorf("HashPattern(\"\") = %q, want empty", got)
	}

	a := HashPattern("Connection refused")
	if len(a) != 64 {
		t.Errorf("HashPattern returned %d chars, want 64", len(a))
	}
	if b := HashPattern("Connection refused"); a != b {
		t.Error("HashPattern is not stable across calls")
	}
}

func TestMessageText(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{"nil", nil, ""},
		{"string", "connection refused", "connection refused"},
		{"grouped strings", []any{"line one", "line two"}, "line one\nline two"},
		{"grouped fragments", []any{map[string]any{"m": "line one"}}, "line one"},
		{"structured message", map[string]any{"message": "disk full"}, "disk full"},
		{"structured msg", map[string]any{"msg": "disk full"}, "disk full"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MessageText(tt.input); got != tt.expected {
				t.Errorf("MessageText(%v) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestTokenize(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "simple words",
			input:    "hello world",
			expected: []string{"hello", "world"},
		},
		{
			name:     "with brackets",
			input:    "[INFO] message",
			expected: []string{"[", "INFO", "]", "message"},
		},
		{
			name:     "with parentheses",
			input:    "func(arg)",
			expected: []string{"func", "(", "arg", ")"},
		},
		{
			name:     "with curly braces",
			input:    "{key: value}",
			expected: []string{"{", "key:", "value", "}"},
		},
		{
			name:     "with equals",
			input:    "key=value",
			expected: []string{"key", "=", "value"},
		},
		{
			name:     "with commas",
			input:    "a, b, c",
			expected: []string{"a", ",", "b", ",", "c"},
		},
		{
			name:     "with semicolons",
			input:    "a; b; c",
			expected: []string{"a", ";", "b", ";", "c"},
		},
		{
			name:     "multiple spaces",
			input:    "hello    world",
			expected: []string{"hello", "world"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tokenize(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("tokenize(%q) returned %d tokens, want %d\ngot: %v\nwant: %v",
					tt.input, len(result), len(tt.expected), result, tt.expected)
				return
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("tokenize(%q)[%d] = %q, want %q",
						tt.input, i, result[i], tt.expected[i])
				}
			}
		})
	}
}

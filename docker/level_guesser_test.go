package docker

import (
	"testing"
)

func TestGuessLogLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"ERROR: Something went wrong", "error"},
		{"WARN: Something might be wrong", "warn"},
		{"INFO: Something happened", "info"},
		{"debug: Something happened", "debug"},
		{"debug Something happened", "debug"},
		{"TRACE: Something happened", "trace"},
		{"FATAL: Something happened", "fatal"},
		{"level=error Something went wrong", "error"},
		{"[ERROR] Something went wrong", "error"},
		{"[error] Something went wrong", "error"},
		{"[ ERROR ] Something went wrong", "error"},
		{"[error] Something went wrong", "error"},
		{"[test] [error] Something went wrong", "error"},
		{"[foo] [ ERROR] Something went wrong", "error"},
		{"123 ERROR Something went wrong", "error"},
		{"123 Something went wrong", ""},
	}

	for _, test := range tests {
		logEvent := &LogEvent{
			Message: test.input,
		}
		if level := guessLogLevel(logEvent); level != test.expected {
			t.Errorf("guessLogLevel(%s) = %s, want %s", test.input, level, test.expected)
		}
	}

}

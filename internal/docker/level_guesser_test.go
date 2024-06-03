package docker

import (
	"testing"

	orderedmap "github.com/wk8/go-ordered-map/v2"
)

func TestGuessLogLevel(t *testing.T) {
	tests := []struct {
		input    any
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
		{map[string]interface{}{"level": "info"}, "info"},
		{map[string]interface{}{"level": "INFO"}, "info"},
		{map[string]string{"level": "info"}, "info"},
		{map[string]string{"level": "INFO"}, "info"},
		{orderedmap.New[string, string](
			orderedmap.WithInitialData(
				orderedmap.Pair[string, string]{Key: "key", Value: "value"},
				orderedmap.Pair[string, string]{Key: "level", Value: "info"},
			),
		), "info"},
		{orderedmap.New[string, any](
			orderedmap.WithInitialData(
				orderedmap.Pair[string, any]{Key: "key", Value: "value"},
				orderedmap.Pair[string, any]{Key: "level", Value: "info"},
			),
		), "info"},
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

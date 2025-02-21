package container

import (
	"encoding/json"
	"testing"

	orderedmap "github.com/wk8/go-ordered-map/v2"
)

func TestGuessLogLevel(t *testing.T) {
	var nilOrderedMap *orderedmap.OrderedMap[string, any]
	tests := []struct {
		input    any
		expected string
	}{
		{"2024/12/30 12:21AM INF this is a test", "info"},
		{"2025-01-07 22:00:08,059: DEBUG/MainProcess TaskPool: ", "debug"},
		{"Some test with error-test", "error"},
		{"2024-12-30T17:43:16Z DBG loggging debug from here", "debug"},
		{"2025-01-07 15:40:15,784 LL=\"ERROR\" some message", "error"},
		{"2025-01-07 15:40:15,784 LL=\"WARN\" some message", "warn"},
		{"2025-01-07 15:40:15,784 LL=\"INFO\" some message", "info"},
		{"2025-01-07 15:40:15,784 LL=\"DEBUG\" some message", "debug"},
		{"ERROR: Something went wrong", "error"},
		{"WARN: Something might be wrong", "warn"},
		{"INFO: Something happened", "info"},
		{"debug: Something happened", "debug"},
		{"debug Something happened", "debug"},
		{"TRACE: Something happened", "trace"},
		{"FATAL: Something happened", "fatal"},
		{"[ERROR] Something went wrong", "error"},
		{"[error] Something went wrong", "error"},
		{"[ ERROR ] Something went wrong", "error"},
		{"[error] Something went wrong", "error"},
		{"[test] [error] Something went wrong", "error"},
		{"[foo] [ ERROR] Something went wrong", "error"},
		{"123 ERROR Something went wrong", "error"},
		{"123 Something went wrong", "unknown"},
		{"DBG Something went wrong", "debug"},
		{"DBG with more error=msg", "debug"},
		{"inf Something went wrong", "info"},
		{"crit: Something went wrong", "fatal"},
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
		{orderedmap.New[string, string](
			orderedmap.WithInitialData(
				orderedmap.Pair[string, string]{Key: "key", Value: "value"},
				orderedmap.Pair[string, string]{Key: "severity", Value: "info"},
			),
		), "info"},
		{orderedmap.New[string, any](
			orderedmap.WithInitialData(
				orderedmap.Pair[string, any]{Key: "key", Value: "value"},
				orderedmap.Pair[string, any]{Key: "severity", Value: "info"},
			),
		), "info"},
		{nilOrderedMap, "unknown"},
		{nil, "unknown"},
	}

	for _, test := range tests {
		name, _ := json.Marshal(test.input)
		t.Run(string(name), func(t *testing.T) {
			actual := guessLogLevel(&LogEvent{Message: test.input})
			if actual != test.expected {
				t.Errorf("Expected %s, got %s", test.expected, actual)
			}
		})
	}
}

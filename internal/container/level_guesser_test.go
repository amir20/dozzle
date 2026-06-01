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
		{"[21:01:45] [WRN] this is a test", "warn"},
		{"2026-01-05 12:13:24,566 - retry.api                        (7fd8ad34eb30) :  WARNING (api:40) - HTTPSConnectionPool(host='podnapisi.net', port=443): Max retries exceeded", "warn"},
		{"2026-01-05 08:21:16,511 - root                             (7fd8bf822b30) :  INFO (get_providers:408) - Throttling podnapisi for 10 minutes", "info"},
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
		{orderedmap.New[string, string](
			orderedmap.WithInitialData(
				orderedmap.Pair[string, string]{Key: "key", Value: "value"},
				orderedmap.Pair[string, string]{Key: "@l", Value: "info"},
			),
		), "info"},
		{orderedmap.New[string, any](
			orderedmap.WithInitialData(
				orderedmap.Pair[string, any]{Key: "key", Value: "value"},
				orderedmap.Pair[string, any]{Key: "@l", Value: "debug"},
			),
		), "debug"},
		{orderedmap.New[string, string](
			orderedmap.WithInitialData(
				orderedmap.Pair[string, string]{Key: "@l", Value: "error"},
				orderedmap.Pair[string, string]{Key: "@t", Value: "2024-01-01T00:00:00Z"},
			),
		), "error"},
		// Zigbee2MQTT-style: bracketed timestamp + " <level>:" inside the line.
		{"[2025-12-22 12:00:00] info: 	z2m: started", "info"},
		{"[2025-12-22 12:00:00] warn: 	z2m: queue full", "warn"},
		{"[2025-12-22 12:00:00] error: 	z2m: connection failed", "error"},
		{"[2025-12-22 12:00:00] debug: 	z2m: handling message", "debug"},
		// "<tag>:<level> " style (no space before the colon).
		{"Zigbee2MQTT:info  2025-12-22 12:00:00: started", "info"},
		{"Zigbee2MQTT:warn  2025-12-22 12:00:00: queue full", "warn"},
		{"Zigbee2MQTT:error  2025-12-22 12:00:00: failure", "error"},
		// Pipe-delimited
		{"2024-01-01 12:00:00 | ERROR | something went wrong", "error"},
		{"2024-01-01 12:00:00 | INFO | starting up", "info"},
		{"app INFO| starting up", "info"},
		// Single-letter bracket levels
		{"[I] starting up", "info"},
		{"[E] something went wrong", "error"},
		{"[W] something might be wrong", "warn"},
		{"[D] debugging info", "debug"},
		{"[F] fatal error", "fatal"},
		{"[T] trace message", "trace"},
		{"[V] verbose message", "trace"},
		{"12:00:00 [I] starting up", "info"},
		// Issue #4768: a real level prefix must win over a level word in the message body.
		{"INFO: connection established, retrying after error: timeout", "info"},
		{"INFO handling request failed with error: bad gateway", "info"},
		{"2024-12-30T17:43:16Z INF some message about an error: foo", "info"},
		{"INFO request completed but contained ERROR token", "info"},
		{"WARN: connection error: retrying", "warn"},
		// Symmetric: an ERROR prefix still wins over a later info word.
		{"ERROR: handler failed, info: will retry", "error"},
		// Equal confidence between two different levels -> unknown (don't guess).
		{"saw info: here and error: there", "unknown"},
		{"[INFO] [DEBUG] both bracketed", "unknown"},
		{orderedmap.New[string, any](
			orderedmap.WithInitialData(
				orderedmap.Pair[string, any]{Key: "key", Value: "value"},
				orderedmap.Pair[string, any]{Key: "level", Value: "warning"},
			),
		), "warn"},
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

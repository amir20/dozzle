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
		// klog / glog format used across the Kubernetes toolchain
		{"E0806 14:55:55.980915       1 fsHandler.go:121] failed to collect filesystem stats", "error"},
		{"W0806 14:54:52.068675       1 info.go:52] Couldn't collect info from any of the files", "warn"},
		{"I0808 13:37:10.853121       1 cadvisor.go:182] Starting cAdvisor version: v0.60.5", "info"},
		{"F0806 14:55:55.980915       1 main.go:10] exiting", "fatal"},
		// the level letter must be glued to a full klog timestamp, not just any capital
		{"E0806 is not a klog line", "unknown"},
		{"Warning0806 14:55:55.980915 not klog either", "warn"},
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
		// .NET / Serilog / Microsoft.Extensions.Logging spell out "Information".
		{"Information: service started", "info"},
		{"[Information] service started", "info"},
		{"2024-12-30T17:43:16Z Information starting up", "info"},
		{orderedmap.New[string, string](
			orderedmap.WithInitialData(
				orderedmap.Pair[string, string]{Key: "level", Value: "Information"},
			),
		), "info"},
		{orderedmap.New[string, any](
			orderedmap.WithInitialData(
				orderedmap.Pair[string, any]{Key: "@l", Value: "Information"},
			),
		), "info"},
		// logfmt style key=value, used by logrus, go-kit, slog, Traefik, Grafana, containerd.
		{`time="2024-01-01T00:00:00Z" level=error msg="failed to connect"`, "error"},
		{`ts=2024-01-01T00:00:00.000Z caller=main.go:10 level=warn msg="disk almost full"`, "warn"},
		{"time=2024-01-01T00:00:00Z level=INFO msg=\"server started\"", "info"},
		{"2024-01-01 12:00:00 level=debug handling request", "debug"},
		{`level="warning" some message`, "warn"},
		{"lvl=trace some message", "trace"},
		{"log_level=error something broke", "error"},
		{"loglevel=fatal giving up", "fatal"},
		{"severity: ERROR something broke", "error"},
		{"levelname=INFO started", "info"},
		// The key must hold the level, not just look like it.
		{"level=whatever some message", "unknown"},
		{"lowlevel=error is not a level key", "unknown"},
		// An explicit key beats a level word in the message.
		{`level=info msg="connection error: retrying"`, "info"},
		// ...but a real prefix still beats the key/value pair.
		{`ERROR: request failed level=info`, "error"},
		// Syslog priority prefix (RFC 3164/5424), emitted by systemd and rsyslog.
		{"<11>Jan  1 00:00:00 host app: connection refused", "error"},
		{"<30>Jan  1 00:00:00 host app: listening on :80", "info"},
		{"<28>Jan  1 00:00:00 host app: retrying", "warn"},
		{"<7>Jan  1 00:00:00 host app: cache hit", "debug"},
		{"<1>1 2024-01-01T00:00:00Z host app - - - shutting down", "fatal"},
		{"<190>Jan  1 00:00:00 host app: local7 info message", "info"},
		{"<999> is not a syslog priority", "unknown"},
		{"<html> is not a syslog priority", "unknown"},
		// Syslog level names, used by nginx, haproxy, php-fpm and postfix.
		{"2023/01/01 12:00:00 [notice] 1#1: start worker process 30", "info"},
		{"2023/01/01 12:00:00 [emerg] 1#1: bind() to 0.0.0.0:80 failed (98: Address in use)", "fatal"},
		{"2023/01/01 12:00:00 [alert] 1#1: worker process 30 exited on signal 9", "fatal"},
		{"NOTICE: fpm is running, pid 1", "info"},
		{"panic: runtime error: invalid memory address", "fatal"},
		{"level=alert disk is full", "fatal"},
		// "alert" is an ordinary word, so it only counts where the shape is unambiguous.
		{"alert sent to slack channel", "unknown"},
		{"Alerts: 5 fired in the last hour", "unknown"},
		{"Received notice from upstream server", "unknown"},
		// bunyan / pino write numeric levels.
		{orderedmap.New[string, any](
			orderedmap.WithInitialData(
				orderedmap.Pair[string, any]{Key: "level", Value: float64(30)},
				orderedmap.Pair[string, any]{Key: "msg", Value: "started"},
			),
		), "info"},
		{orderedmap.New[string, any](
			orderedmap.WithInitialData(orderedmap.Pair[string, any]{Key: "level", Value: float64(10)}),
		), "trace"},
		{orderedmap.New[string, any](
			orderedmap.WithInitialData(orderedmap.Pair[string, any]{Key: "level", Value: float64(20)}),
		), "debug"},
		{orderedmap.New[string, any](
			orderedmap.WithInitialData(orderedmap.Pair[string, any]{Key: "level", Value: float64(40)}),
		), "warn"},
		{orderedmap.New[string, any](
			orderedmap.WithInitialData(orderedmap.Pair[string, any]{Key: "level", Value: float64(50)}),
		), "error"},
		{orderedmap.New[string, any](
			orderedmap.WithInitialData(orderedmap.Pair[string, any]{Key: "level", Value: float64(60)}),
		), "fatal"},
		// Custom levels land in the band below them.
		{orderedmap.New[string, any](
			orderedmap.WithInitialData(orderedmap.Pair[string, any]{Key: "level", Value: float64(35)}),
		), "info"},
		// Below 10 the numeric scales contradict each other, so don't guess.
		{orderedmap.New[string, any](
			orderedmap.WithInitialData(orderedmap.Pair[string, any]{Key: "level", Value: float64(3)}),
		), "unknown"},
		// A key that cannot be resolved falls through to the next one.
		{orderedmap.New[string, any](
			orderedmap.WithInitialData(
				orderedmap.Pair[string, any]{Key: "level", Value: float64(3)},
				orderedmap.Pair[string, any]{Key: "severity", Value: "error"},
			),
		), "error"},
		{orderedmap.New[string, string](
			orderedmap.WithInitialData(orderedmap.Pair[string, string]{Key: "level", Value: "50"}),
		), "error"},
		// A number under "severity" is a syslog or OpenTelemetry severity, which
		// counts the other way around, so it is not read as a bunyan level.
		{orderedmap.New[string, any](
			orderedmap.WithInitialData(orderedmap.Pair[string, any]{Key: "severity", Value: float64(17)}),
		), "unknown"},
		// Level keys are matched case-insensitively and ignoring separators.
		{orderedmap.New[string, string](
			orderedmap.WithInitialData(orderedmap.Pair[string, string]{Key: "Level", Value: "Error"}),
		), "error"},
		{orderedmap.New[string, string](
			orderedmap.WithInitialData(orderedmap.Pair[string, string]{Key: "log.level", Value: "debug"}),
		), "debug"},
		{orderedmap.New[string, string](
			orderedmap.WithInitialData(orderedmap.Pair[string, string]{Key: "log_level", Value: "warn"}),
		), "warn"},
		{orderedmap.New[string, string](
			orderedmap.WithInitialData(orderedmap.Pair[string, string]{Key: "logLevel", Value: "trace"}),
		), "trace"},
		{orderedmap.New[string, string](
			orderedmap.WithInitialData(orderedmap.Pair[string, string]{Key: "lvl", Value: "info"}),
		), "info"},
		// Python's json logger writes levelname; OpenTelemetry writes severityText.
		{orderedmap.New[string, string](
			orderedmap.WithInitialData(orderedmap.Pair[string, string]{Key: "levelname", Value: "WARNING"}),
		), "warn"},
		{orderedmap.New[string, string](
			orderedmap.WithInitialData(orderedmap.Pair[string, string]{Key: "severityText", Value: "ERROR"}),
		), "error"},
		// Syslog and GCP severity names.
		{orderedmap.New[string, string](
			orderedmap.WithInitialData(orderedmap.Pair[string, string]{Key: "severity", Value: "NOTICE"}),
		), "info"},
		{orderedmap.New[string, string](
			orderedmap.WithInitialData(orderedmap.Pair[string, string]{Key: "severity", Value: "EMERGENCY"}),
		), "fatal"},
		{orderedmap.New[string, string](
			orderedmap.WithInitialData(orderedmap.Pair[string, string]{Key: "severity", Value: "CRITICAL"}),
		), "fatal"},
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

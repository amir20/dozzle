package logparse

import (
	"testing"

	"github.com/amir20/dozzle/internal/container"
	"github.com/stretchr/testify/assert"
)

const ts = "2026-10-02T15:04:12.502114Z "

// Every shape the level guesser recognizes, at error and fatal and at levels
// that are neither. IsErrorLine must agree with the full guess on all of them:
// its prefilter may only ever skip lines the guess would not call an error.
var errorLineCorpus = []string{
	"ERROR: Something went wrong",
	"error: lowercase prefix",
	"ERR something",
	"FAIL: test suite",
	"FATAL: out of memory",
	"CRIT disk full",
	"CRITICAL disk full",
	"SEVERE: connection lost",
	"E0806 14:55:55.980915       1 fsHandler.go:121] failed to collect",
	"F0806 14:55:55.980915       1 main.go:1] fatal",
	"[2026-09-28 09:58:00.046 ERR] [MalwareBlocker] Error creating",
	"[09:58:00 FTL] Shutting down",
	"[ERROR] bracketed",
	"[ error ] spaced bracket",
	"[E] single letter",
	"[F] single letter fatal",
	"› ✖  error     consola style",
	"Zigbee2MQTT:error 2026-10-02 failed",
	"::ERROR:: tagged",
	`2025-01-07 15:40:15,784 LL="ERROR" some message`,
	"Some test with error-test",
	"request failed with error: boom",
	"123 ERROR foo",
	`{"level":"error","msg":"json"}`,
	`{"level":50,"msg":"pino error"}`,
	`{"level":60,"msg":"pino fatal"}`,
	`{"level": 50,"msg":"spaced pino"}`,
	`{"severity":"CRITICAL","msg":"x"}`,
	`{"severityText":"ERROR","body":"otel"}`,
	`{"severityNumber":17,"body":"otel number"}`,
	`{"severityNumber":21,"body":"otel fatal number"}`,
	`{"@l":"Error","@m":"serilog compact"}`,
	"level=error msg=logfmt",
	"\x1b[31mERROR\x1b[0m colored",
	// Not errors.
	"INFO: all good",
	"WARN: careful",
	"[W] warn letter",
	"I0806 14:55:55.980915       1 main.go:1] info",
	"GET /v1/orders 200 18ms",
	`{"level":30,"msg":"pino info"}`,
	`{"severityNumber":9,"body":"otel info"}`,
	"level=info msg=logfmt",
	"",
}

func TestIsErrorLineAgreesWithGuess(t *testing.T) {
	for _, message := range errorLineCorpus {
		t.Run(message, func(t *testing.T) {
			line := ts + message
			level := guessLogLevel(createEvent(line, container.STDOUT))
			want := level == "error" || level == "fatal"
			assert.Equal(t, want, IsErrorLine(line, container.STDOUT), "guessed level %q", level)
		})
	}
}

func TestMayBeErrorRejectsPlainLines(t *testing.T) {
	for _, message := range []string{
		"GET /v1/orders?page=2 200 18ms",
		"pool stats active=8 idle=12 waiting=0",
		"INFO request handled path=/v1/orders/1 status=200",
		`{"level":30,"msg":"ok"}`,
	} {
		assert.False(t, mayBeError(ts+message), message)
	}
}

func BenchmarkIsErrorLine(b *testing.B) {
	plain := ts + "INFO request handled path=/v1/orders/123456 status=200 duration=12ms user=abcdef123456"
	b.Run("plain", func(b *testing.B) {
		for b.Loop() {
			IsErrorLine(plain, container.STDOUT)
		}
	})
	b.Run("guessed", func(b *testing.B) {
		for b.Loop() {
			guessLogLevel(createEvent(plain, container.STDOUT))
		}
	})
}

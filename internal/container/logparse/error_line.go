package logparse

import (
	"strings"

	"github.com/amir20/dozzle/internal/container"
)

// IsErrorLine reports whether line, as Docker returns it with its timestamp
// prefix, is one the stream would show at the error or fatal level.
//
// It exists for counting, where running the full level guesser on every line
// costs more than reading the log. Most lines carry none of the words an error
// level is spelled with, so a single byte scan rules them out and only the few
// that remain pay for the real guess. That keeps the answer identical to the
// level the same line gets in the viewer.
func IsErrorLine(line string, std container.StdType) bool {
	if !mayBeError(line) {
		return false
	}
	switch guessLogLevel(createEvent(line, std)) {
	case "error", "fatal":
		return true
	}
	return false
}

// mayBeError is a superset test for the error and fatal aliases in logLevels
// (error, err, fail, fatal, sev, severe, crit, critical), their single-letter
// forms, and the numeric levels of pino and OpenTelemetry. A false negative
// would hide an error from the count, so anything ambiguous passes.
func mayBeError(line string) bool {
	message := line
	if _, after, ok := strings.Cut(line, " "); ok {
		message = after
	}

	// klog: "E0806 14:55:55.980915 ...", glued to the letter, no separator.
	if len(message) > 1 && (message[0] == 'E' || message[0] == 'F') && isDigit(message[1]) {
		return true
	}

	for i := 0; i+2 < len(message); i++ {
		// ORing in 0x20 lowercases ASCII letters and leaves every byte that could
		// alias one of the letters below unchanged.
		switch message[i] | 0x20 {
		case 'e':
			if message[i+1]|0x20 == 'r' && message[i+2]|0x20 == 'r' {
				return true
			}
		case 'f':
			if i+3 < len(message) && message[i+1]|0x20 == 'a' {
				c2, c3 := message[i+2]|0x20, message[i+3]|0x20
				if (c2 == 'i' && c3 == 'l') || (c2 == 't' && c3 == 'a') {
					return true
				}
			}
		case 's':
			if message[i+1]|0x20 == 'e' && message[i+2]|0x20 == 'v' {
				return true
			}
		case 'c':
			if i+3 < len(message) && message[i+1]|0x20 == 'r' && message[i+2]|0x20 == 'i' && message[i+3]|0x20 == 't' {
				return true
			}
		case '[' | 0x20:
			// '[' | 0x20 is '{', so this arm also sees '{'. Both are cheap to check.
			if message[i] == '[' && (message[i+1] == 'E' || message[i+1] == 'F') && message[i+2] == ']' {
				return true
			}
		}
	}

	// Numeric levels only mean anything inside JSON.
	if strings.HasPrefix(message, "{") {
		return strings.Contains(message, `"level":5`) ||
			strings.Contains(message, `"level":6`) ||
			strings.Contains(message, `"level": 5`) ||
			strings.Contains(message, `"level": 6`) ||
			strings.Contains(message, `"severityNumber"`)
	}
	return false
}

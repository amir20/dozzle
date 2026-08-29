package notification

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"unicode"
)

// Log pattern extraction, kept byte-for-byte in sync with Dozzle Cloud's
// api/internal/pattern package. Both sides must derive the same pattern from the
// same line: the client uses it to collapse a repeating log into one notification
// plus a suppressed count, and the server uses it to group notifications into
// incidents. If the two implementations drift, a client-side rollup and the
// server's own grouping stop lining up. Change them together.

// tokenShape converts a token to its canonical shape for pattern matching.
// Short tokens (≤2 chars) are preserved as-is.
// Numeric-only tokens become <n>.
// Timestamp-like tokens (numbers with /, :, ., -) become <ts>.
// Alphanumeric tokens that look like IDs become <id>.
// Very long tokens become <v>.
func tokenShape(token string) string {
	runeCount := 0
	hasLetter := false
	hasDigit := false
	hasSpecial := false

	for _, r := range token {
		runeCount++
		switch {
		case unicode.IsLetter(r):
			hasLetter = true
		case unicode.IsDigit(r):
			hasDigit = true
		case r == '-' || r == '_' || r == ':' || r == '/' || r == '.':
			hasSpecial = true
		}
	}

	if runeCount <= 2 {
		return token
	}

	switch {
	case !hasLetter && hasDigit && hasSpecial:
		return "<ts>" // "1/22/2026", "10:30:45", "192.168.1.1"
	case !hasLetter && hasDigit:
		return "<n>" // "12345"
	case hasLetter && hasDigit && runeCount > 8:
		return "<id>" // UUIDs, hashes, etc.
	case hasLetter && hasDigit && hasSpecial:
		return "<id>" // Mixed alphanumeric with special chars
	case runeCount > 32:
		return "<v>" // Very long values
	default:
		return token
	}
}

// ExtractPattern takes a log message and returns its pattern by tokenizing
// and replacing variable parts with placeholders.
func ExtractPattern(message string) string {
	if message == "" {
		return ""
	}

	tokens := tokenize(message)
	result := make([]string, len(tokens))

	for i, token := range tokens {
		result[i] = tokenShape(token)
	}

	return strings.Join(result, " ")
}

// tokenize splits a log message into tokens, preserving punctuation as separate tokens
// where appropriate for pattern matching.
func tokenize(message string) []string {
	var tokens []string
	var current strings.Builder

	flush := func() {
		if current.Len() > 0 {
			tokens = append(tokens, current.String())
			current.Reset()
		}
	}

	for _, r := range message {
		switch {
		case unicode.IsSpace(r):
			flush()
		case r == '[' || r == ']' || r == '(' || r == ')' || r == '{' || r == '}':
			// Brackets are separate tokens
			flush()
			tokens = append(tokens, string(r))
		case r == '=' || r == ',' || r == ';':
			// Key-value separators are separate tokens
			flush()
			tokens = append(tokens, string(r))
		default:
			current.WriteRune(r)
		}
	}
	flush()

	return tokens
}

// HashPattern returns the stable hash used to key a pattern. Matches the
// server's hashing so the same line produces the same key on both sides.
func HashPattern(pattern string) string {
	if pattern == "" {
		return ""
	}
	h := sha256.Sum256([]byte(pattern))
	return hex.EncodeToString(h[:])
}

// MessageText flattens a NotificationLog message into the plain text that
// pattern extraction runs on. Messages arrive as a string for simple and
// grouped logs, or as a map for structured (JSON) logs.
func MessageText(msg any) string {
	if msg == nil {
		return ""
	}
	switch v := msg.(type) {
	case string:
		return v
	case []any:
		parts := make([]string, len(v))
		for i, elem := range v {
			switch e := elem.(type) {
			case string:
				parts[i] = e
			case map[string]any:
				if m, ok := e["m"]; ok {
					parts[i] = fmt.Sprintf("%v", m)
				} else {
					parts[i] = fmt.Sprintf("%v", e)
				}
			default:
				parts[i] = fmt.Sprintf("%v", elem)
			}
		}
		return strings.Join(parts, "\n")
	case map[string]any:
		for _, key := range []string{"message", "msg", "text", "log"} {
			if val, ok := v[key]; ok {
				return fmt.Sprintf("%v", val)
			}
		}
	}
	return fmt.Sprintf("%v", msg)
}

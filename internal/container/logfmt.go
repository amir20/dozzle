package container

import (
	"errors"
	"strconv"
	"strings"

	orderedmap "github.com/wk8/go-ordered-map/v2"
)

// ParseLogFmt parses a log entry in logfmt format and returns a map of key-value pairs.
func ParseLogFmt(log string) (*orderedmap.OrderedMap[string, string], error) {
	result := orderedmap.New[string, string]()
	var key, value string
	inQuotes, escaping, isKey := false, false, true
	start := 0

	for i := 0; i < len(log); i++ {
		char := log[i]
		if isKey {
			if char == '=' {
				if start >= i {
					return nil, errors.New("invalid format: key is empty")
				}
				key = log[start:i]
				isKey = false
				start = i + 1
			} else if char == ' ' {
				if i > start {
					return nil, errors.New("invalid format: unexpected space in key")
				}
			}

		} else {
			if inQuotes {
				if escaping {
					escaping = false
				} else if char == '\\' {
					escaping = true
				} else if char == '"' {
					value = unescapeQuoted(log[start-1 : i+1])
					result.Set(key, value)
					inQuotes = false
					isKey = true
					start = i + 2
				}
			} else {
				if char == '"' {
					inQuotes = true
					start = i + 1
				} else if char == ' ' {
					value = log[start:i]
					result.Set(key, value)
					isKey = true
					start = i + 1
				}
			}
		}
	}

	// Handle the last key-value pair if there is no trailing space
	if !isKey && start < len(log) {
		if inQuotes {
			return nil, errors.New("invalid format: unclosed quotes")
		}
		value = log[start:]
		result.Set(key, value)
	} else if isKey && start < len(log) {
		return nil, errors.New("invalid format: unexpected key without value")
	}

	if !isKey {
		if inQuotes {
			return nil, errors.New("invalid format: unclosed quotes")
		}
		value = log[start:]
		result.Set(key, value)
	}

	return result, nil
}

// unescapeQuoted decodes a quoted logfmt value (including the surrounding
// quotes) so escape sequences like \" and \\ produced by logfmt encoders
// (logrus, go-kit, go-logfmt) yield the original text instead of leaking
// backslashes into the parsed value. If the sequence is not decodable, the
// raw content between the quotes is returned unchanged.
func unescapeQuoted(quoted string) string {
	raw := quoted[1 : len(quoted)-1]
	if !strings.ContainsRune(raw, '\\') {
		return raw
	}
	if unquoted, err := strconv.Unquote(quoted); err == nil {
		return unquoted
	}
	return raw
}

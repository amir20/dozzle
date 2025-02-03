package container

import (
	"errors"

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
					value = log[start:i]
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

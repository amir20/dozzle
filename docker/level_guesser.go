package docker

import (
	"regexp"
	"strings"
)

var KEY_VALUE_REGEX = regexp.MustCompile(`level=(\w+)`)
var ANSI_COLOR_REGEX = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func guessLogLevel(logEvent *LogEvent) string {
	switch value := logEvent.Message.(type) {
	case string:
		levels := []string{"error", "warn", "warning", "info", "debug", "trace", "fatal"}
		stripped := ANSI_COLOR_REGEX.ReplaceAllString(value, "") // remove ansi color codes
		for _, level := range levels {
			if match, _ := regexp.MatchString("(?i)^"+level+"[^a-z]", stripped); match {
				return level
			}

			if strings.Contains(value, "["+strings.ToUpper(level)+"]") {
				return level
			}

			if strings.Contains(value, " "+strings.ToUpper(level)+" ") {
				return level
			}
		}

		if matches := KEY_VALUE_REGEX.FindStringSubmatch(value); matches != nil {
			return matches[1]
		}

	case map[string]interface{}:
		if level, ok := value["level"].(string); ok {
			return level
		}
	}

	return ""
}

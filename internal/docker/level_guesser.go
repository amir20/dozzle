package docker

import (
	"regexp"
	"strings"
)

var KEY_VALUE_REGEX = regexp.MustCompile(`level=(\w+)`)
var LOG_LEVELS = []string{"error", "warn", "warning", "info", "debug", "trace", "fatal"}
var LOG_LEVELS_PLAIN = map[string]*regexp.Regexp{}
var LOG_LEVEL_BRACKET = map[string]*regexp.Regexp{}

func init() {
	for _, level := range LOG_LEVELS {
		LOG_LEVELS_PLAIN[level] = regexp.MustCompile("(?i)^" + level + "[^a-z]")
	}

	for _, level := range LOG_LEVELS {
		LOG_LEVEL_BRACKET[level] = regexp.MustCompile("(?i)\\[ ?" + level + " ?\\]")
	}
}

func guessLogLevel(logEvent *LogEvent) string {
	switch value := logEvent.Message.(type) {
	case string:
		for _, level := range LOG_LEVELS {
			if LOG_LEVELS_PLAIN[level].MatchString(value) {
				return level
			}

			if LOG_LEVEL_BRACKET[level].MatchString(value) {
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

	case map[string]string:
		if level, ok := value["level"]; ok {
			return level
		}
	}

	return ""
}

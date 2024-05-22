package docker

import (
	"regexp"
	"strings"

	"github.com/iancoleman/orderedmap"
	log "github.com/sirupsen/logrus"
)

var keyValueRegex = regexp.MustCompile(`level=(\w+)`)
var logLevels = []string{"error", "warn", "warning", "info", "debug", "trace", "fatal"}
var plainLevels = map[string]*regexp.Regexp{}
var bracketLevels = map[string]*regexp.Regexp{}

func init() {
	for _, level := range logLevels {
		plainLevels[level] = regexp.MustCompile("(?i)^" + level + "[^a-z]")
	}

	for _, level := range logLevels {
		bracketLevels[level] = regexp.MustCompile("(?i)\\[ ?" + level + " ?\\]")
	}
}

func guessLogLevel(logEvent *LogEvent) string {
	switch value := logEvent.Message.(type) {
	case string:
		value = stripANSI(value)
		for _, level := range logLevels {
			if plainLevels[level].MatchString(value) {
				return level
			}

			if bracketLevels[level].MatchString(value) {
				return level
			}

			if strings.Contains(value, " "+strings.ToUpper(level)+" ") {
				return level
			}
		}

		if matches := keyValueRegex.FindStringSubmatch(value); matches != nil {
			return matches[1]
		}

	case *orderedmap.OrderedMap:
		if level, ok := value.Get("level"); ok {
			if level, ok := level.(string); ok {
				return strings.ToLower(level)
			}
		}

	case map[string]interface{}:
		if level, ok := value["level"].(string); ok {
			return strings.ToLower(level)
		}

	case map[string]string:
		if level, ok := value["level"]; ok {
			return strings.ToLower(level)
		}

	default:
		log.Debugf("unknown type to guess level: %T", value)
	}

	return ""
}

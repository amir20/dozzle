package container

import (
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

var SupportedLogLevels map[string]struct{}

// Changing this also needs to change the logContext.ts file
var logLevels = [][]string{
	{"error", "err"},
	{"warn", "warning"},
	{"info", "inf"},
	{"debug", "dbg"},
	{"trace"},
	{"fatal", "sev", "severe", "crit", "critical"},
}

var plainLevels = map[string]*regexp.Regexp{}
var bracketLevels = map[string]*regexp.Regexp{}
var separatorLevels = map[string]*regexp.Regexp{}
var timestampRegex = regexp.MustCompile(`^(?:\d{4}[-/]\d{2}[-/]\d{2}(?:[T ](?:\d{2}:\d{2}:\d{2}(?:\.\d+)?Z?|\d{2}:\d{2}(?:AM|PM)))?\s+)`)

func init() {
	for _, levelGroup := range logLevels {
		first := levelGroup[0]
		levelsGroup := "(?:" + strings.Join(levelGroup, "|") + ")"
		plainLevels[first] = regexp.MustCompile("(?i)^" + levelsGroup + "[^a-z]")
		bracketLevels[first] = regexp.MustCompile("(?i)\\[ ?" + levelsGroup + " ?\\]")
		separatorLevels[first] = regexp.MustCompile("(?i) " + levelsGroup + "[/-]")
	}

	SupportedLogLevels = make(map[string]struct{}, len(logLevels)+1)
	for _, levelGroup := range logLevels {
		SupportedLogLevels[levelGroup[0]] = struct{}{}
	}
	SupportedLogLevels["unknown"] = struct{}{}
}

func guessLogLevel(logEvent *LogEvent) string {
	switch value := logEvent.Message.(type) {
	case string:
		value = stripANSI(value)
		value = timestampRegex.ReplaceAllString(value, "")
		for _, levelGroup := range logLevels {
			first := levelGroup[0]
			// Look for the level at the beginning of the message
			if plainLevels[first].MatchString(value) {
				return first
			}

			// Look for the level in brackets
			if bracketLevels[first].MatchString(value) {
				return first
			}

			// Look for the level with a separator after
			if separatorLevels[first].MatchString(value) {
				return first
			}

			// Look for the level in the middle of the message that are uppercase and surrounded by quotes
			if strings.Contains(value, "\""+strings.ToUpper(first)+"\"") {
				return first
			}

			// Look for the level in the middle of the message that are uppercase
			if strings.Contains(value, " "+strings.ToUpper(first)+" ") {
				return first
			}
		}

		return "unknown"

	case *orderedmap.OrderedMap[string, any]:
		if value == nil {
			return "unknown"
		}

		if level, ok := value.Get("level"); ok {
			if level, ok := level.(string); ok {
				return normalizeLogLevel(level)
			}
		} else if severity, ok := value.Get("severity"); ok {
			if severity, ok := severity.(string); ok {
				return normalizeLogLevel(severity)
			}
		}

	case *orderedmap.OrderedMap[string, string]:
		if value == nil {
			return "unknown"
		}
		if level, ok := value.Get("level"); ok {
			return normalizeLogLevel(level)
		} else if severity, ok := value.Get("severity"); ok {
			return normalizeLogLevel(severity)
		}

	case map[string]interface{}:
		panic("not implemented")

	case map[string]string:
		panic("not implemented")

	default:
		log.Debug().Type("type", value).Msg("unknown logEvent type")
	}

	return "unknown"
}

func normalizeLogLevel(level string) string {
	level = stripANSI(level)
	level = strings.ToLower(level)
	if _, ok := SupportedLogLevels[level]; ok {
		return level
	}

	return "unknown"
}

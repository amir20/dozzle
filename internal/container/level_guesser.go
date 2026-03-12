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
	{"warn", "warning", "wrn"},
	{"info", "inf"},
	{"debug", "dbg"},
	{"trace", "verbose", "ver", "vbs"},
	{"fatal", "sev", "severe", "crit", "critical"},
}

// aliasToCanonical maps every alias to its canonical level name.
var aliasToCanonical = map[string]string{}

type levelPatterns struct {
	plain     *regexp.Regexp // e.g. ^error[^a-z]
	bracket   *regexp.Regexp // e.g. [ error ]
	separator *regexp.Regexp // e.g. " error/"
	quoted    *regexp.Regexp // e.g. "ERROR"
	spaced    *regexp.Regexp // e.g. " ERROR "
}

var levelRegexes = map[string]levelPatterns{}

var timestampRegex = regexp.MustCompile(`^(?:\d{4}[-/]\d{2}[-/]\d{2}(?:[T ](?:\d{2}:\d{2}:\d{2}(?:\.\d+)?Z?|\d{2}:\d{2}(?:AM|PM)))?\s+)`)

// JSON keys to check for log level (in priority order).
var levelKeys = []string{"@l", "level", "severity"}

func init() {
	SupportedLogLevels = make(map[string]struct{}, len(logLevels)+1)
	for _, group := range logLevels {
		canonical := group[0]
		SupportedLogLevels[canonical] = struct{}{}
		for _, alias := range group {
			aliasToCanonical[alias] = canonical
		}

		alt := "(?:" + strings.Join(group, "|") + ")"
		upperAlt := make([]string, len(group))
		for i, l := range group {
			upperAlt[i] = strings.ToUpper(l)
		}
		upperGroup := "(?:" + strings.Join(upperAlt, "|") + ")"

		levelRegexes[canonical] = levelPatterns{
			plain:     regexp.MustCompile("(?i)^" + alt + "[^a-z]"),
			bracket:   regexp.MustCompile("(?i)\\[ ?" + alt + " ?\\]"),
			separator: regexp.MustCompile("(?i) " + alt + "[/-]"),
			quoted:    regexp.MustCompile("\"" + upperGroup + "\""),
			spaced:    regexp.MustCompile(" " + upperGroup + " "),
		}
	}
	SupportedLogLevels["unknown"] = struct{}{}
}

func guessLogLevel(logEvent *LogEvent) string {
	switch value := logEvent.Message.(type) {
	case string:
		return guessFromString(value)

	case *orderedmap.OrderedMap[string, any]:
		if value == nil {
			return "unknown"
		}
		for _, key := range levelKeys {
			if v, ok := value.Get(key); ok {
				if s, ok := v.(string); ok {
					return normalizeLogLevel(s)
				}
			}
		}

	case *orderedmap.OrderedMap[string, string]:
		if value == nil {
			return "unknown"
		}
		for _, key := range levelKeys {
			if v, ok := value.Get(key); ok {
				return normalizeLogLevel(v)
			}
		}

	case map[string]any:
		panic("not implemented")

	case map[string]string:
		panic("not implemented")

	default:
		log.Debug().Type("type", value).Msg("unknown logEvent type")
	}

	return "unknown"
}

func guessFromString(value string) string {
	value = StripANSI(value)
	value = timestampRegex.ReplaceAllString(value, "")
	for _, group := range logLevels {
		p := levelRegexes[group[0]]
		if p.plain.MatchString(value) || p.bracket.MatchString(value) || p.separator.MatchString(value) || p.quoted.MatchString(value) || p.spaced.MatchString(value) {
			return group[0]
		}
	}
	return "unknown"
}

func normalizeLogLevel(level string) string {
	level = StripANSI(level)
	level = strings.ToLower(level)
	if canonical, ok := aliasToCanonical[level]; ok {
		return canonical
	}
	if _, ok := SupportedLogLevels[level]; ok {
		return level
	}
	return "unknown"
}

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

// levelRegexes holds one combined regex per canonical level. Each regex is an
// alternation of all the shapes a level can take in a log line:
//
//	(?i:^<alt>[^a-z]      // plain prefix:  "error: ..."
//	|\[ ?<alt> ?\]        // bracketed:     "[ERROR]" / "[ error ]"
//	| <alt>[/|:-]         // separator:     " error|", " info:" (z2m)
//	|\w:<alt>\s)          // tagged:        "Zigbee2MQTT:info " (z2m)
//	|"<UPPER>"            // quoted:        "\"ERROR\""
//	|\s<UPPER>\s          // spaced:        " ERROR "
//
// The case-insensitive group covers the boundary-anchored forms; the trailing
// uppercase-only branches catch mid-line `ERROR` tokens without false-firing on
// the word "error" in prose.
var levelRegexes = map[string]*regexp.Regexp{}

// singleLetterBracket matches single-letter levels in brackets, e.g. [I], [E], [W]
var singleLetterBracket = regexp.MustCompile(`\[([EWIDFTV])\]`)

var timestampRegex = regexp.MustCompile(`^(?:\d{4}[-/]\d{2}[-/]\d{2}(?:[T ](?:\d{2}:\d{2}:\d{2}(?:\.\d+)?Z?|\d{2}:\d{2}(?:AM|PM)))?\s+)`)

// JSON keys to check for log level (in priority order).
var levelKeys = []string{"@l", "level", "log.level", "severity"}

func init() {
	SupportedLogLevels = make(map[string]struct{}, len(logLevels)+1)
	for _, group := range logLevels {
		canonical := group[0]
		SupportedLogLevels[canonical] = struct{}{}
		for _, alias := range group {
			aliasToCanonical[alias] = canonical
		}

		alt := "(?:" + strings.Join(group, "|") + ")"
		upper := strings.ToUpper(alt)

		levelRegexes[canonical] = regexp.MustCompile(
			`(?i:^` + alt + `[^a-z]` +
				`|\[ ?` + alt + ` ?\]` +
				`| ` + alt + `[/|:-]` +
				`|\w:` + alt + `\s)` +
				`|"` + upper + `"` +
				`|\s` + upper + `\s`,
		)
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

	default:
		log.Debug().Type("type", value).Msg("unknown logEvent type")
	}

	return "unknown"
}

var singleLetterToLevel = map[byte]string{
	'E': "error",
	'W': "warn",
	'I': "info",
	'D': "debug",
	'T': "trace",
	'F': "fatal",
	'V': "trace",
}

func guessFromString(value string) string {
	value = StripANSI(value)
	value = timestampRegex.ReplaceAllString(value, "")
	for _, group := range logLevels {
		if levelRegexes[group[0]].MatchString(value) {
			return group[0]
		}
	}

	if m := singleLetterBracket.FindStringSubmatch(value); m != nil {
		return singleLetterToLevel[m[1][0]]
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

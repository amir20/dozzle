package container

import (
	"regexp"
	"sort"
	"strings"

	"github.com/rs/zerolog/log"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

var SupportedLogLevels map[string]struct{}

// Changing this also needs to change the logContext.ts file
var logLevels = [][]string{
	{"error", "err"},
	{"warn", "warning", "wrn"},
	{"info", "inf", "information"},
	{"debug", "dbg"},
	{"trace", "verbose", "ver", "vbs"},
	{"fatal", "sev", "severe", "crit", "critical"},
}

// aliasToCanonical maps every alias to its canonical level name.
var aliasToCanonical = map[string]string{}

// levelMatcher extracts a canonical log level from a line. re must expose a
// single capture group holding the level alias, or a single-letter code when
// single is true (mapped via singleLetterToLevel).
type levelMatcher struct {
	re     *regexp.Regexp
	single bool
}

// levelTiers groups matchers by how confidently their shape identifies the log
// level, highest confidence first:
//
//  1. ^<level>     start-of-line prefix: "ERROR: ...", "INF ..."
//  2. [<level>]    bracketed tag / single-letter: "[ERROR]", "[E]"
//  3. <tag>:<level> structured prefix: "Zigbee2MQTT:info "
//  4. "<LEVEL>"    quoted upper-case value: LL="ERROR"
//  5. <sp><level>[/|:-] separator: " error:", " info|"
//  6. <sp><LEVEL><sp> bare upper-case token mid-line: "123 ERROR foo"
//
// guessFromString walks the tiers in order and stops at the first that matches,
// so a real level prefix at the front of the line always beats a level word
// buried in the message body. Within a tier, two different levels mean the line
// is ambiguous and we return "unknown" rather than guess. Match position and
// level severity are deliberately not used as tie-breakers.
var levelTiers [][]levelMatcher

// singleLetterBracket matches single-letter levels in brackets, e.g. [I], [E], [W]
var singleLetterBracket = regexp.MustCompile(`\[([EWIDFTV])\]`)

var timestampRegex = regexp.MustCompile(`^(?:\d{4}[-/]\d{2}[-/]\d{2}(?:[T ](?:\d{2}:\d{2}:\d{2}(?:\.\d+)?Z?|\d{2}:\d{2}(?:AM|PM)))?\s+)`)

// JSON keys to check for log level (in priority order).
var levelKeys = []string{"@l", "level", "log.level", "severity"}

func init() {
	SupportedLogLevels = make(map[string]struct{}, len(logLevels)+1)
	var aliases []string
	for _, group := range logLevels {
		canonical := group[0]
		SupportedLogLevels[canonical] = struct{}{}
		for _, alias := range group {
			aliasToCanonical[alias] = canonical
			aliases = append(aliases, alias)
		}
	}
	SupportedLogLevels["unknown"] = struct{}{}

	// Longest aliases first so e.g. "warning" is preferred over "warn".
	sort.SliceStable(aliases, func(i, j int) bool { return len(aliases[i]) > len(aliases[j]) })
	joined := strings.Join(aliases, "|")
	upper := strings.ToUpper(joined)

	levelTiers = [][]levelMatcher{
		{{re: regexp.MustCompile(`(?i)^(` + joined + `)[^a-z]`)}},
		{
			{re: regexp.MustCompile(`(?i)\[ ?(` + joined + `) ?\]`)},
			{re: singleLetterBracket, single: true},
		},
		{{re: regexp.MustCompile(`(?i):(` + joined + `)\s`)}},
		{{re: regexp.MustCompile(`"(` + upper + `)"`)}},
		{{re: regexp.MustCompile(`(?i) (` + joined + `)[/|:-]`)}},
		{{re: regexp.MustCompile(`\s(` + upper + `)\s`)}},
	}
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
	for _, tier := range levelTiers {
		level := ""
		for _, m := range tier {
			for _, match := range m.re.FindAllStringSubmatch(value, -1) {
				var canonical string
				if m.single {
					canonical = singleLetterToLevel[match[1][0]]
				} else {
					canonical = aliasToCanonical[strings.ToLower(match[1])]
				}
				if canonical == "" {
					continue
				}
				if level == "" {
					level = canonical
				} else if level != canonical {
					// Two different levels at the same confidence: ambiguous.
					return "unknown"
				}
			}
		}
		if level != "" {
			return level
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

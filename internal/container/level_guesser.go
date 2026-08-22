package container

import (
	"encoding/json"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

var SupportedLogLevels map[string]struct{}

// Changing this also needs to change the logContext.ts file
var logLevels = [][]string{
	{"error", "err"},
	{"warn", "warning", "wrn"},
	{"info", "inf", "information", "notice"},
	{"debug", "dbg"},
	{"trace", "verbose", "ver", "vbs"},
	{"fatal", "sev", "severe", "crit", "critical", "panic", "emerg", "emergency"},
}

// Aliases that are also ordinary words ("alert sent to slack") and would guess
// wrong far too often on their own. They resolve in structured logs and in the
// two shapes that leave no doubt, bracketed tags and level keys, and are kept
// out of every loose text matcher.
var ambiguousAliases = map[string]string{
	"alert": "fatal",
}

// aliasToCanonical maps every alias to its canonical level name.
var aliasToCanonical = map[string]string{}

// levelMatcher extracts a canonical log level from a line. re must expose a
// single capture group holding the level alias, or whatever convert expects
// when convert is set.
type levelMatcher struct {
	re *regexp.Regexp
	// convert turns the capture group into a canonical level, returning ""
	// when it does not resolve. When nil the capture is looked up as an alias.
	convert func(string) string
}

// levelTiers groups matchers by how confidently their shape identifies the log
// level, highest confidence first:
//
//  1. ^<level>       start-of-line prefix: "ERROR: ...", "INF ...", "E0806 14:55:55.980915 ...", "<11>Jan  1 ..."
//  2. level=<level>  explicit key/value: "level=info", "lvl=WARN", "severity: error"
//  3. [<level>]      bracketed tag / single-letter: "[ERROR]", "[E]"
//  4. <tag>:<level>  structured prefix: "Zigbee2MQTT:info "
//  5. "<LEVEL>"      quoted upper-case value: LL="ERROR"
//  6. <sp><level>[/|:-] separator: " error:", " info|"
//  7. <sp><LEVEL><sp> bare upper-case token mid-line: "123 ERROR foo"
//
// guessFromString walks the tiers in order and stops at the first that matches,
// so a real level prefix at the front of the line always beats a level word
// buried in the message body. Within a tier, two different levels mean the line
// is ambiguous and we return "unknown" rather than guess. Match position and
// level severity are deliberately not used as tie-breakers.
var levelTiers [][]levelMatcher

// singleLetterBracket matches single-letter levels in brackets, e.g. [I], [E], [W]
var singleLetterBracket = regexp.MustCompile(`\[([EWIDFTV])\]`)

// klogPrefix matches the Kubernetes klog/glog header, where the single-letter
// level is glued straight onto the timestamp with no separator, e.g.
//
//	E0806 14:55:55.980915       1 fsHandler.go:121] failed to collect ...
//
// Used by the whole Kubernetes toolchain (kubelet, kube-*, cAdvisor, etcd
// tooling). The full "Lmmdd hh:mm:ss.uuuuuu" shape is required so ordinary
// prose starting with a capital letter cannot match.
var klogPrefix = regexp.MustCompile(`^([EWIDFTV])\d{4} \d{2}:\d{2}:\d{2}\.\d{6}`)

// syslogPriority matches the RFC 3164 / RFC 5424 priority value that starts a
// syslog line, e.g. "<11>Jan  1 00:00:00 host app: disk failure". systemd,
// rsyslog and anything piped through systemd-cat emit it. The value is
// facility*8 + severity, so the severity is the remainder of 8.
var syslogPriority = regexp.MustCompile(`^<(\d{1,3})>`)

var timestampRegex = regexp.MustCompile(`^(?:\d{4}[-/]\d{2}[-/]\d{2}(?:[T ](?:\d{2}:\d{2}:\d{2}(?:\.\d+)?Z?|\d{2}:\d{2}(?:AM|PM)))?\s+)`)

// Keys that carry a log level in structured logs, in priority order. Keys are
// matched case-insensitively and ignoring ".", "_" and "-", so "level",
// "Level", "log.level", "log_level" and "logLevel" all resolve to the same
// entry. Covers Serilog (@l), Python's json logger (levelname), OpenTelemetry
// and GCP (severity, severityText) and the logfmt loggers (lvl).
var levelKeys = []string{"@l", "level", "levelname", "loglevel", "lvl", "severity", "severitytext"}

var levelKeyRank = map[string]int{}

var keyNormalizer = strings.NewReplacer(".", "", "_", "", "-", "")

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

	for i, key := range levelKeys {
		levelKeyRank[key] = i
	}

	unambiguous := make([]string, len(aliases))
	copy(unambiguous, aliases)
	for alias, canonical := range ambiguousAliases {
		aliasToCanonical[alias] = canonical
		aliases = append(aliases, alias)
	}

	// Longest aliases first so e.g. "warning" is preferred over "warn".
	byLength := func(list []string) {
		sort.SliceStable(list, func(i, j int) bool { return len(list[i]) > len(list[j]) })
	}
	byLength(aliases)
	byLength(unambiguous)
	joined := strings.Join(unambiguous, "|")
	// Shapes that cannot be confused with prose also accept the ambiguous aliases.
	joinedAll := strings.Join(aliases, "|")
	upper := strings.ToUpper(joined)

	levelTiers = [][]levelMatcher{
		{
			{re: regexp.MustCompile(`(?i)^(` + joined + `)[^a-z]`)},
			{re: klogPrefix, convert: levelFromSingleLetter},
			{re: syslogPriority, convert: levelFromSyslogPriority},
		},
		{{re: regexp.MustCompile(`(?i)(?:^|[\s,{])"?(?:log[._-]?level|levelname|level|lvl|severity)"?\s*[=:]\s*"?(` + joinedAll + `)(?:[^a-z]|$)`)}},
		{
			{re: regexp.MustCompile(`(?i)\[ ?(` + joinedAll + `) ?\]`)},
			{re: singleLetterBracket, convert: levelFromSingleLetter},
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
		return guessFromMap(value)

	case *orderedmap.OrderedMap[string, string]:
		if value == nil {
			return "unknown"
		}
		return guessFromMap(value)

	default:
		log.Debug().Type("type", value).Msg("unknown logEvent type")
	}

	return "unknown"
}

// guessFromMap picks the level out of a structured entry. Every level key is
// collected in one pass and then resolved in priority order, so an unusable
// value on a high priority key (a number where a name was expected, an object)
// falls through to the next key instead of losing the level.
func guessFromMap[V any](value *orderedmap.OrderedMap[string, V]) string {
	candidates := make([]any, len(levelKeys))
	found := false
	for pair := value.Oldest(); pair != nil; pair = pair.Next() {
		rank, ok := levelKeyRank[normalizeKey(pair.Key)]
		if !ok || candidates[rank] != nil {
			continue
		}
		candidates[rank] = pair.Value
		found = true
	}
	if !found {
		return "unknown"
	}
	for i, candidate := range candidates {
		if candidate == nil {
			continue
		}
		if level := resolveLevelValue(candidate, numericLevelKeys[levelKeys[i]]); level != "unknown" {
			return level
		}
	}
	return "unknown"
}

// Keys where a number means a bunyan/pino level. "severity" is left out: a
// number there is a syslog or OpenTelemetry severity, which counts the other
// way around.
var numericLevelKeys = map[string]bool{"@l": true, "level": true, "loglevel": true, "lvl": true}

// normalizeKey lowercases a key and drops the separators loggers sprinkle
// through it, so "log.level", "log_level" and "logLevel" all collapse to
// "loglevel".
func normalizeKey(key string) string {
	return keyNormalizer.Replace(strings.ToLower(key))
}

// resolveLevelValue turns the value of a level key into a canonical level.
// Numbers are only read when numeric says so, because bunyan and pino, the two
// most common Node loggers, write levels as numbers.
func resolveLevelValue(value any, numeric bool) string {
	number := func(n float64) string {
		if !numeric {
			return "unknown"
		}
		return levelFromNumber(n)
	}
	switch v := value.(type) {
	case string:
		if level := normalizeLogLevel(v); level != "unknown" {
			return level
		}
		if n, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
			return number(n)
		}
	case float64:
		return number(v)
	case float32:
		return number(float64(v))
	case int:
		return number(float64(v))
	case int64:
		return number(float64(v))
	case json.Number:
		if n, err := v.Float64(); err == nil {
			return number(n)
		}
	}
	return "unknown"
}

// levelFromNumber maps the numeric levels used by bunyan and pino (10 trace,
// 20 debug, 30 info, 40 warn, 50 error, 60 fatal) to canonical names. Custom
// levels in between fall into the band below them, which is how those loggers
// render them too. Values below 10 are left unknown on purpose: there the
// bunyan scale collides with syslog severities (0-7, and inverted) and with
// OpenTelemetry severity numbers, so a guess would be wrong as often as right.
func levelFromNumber(n float64) string {
	switch {
	case n >= 60:
		return "fatal"
	case n >= 50:
		return "error"
	case n >= 40:
		return "warn"
	case n >= 30:
		return "info"
	case n >= 20:
		return "debug"
	case n >= 10:
		return "trace"
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

func levelFromSingleLetter(match string) string {
	return singleLetterToLevel[match[0]]
}

// syslogSeverityToLevel maps the eight RFC 5424 severities, in order, onto the
// levels Dozzle knows about.
var syslogSeverityToLevel = [8]string{"fatal", "fatal", "fatal", "error", "warn", "info", "info", "debug"}

func levelFromSyslogPriority(match string) string {
	pri, err := strconv.Atoi(match)
	if err != nil || pri > 191 {
		return ""
	}
	return syslogSeverityToLevel[pri%8]
}

func guessFromString(value string) string {
	value = StripANSI(value)
	value = timestampRegex.ReplaceAllString(value, "")
	for _, tier := range levelTiers {
		level := ""
		for _, m := range tier {
			for _, match := range m.re.FindAllStringSubmatch(value, -1) {
				var canonical string
				if m.convert != nil {
					canonical = m.convert(match[1])
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

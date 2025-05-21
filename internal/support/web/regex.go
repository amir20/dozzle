package support_web

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/amir20/dozzle/internal/container"
	"github.com/rs/zerolog/log"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

// PatternMatcher defines the interface for a regex pattern matcher
type PatternMatcher struct {
	Regex       *regexp.Regexp
	MarkerStart string
	MarkerEnd   string
}

// NewPatternMatcher creates a new pattern matcher with the specified regex and markers
func NewPatternMatcher(re *regexp.Regexp, markerStart, markerEnd string) *PatternMatcher {
	return &PatternMatcher{
		Regex:       re,
		MarkerStart: markerStart,
		MarkerEnd:   markerEnd,
	}
}

// CreateRegex compiles a regex pattern with optional case-insensitive flag
func CreateRegex(pattern string, caseInsensitive bool) (*regexp.Regexp, error) {
	flags := ""
	if caseInsensitive || pattern == strings.ToLower(pattern) {
		flags = "(?i)"
	}

	re, err := regexp.Compile(flags + pattern)
	if err != nil {
		log.Debug().Err(err).Str("pattern", pattern).Msg("failed to compile regex")
		return nil, err
	}

	return re, nil
}

// MarkInLogEvent applies the pattern matcher to the log event's message
func (pm *PatternMatcher) MarkInLogEvent(logEvent *container.LogEvent) bool {
	switch value := logEvent.Message.(type) {
	case string:
		if pm.Regex.MatchString(value) {
			logEvent.Message = pm.Regex.ReplaceAllString(value, pm.MarkerStart+"$0"+pm.MarkerEnd)
			return true
		}

	case *orderedmap.OrderedMap[string, any]:
		return pm.markMapAny(value)

	case *orderedmap.OrderedMap[string, string]:
		return pm.markMapString(value)

	case map[string]interface{}:
		return pm.markMap(value)

	case map[string]string:
		panic("not implemented")

	default:
		log.Debug().Type("type", value).Msg("unknown logEvent type")
	}

	return false
}

func (pm *PatternMatcher) markMapAny(orderedMap *orderedmap.OrderedMap[string, any]) bool {
	found := false
	for pair := orderedMap.Oldest(); pair != nil; pair = pair.Next() {
		switch value := pair.Value.(type) {
		case string:
			if replaced, matched := pm.markString(value); matched {
				found = true
				orderedMap.Set(pair.Key, replaced)
			}

		case []any:
			if pm.markArray(value) {
				found = true
			}

		case *orderedmap.OrderedMap[string, any]:
			if pm.markMapAny(value) {
				found = true
			}

		case *orderedmap.OrderedMap[string, string]:
			if pm.markMapString(value) {
				found = true
			}

		case map[string]interface{}:
			if pm.markMap(value) {
				found = true
			}

		case int, float64, bool:
			formatted := fmt.Sprintf("%v", value)
			if pm.Regex.MatchString(formatted) {
				orderedMap.Set(pair.Key, pm.Regex.ReplaceAllString(formatted, pm.MarkerStart+"$0"+pm.MarkerEnd))
				found = true
			}

		default:
			log.Debug().Type("type", value).Msg("unknown logEvent type inside markMapAny")
		}
	}

	return found
}

func (pm *PatternMatcher) markMap(data map[string]interface{}) bool {
	found := false
	for key, value := range data {
		switch value := value.(type) {
		case string:
			if replaced, matched := pm.markString(value); matched {
				found = true
				data[key] = replaced
			}
		case []any:
			if pm.markArray(value) {
				found = true
			}

		case map[string]interface{}:
			if pm.markMap(value) {
				found = true
			}

		case int, float64, bool:
			formatted := fmt.Sprintf("%v", value)
			if pm.Regex.MatchString(formatted) {
				data[key] = pm.Regex.ReplaceAllString(formatted, pm.MarkerStart+"$0"+pm.MarkerEnd)
				found = true
			}
		default:
			log.Debug().Type("type", value).Msg("unknown logEvent type inside markMap")
		}
	}

	return found
}

func (pm *PatternMatcher) markMapString(orderedMap *orderedmap.OrderedMap[string, string]) bool {
	found := false
	for pair := orderedMap.Oldest(); pair != nil; pair = pair.Next() {
		if replaced, matched := pm.markString(pair.Value); matched {
			found = true
			orderedMap.Set(pair.Key, replaced)
		}
	}
	return found
}

func (pm *PatternMatcher) markArray(data []any) bool {
	found := false
	for i, value := range data {
		switch value := value.(type) {
		case string:
			if replaced, matched := pm.markString(value); matched {
				found = true
				data[i] = replaced
			}
		case int, float64, bool:
			formatted := fmt.Sprintf("%v", value)
			if pm.Regex.MatchString(formatted) {
				data[i] = pm.Regex.ReplaceAllString(formatted, pm.MarkerStart+"$0"+pm.MarkerEnd)
				found = true
			}
		case []any:
			if pm.markArray(value) {
				found = true
			}
		case map[string]interface{}:
			if pm.markMap(value) {
				found = true
			}
		}
	}

	return found
}

func (pm *PatternMatcher) markString(value string) (string, bool) {
	if pm.Regex.MatchString(value) {
		replaced := pm.Regex.ReplaceAllString(value, pm.MarkerStart+"$0"+pm.MarkerEnd)
		return replaced, true
	}

	return value, false
}

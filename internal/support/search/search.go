package search

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/amir20/dozzle/internal/container"
	"github.com/rs/zerolog/log"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

const (
	MarkerStart = "\uE000"
	MarkerEnd   = "\uE001"
)

func ParseRegex(search string) (*regexp.Regexp, error) {
	flags := ""

	if search == strings.ToLower(search) {
		flags = "(?i)"
	}
	re, err := regexp.Compile(flags + search)

	if err != nil {
		log.Debug().Err(err).Str("search", search).Msg("failed to compile regex")
		return nil, err
	}

	return re, nil
}

func Search(re *regexp.Regexp, logEvent *container.LogEvent) bool {
	switch value := logEvent.Message.(type) {
	case string:
		if re.MatchString(value) {
			logEvent.Message = re.ReplaceAllString(value, MarkerStart+"$0"+MarkerEnd)
			return true
		}

	case *orderedmap.OrderedMap[string, any]:
		return searchMapAny(re, value)

	case *orderedmap.OrderedMap[string, string]:
		return searchMapString(re, value)

	case map[string]interface{}:
		panic("not implemented")
	case map[string]string:
		panic("not implemented")
	default:
		log.Debug().Type("type", value).Msg("unknown logEvent type")
	}

	return false
}

func searchMapAny(re *regexp.Regexp, orderedMap *orderedmap.OrderedMap[string, any]) bool {
	found := false
	for pair := orderedMap.Oldest(); pair != nil; pair = pair.Next() {
		switch value := pair.Value.(type) {
		case string:
			if replaced, matched := searchString(re, value); matched {
				found = true
				orderedMap.Set(pair.Key, replaced)
			}

		case []any:
			if searchArray(re, value) {
				found = true
			}

		case *orderedmap.OrderedMap[string, any]:
			if searchMapAny(re, value) {
				found = true
			}

		case *orderedmap.OrderedMap[string, string]:
			if searchMapString(re, value) {
				found = true
			}

		case map[string]interface{}:
			if searchMap(re, value) {
				found = true
			}

		case int, float64, bool:
			formatted := fmt.Sprintf("%v", value)
			if re.MatchString(formatted) {
				orderedMap.Set(pair.Key, re.ReplaceAllString(formatted, MarkerStart+"$0"+MarkerEnd))
				found = true
			}

		default:
			log.Debug().Type("type", value).Msg("unknown logEvent type inside searchMapAny")
		}
	}

	return found
}

func searchMap(re *regexp.Regexp, data map[string]interface{}) bool {
	found := false
	for key, value := range data {
		switch value := value.(type) {
		case string:
			if replaced, matched := searchString(re, value); matched {
				found = true
				data[key] = replaced
			}
		case []any:
			if searchArray(re, value) {
				found = true
			}

		case map[string]interface{}:
			if searchMap(re, value) {
				found = true
			}

		case int, float64, bool:
			formatted := fmt.Sprintf("%v", value)
			if re.MatchString(formatted) {
				data[key] = re.ReplaceAllString(formatted, MarkerStart+"$0"+MarkerEnd)
				found = true
			}
		default:
			log.Debug().Type("type", value).Msg("unknown logEvent type inside searchMap")
		}
	}

	return found
}

func searchMapString(re *regexp.Regexp, orderedMap *orderedmap.OrderedMap[string, string]) bool {
	found := false
	for pair := orderedMap.Oldest(); pair != nil; pair = pair.Next() {
		if replaced, matched := searchString(re, pair.Value); matched {
			found = true
			orderedMap.Set(pair.Key, replaced)
		}
	}
	return found
}

func searchArray(re *regexp.Regexp, data []any) bool {
	found := false
	for i, value := range data {
		switch value := value.(type) {
		case string:
			if replaced, matched := searchString(re, value); matched {
				found = true
				data[i] = replaced
			}
		case int, float64, bool:
			formatted := fmt.Sprintf("%v", value)
			if re.MatchString(formatted) {
				data[i] = re.ReplaceAllString(formatted, MarkerStart+"$0"+MarkerEnd)
				found = true
			}
		case []any:
			if searchArray(re, value) {
				found = true
			}
		case map[string]interface{}:
			if searchMap(re, value) {
				found = true
			}
		}
	}

	return found
}

func searchString(re *regexp.Regexp, value string) (string, bool) {
	if re.MatchString(value) {
		replaced := re.ReplaceAllString(value, MarkerStart+"$0"+MarkerEnd)
		return replaced, true
	}

	return value, false
}

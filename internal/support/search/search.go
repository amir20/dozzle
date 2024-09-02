package search

import (
	"regexp"
	"strings"

	"github.com/amir20/dozzle/internal/docker"
	"github.com/rs/zerolog/log"
	orderedmap "github.com/wk8/go-ordered-map/v2"
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

func Search(re *regexp.Regexp, logEvent *docker.LogEvent) bool {
	switch value := logEvent.Message.(type) {
	case string:
		if re.MatchString(value) {
			logEvent.Message = re.ReplaceAllString(value, "<mark>$0</mark>")
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
			if re.MatchString(value) {
				found = true
				orderedMap.Set(pair.Key, re.ReplaceAllString(value, "<mark>$0</mark>"))
			}

		case []any:
			for i, v := range value {
				switch v := v.(type) {
				case string:
					if re.MatchString(v) {
						found = true
						value[i] = re.ReplaceAllString(v, "<mark>$0</mark>")
					}
				}
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
			if re.MatchString(value) {
				data[key] = re.ReplaceAllString(value, "<mark>$0</mark>")
				found = true
			}

		case []any:
			for i, v := range value {
				switch v := v.(type) {
				case string:
					if re.MatchString(v) {
						found = true
						value[i] = re.ReplaceAllString(v, "<mark>$0</mark>")
					}
				}
			}

		case map[string]interface{}:
			if searchMap(re, value) {
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
		if re.MatchString(pair.Value) {
			orderedMap.Set(pair.Key, re.ReplaceAllString(pair.Value, "<mark>$0</mark>"))
			found = true
		}
	}
	return found
}

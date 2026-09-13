package support_web

import (
	"html"
	"regexp"
	"strings"

	"github.com/amir20/dozzle/internal/container"

	"github.com/rs/zerolog/log"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

// URL marker regex compiled once for performance
var urlMarkerRegex = regexp.MustCompile(URLMarkerStart + "(.*?)" + URLMarkerEnd)
var searchMarkerStripper = strings.NewReplacer(MarkerStart, "", MarkerEnd, "")
var searchMarkerHTMLReplacer = strings.NewReplacer(MarkerStart, "<mark>", MarkerEnd, "</mark>")
var urlMarkerStripper = strings.NewReplacer(URLMarkerStart, "", URLMarkerEnd, "")

func EscapeHTMLValues(logEvent *container.LogEvent) {
	switch value := logEvent.Message.(type) {
	case string:
		logEvent.Message = escapeAndProcessMarkers(value)

	case []container.LogFragment:
		for i, fragment := range value {
			value[i].Message = escapeAndProcessMarkers(fragment.Message)
		}

	case *orderedmap.OrderedMap[string, any]:
		escapeAnyMap(value)

	case *orderedmap.OrderedMap[string, string]:
		escapeStringMap(value)

	case map[string]any:
		panic("not implemented")

	case map[string]string:
		panic("not implemented")

	default:
		log.Trace().Type("type", value).Msg("unknown logEvent type")
	}
}

// escapeAndProcessMarkers marks URLs itself rather than trusting markers already in
// the string: they are plain characters any container can log, so a forged pair
// around javascript: would otherwise become a live href.
func escapeAndProcessMarkers(value string) string {
	value = urlMarkerStripper.Replace(value)
	value = urlRegex.ReplaceAllString(value, URLMarkerStart+"$0"+URLMarkerEnd)
	value = html.EscapeString(value)
	value = urlMarkerRegex.ReplaceAllStringFunc(value, func(match string) string {
		url := strings.TrimSuffix(strings.TrimPrefix(match, URLMarkerStart), URLMarkerEnd)
		href := searchMarkerStripper.Replace(url)
		text := searchMarkerHTMLReplacer.Replace(url)
		if !strings.HasPrefix(href, "http://") && !strings.HasPrefix(href, "https://") {
			return text
		}
		return "<a href=\"" + href + "\" target=\"_blank\" rel=\"noopener noreferrer external\">" + text + "</a>"
	})
	value = searchMarkerHTMLReplacer.Replace(value)
	return value
}

func escapeAnyMap(orderedMap *orderedmap.OrderedMap[string, any]) {
	for pair := orderedMap.Oldest(); pair != nil; pair = pair.Next() {
		switch value := pair.Value.(type) {
		case string:
			orderedMap.Set(pair.Key, escapeAndProcessMarkers(value))
		case *orderedmap.OrderedMap[string, any]:
			escapeAnyMap(value)
		case *orderedmap.OrderedMap[string, string]:
			escapeStringMap(value)
		case map[string]any:
			escapeMapStringInterface(value)
		case map[string]string:
			escapeStringMapString(value)
		case []any:
			escapeSlice(value)
			orderedMap.Set(pair.Key, value)
		default:
			log.Trace().Type("type", value).Msg("unknown logEvent type")
		}
	}
}

func escapeStringMap(orderedMap *orderedmap.OrderedMap[string, string]) {
	for pair := orderedMap.Oldest(); pair != nil; pair = pair.Next() {
		orderedMap.Set(pair.Key, escapeAndProcessMarkers(pair.Value))
	}
}

func escapeMapStringInterface(value map[string]any) {
	for key, val := range value {
		switch val := val.(type) {
		case string:
			value[key] = escapeAndProcessMarkers(val)
		case map[string]any:
			escapeMapStringInterface(val)
		case map[string]string:
			escapeStringMapString(val)
		case []any:
			escapeSlice(val)
		}
	}
}

func escapeStringMapString(value map[string]string) {
	for key, val := range value {
		value[key] = escapeAndProcessMarkers(val)
	}
}

func escapeSlice(slice []any) {
	for i, val := range slice {
		switch val := val.(type) {
		case string:
			slice[i] = escapeAndProcessMarkers(val)
		case *orderedmap.OrderedMap[string, any]:
			escapeAnyMap(val)
		case *orderedmap.OrderedMap[string, string]:
			escapeStringMap(val)
		case map[string]any:
			escapeMapStringInterface(val)
		case map[string]string:
			escapeStringMapString(val)
		case []any:
			escapeSlice(val)
		}
	}
}

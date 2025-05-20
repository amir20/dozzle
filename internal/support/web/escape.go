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

func EscapeHTMLValues(logEvent *container.LogEvent) {
	// Mark URLs before HTML escaping
	MarkURLs(logEvent)

	switch value := logEvent.Message.(type) {
	case string:
		logEvent.Message = escapeAndProcessMarkers(value)

	case *orderedmap.OrderedMap[string, any]:
		escapeAnyMap(value)

	case *orderedmap.OrderedMap[string, string]:
		escapeStringMap(value)

	case map[string]interface{}:
		panic("not implemented")

	case map[string]string:
		panic("not implemented")

	default:
		log.Debug().Type("type", value).Msg("unknown logEvent type")
	}
}

func escapeAndProcessMarkers(value string) string {
	value = html.EscapeString(value)
	value = strings.ReplaceAll(value, MarkerStart, "<mark>")
	value = strings.ReplaceAll(value, MarkerEnd, "</mark>")
	// Process URL markers
	value = urlMarkerRegex.ReplaceAllString(value, "<a href=\"$1\" target=\"_blank\" rel=\"noopener noreferrer external\">$1</a>")
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
		}
	}

}

func escapeStringMap(orderedMap *orderedmap.OrderedMap[string, string]) {
	for pair := orderedMap.Oldest(); pair != nil; pair = pair.Next() {
		orderedMap.Set(pair.Key, escapeAndProcessMarkers(pair.Value))
	}
}

package support_web

import (
	"html"
	"strings"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/support/search"
	"github.com/rs/zerolog/log"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

func EscapeHTMLValues(logEvent *container.LogEvent) {
	switch value := logEvent.Message.(type) {
	case string:
		value = html.EscapeString(value)
		value = strings.ReplaceAll(value, search.MarkerStart, "<mark>")
		logEvent.Message = strings.ReplaceAll(value, search.MarkerEnd, "</mark>")

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

func escapeAnyMap(orderedMap *orderedmap.OrderedMap[string, any]) {
	for pair := orderedMap.Oldest(); pair != nil; pair = pair.Next() {
		switch value := pair.Value.(type) {
		case string:
			value = html.EscapeString(value)
			value = strings.ReplaceAll(value, search.MarkerStart, "<mark>")
			value = strings.ReplaceAll(value, search.MarkerEnd, "</mark>")
			orderedMap.Set(pair.Key, value)
		case *orderedmap.OrderedMap[string, any]:
			escapeAnyMap(value)
		case *orderedmap.OrderedMap[string, string]:
			escapeStringMap(value)
		}
	}

}

func escapeStringMap(orderedMap *orderedmap.OrderedMap[string, string]) {
	for pair := orderedMap.Oldest(); pair != nil; pair = pair.Next() {
		value := html.EscapeString(pair.Value)
		value = strings.ReplaceAll(value, search.MarkerStart, "<mark>")
		value = strings.ReplaceAll(value, search.MarkerEnd, "</mark>")
		orderedMap.Set(pair.Key, value)
	}
}

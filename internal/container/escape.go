package container

import (
	"html"

	"github.com/rs/zerolog/log"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

func escape(logEvent *LogEvent) {
	switch value := logEvent.Message.(type) {
	case string:
		logEvent.Message = html.EscapeString(value)

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
			orderedMap.Set(pair.Key, html.EscapeString(value))
		case *orderedmap.OrderedMap[string, any]:
			escapeAnyMap(value)
		case *orderedmap.OrderedMap[string, string]:
			escapeStringMap(value)
		}
	}

}

func escapeStringMap(orderedMap *orderedmap.OrderedMap[string, string]) {
	for pair := orderedMap.Oldest(); pair != nil; pair = pair.Next() {
		orderedMap.Set(pair.Key, html.EscapeString(pair.Value))
	}
}

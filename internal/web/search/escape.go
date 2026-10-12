package search

import (
	"html"
	"regexp"
	"strings"

	"github.com/amir20/dozzle/internal/container"

	"github.com/rs/zerolog/log"
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

	case container.Fields[any]:
		escapeAnyMap(value)

	case container.Fields[string]:
		escapeStringMap(value)

	case map[string]any:
		escapeMapStringInterface(value)

	case map[string]string:
		escapeStringMapString(value)

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

func escapeAnyMap(fields container.Fields[any]) {
	for i, pair := range fields {
		switch value := pair.Value.(type) {
		case string:
			fields[i].Value = escapeAndProcessMarkers(value)
		case container.Fields[any]:
			escapeAnyMap(value)
		case container.Fields[string]:
			escapeStringMap(value)
		case map[string]any:
			escapeMapStringInterface(value)
		case map[string]string:
			escapeStringMapString(value)
		case []any:
			escapeSlice(value)
		default:
			log.Trace().Type("type", value).Msg("unknown logEvent type")
		}
	}
}

func escapeStringMap(fields container.Fields[string]) {
	for i, pair := range fields {
		fields[i].Value = escapeAndProcessMarkers(pair.Value)
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
		case container.Fields[any]:
			escapeAnyMap(val)
		case container.Fields[string]:
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

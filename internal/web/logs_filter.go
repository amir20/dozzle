package web

import (
	"net/http"
	"regexp"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/container/logparse"
	"github.com/amir20/dozzle/internal/web/search"
)

func parseStdTypes(r *http.Request) container.StdType {
	var stdTypes container.StdType
	if r.URL.Query().Has("stdout") {
		stdTypes |= container.STDOUT
	}
	if r.URL.Query().Has("stderr") {
		stdTypes |= container.STDERR
	}
	return stdTypes
}

// logFilter is the search the log viewer sends as `filter`, `inverse` and `levels`.
type logFilter struct {
	regex   *regexp.Regexp
	levels  map[string]struct{}
	inverse bool
}

func parseLogFilter(r *http.Request) (logFilter, error) {
	q := r.URL.Query()
	f := logFilter{
		levels:  make(map[string]struct{}),
		inverse: q.Get("inverse") == "true",
	}
	for _, level := range q["levels"] {
		f.levels[level] = struct{}{}
	}
	if q.Has("filter") {
		regex, err := search.ParseRegex(q.Get("filter"))
		if err != nil {
			return logFilter{}, err
		}
		f.regex = regex
	}
	return f, nil
}

func (f logFilter) matchesText(event *container.LogEvent) bool {
	return f.regex == nil || f.inverse != search.Search(f.regex, event)
}

func (f logFilter) matchesLevel(event *container.LogEvent) bool {
	_, ok := f.levels[event.Level]
	return ok
}

func (f logFilter) matches(event *container.LogEvent) bool {
	return f.matchesText(event) && f.matchesLevel(event)
}

// narrowing reports whether the filter hides anything, in which case the live
// tail alone would leave the view near empty and the stream backfills matches.
func (f logFilter) narrowing() bool {
	if f.regex != nil || f.inverse {
		return true
	}
	for level := range logparse.SupportedLogLevels {
		if _, ok := f.levels[level]; !ok {
			return true
		}
	}
	return false
}

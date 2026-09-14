package search

import (
	"strings"
	"testing"

	"github.com/amir20/dozzle/internal/container"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

func TestEscapeHTMLValuesIgnoresForgedURLMarkers(t *testing.T) {
	forged := URLMarkerStart + "javascript:alert(document.domain)" + URLMarkerEnd
	json := orderedmap.New[string, any]()
	json.Set("msg", forged)
	json.Set("nested", []any{forged})

	tests := []struct {
		name    string
		message any
	}{
		{name: "string", message: forged},
		{name: "fragments", message: []container.LogFragment{{Message: forged}}},
		{name: "json", message: json},
		{name: "wrapping a real url", message: URLMarkerStart + "javascript:x//https://example.com" + URLMarkerEnd},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := &container.LogEvent{Type: container.LogTypeSingle, Message: tt.message}
			EscapeHTMLValues(event)

			var out []string
			switch m := event.Message.(type) {
			case string:
				out = append(out, m)
			case []container.LogFragment:
				out = append(out, m[0].Message)
			case *orderedmap.OrderedMap[string, any]:
				out = append(out, m.Value("msg").(string), m.Value("nested").([]any)[0].(string))
			}
			for _, got := range out {
				if strings.Contains(got, `href="javascript:`) {
					t.Fatalf("forged marker produced a javascript href: %q", got)
				}
				if strings.ContainsAny(got, URLMarkerStart+URLMarkerEnd) {
					t.Fatalf("url markers leaked to output: %q", got)
				}
			}
		})
	}
}

func TestEscapeHTMLValuesKeepsSearchedURLClickable(t *testing.T) {
	tests := []struct {
		name   string
		url    string
		search string
		want   string
	}{
		{
			name:   "https path",
			url:    "https://example.com/static/uploads/proofs/image.webp",
			search: "/proofs",
			want:   "https://example.com/static/uploads<mark>/proofs</mark>/image.webp",
		},
		{
			name:   "https segment",
			url:    "https://example.com/static/uploads/proofs/image.webp",
			search: "uploads",
			want:   "https://example.com/static/<mark>uploads</mark>/proofs/image.webp",
		},
		{
			name:   "http path",
			url:    "http://example.com/static/uploads/proofs/image.webp",
			search: "/proofs",
			want:   "http://example.com/static/uploads<mark>/proofs</mark>/image.webp",
		},
		{
			name:   "localhost path",
			url:    "http://localhost:3000/static/uploads/proofs/image.webp",
			search: "/proofs",
			want:   "http://localhost:3000/static/uploads<mark>/proofs</mark>/image.webp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := &container.LogEvent{Type: container.LogTypeSingle, Message: tt.url}
			regex, err := ParseRegex(tt.search)
			if err != nil {
				t.Fatal(err)
			}
			if !Search(regex, event) {
				t.Fatal("expected search to match URL")
			}

			EscapeHTMLValues(event)

			got, ok := event.Message.(string)
			if !ok {
				t.Fatalf("expected message to be string, got %T", event.Message)
			}
			if !strings.Contains(got, `href="`+tt.url+`"`) {
				t.Fatalf("expected full URL href, got %q", got)
			}
			if !strings.Contains(got, ">"+tt.want+"</a>") {
				t.Fatalf("expected highlighted URL text, got %q", got)
			}
		})
	}
}

func TestSearchKeepsTimestampPrefixAligned(t *testing.T) {
	const line = "2026-09-13T22:28:56Z INF <b>ready</b>"
	const tp = 21

	tests := []struct {
		name   string
		search string
		wantTp int
		rest   string
	}{
		{name: "match after the timestamp", search: "ready", wantTp: tp, rest: "INF &lt;b&gt;<mark>ready</mark>&lt;/b&gt;"},
		{name: "match inside the timestamp", search: "22:28", wantTp: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			regex, err := ParseRegex(tt.search)
			if err != nil {
				t.Fatal(err)
			}

			single := &container.LogEvent{Type: container.LogTypeSingle, Message: line, TimestampPrefix: tp}
			group := &container.LogEvent{Type: container.LogTypeGroup, Message: []container.LogFragment{{Message: line, TimestampPrefix: tp}}}
			for _, event := range []*container.LogEvent{single, group} {
				if !Search(regex, event) {
					t.Fatal("expected search to match")
				}
				EscapeHTMLValues(event)
			}

			fragment := group.Message.([]container.LogFragment)[0]
			if single.TimestampPrefix != tt.wantTp || fragment.TimestampPrefix != tt.wantTp {
				t.Fatalf("timestamp prefix = %d / %d, want %d", single.TimestampPrefix, fragment.TimestampPrefix, tt.wantTp)
			}
			if tt.rest != "" {
				if got := single.Message.(string)[single.TimestampPrefix:]; got != tt.rest {
					t.Fatalf("single rest = %q, want %q", got, tt.rest)
				}
				if got := fragment.Message[fragment.TimestampPrefix:]; got != tt.rest {
					t.Fatalf("fragment rest = %q, want %q", got, tt.rest)
				}
			}
		})
	}
}

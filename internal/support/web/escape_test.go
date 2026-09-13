package support_web

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

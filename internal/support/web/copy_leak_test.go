package support_web

import (
	"strings"
	"testing"

	"github.com/amir20/dozzle/internal/container"
)

// A regex filter marks matches with private-use markers as a side effect of
// Search. When a grouped (multi-line) log is then copied as plain text, those
// markers must not leak into the output.
func TestSearchedGroupedLogCopyHasNoMarkers(t *testing.T) {
	event := &container.LogEvent{
		Type: container.LogTypeGroup,
		Message: []container.LogFragment{
			{Message: "Exception in thread main"},
			{Message: "  at com.example.Main(Main.java:10)"},
			{Message: "  at com.example.Foo(Foo.java:20)"},
		},
	}

	regex, err := ParseRegex("example")
	if err != nil {
		t.Fatal(err)
	}

	// This mirrors the copy-logs path: filter via Search (mutating), then render.
	if !Search(regex, event) {
		t.Fatal("expected search to match the grouped log")
	}

	out := event.PlainText()
	if strings.ContainsAny(out, MarkerStart+MarkerEnd+URLMarkerStart+URLMarkerEnd) {
		t.Fatalf("copied text leaked highlight markers: %q", out)
	}
	if !strings.Contains(out, "at com.example.Main(Main.java:10)") {
		t.Fatalf("expected grouped lines preserved, got %q", out)
	}
}

package releases

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// A dev build is not a point on the release line, so no release is newer than
// it and nothing may be marked latest.
func TestOnReleaseLine(t *testing.T) {
	for _, version := range []string{"v11.1.0", "11.1.0", "v11.1.0-beta.1"} {
		assert.True(t, OnReleaseLine(version), version)
	}
	for _, version := range []string{"main-1d901bf", "pr-5216-abc1234", "local", "head", "", "v11.1"} {
		assert.False(t, OnReleaseLine(version), version)
	}
}

func TestParseBodyCounts(t *testing.T) {
	type counts struct{ features, bugFixes, breaking int }
	tests := []struct {
		name string
		body string
		want counts
	}{
		{"real v11.3.1 release, nested scopes", readTestdata(t, "v11.3.1.md"), counts{4, 27, 0}},
		{"real v10.0.1 release with breaking changes", readTestdata(t, "v10.0.1.md"), counts{1, 2, 1}},
		{"empty body", "", counts{0, 0, 0}},
		{"plain sections", "### Features\n\n- a\n- b\n\n### Bug Fixes\n\n- c\n\n### Breaking Changes\n\n- d\n- e\n- f\n", counts{2, 1, 3}},
		{"nested lists count every item", "### Features\n\n- a\n  - a1\n  - a2\n- b\n  - b1\n", counts{5, 0, 0}},
		{"missing sections", "### Bug Fixes\n\n- only fix\n", counts{0, 1, 0}},
		{"emoji and extra text", "### 🚀 New Features (beta)\n\n- a\n- b\n\n### 🐞 Bug Fixes!\n\n- c\n", counts{2, 1, 0}},
		{"h2 and h4 are ignored", "## Features\n\n- a\n\n#### Bug Fixes\n\n- b\n", counts{0, 0, 0}},
		{"paragraph between heading and list", "### Features\n\nSome intro.\n\n- a\n- b\n", counts{0, 0, 0}},
		{"heading at end", "- stray\n\n### Features\n", counts{0, 0, 0}},
		{"repeated heading, last wins", "### Features\n\n- a\n- b\n\n### More Features\n\n- c\n", counts{1, 0, 0}},
		{"emphasis splits the text", "### **Bug** Fixes\n\n- a\n- b\n", counts{0, 2, 0}},
		{"code span heading", "### `Breaking Changes`\n\n- a\n", counts{0, 0, 1}},
		{"html comment before list", "### Features\n<!-- generated -->\n- a\n- b\n", counts{2, 0, 0}},
		{"link reference definition before list", "### Features\n\n[ref]: https://example.com\n\n- a\n", counts{1, 0, 0}},
		{"ordered list", "### Bug Fixes\n\n1. a\n2. b\n3. c\n", counts{0, 3, 0}},
		{"heading inside a list item", "- ### Features\n  - a\n  - b\n", counts{2, 0, 0}},
		{"raw html heading is omitted", "<h3>Features</h3>\n<ul><li>a</li></ul>\n", counts{0, 0, 0}},
		{"blockquoted list", "### Features\n\n> - a\n> - b\n", counts{2, 0, 0}},
		{"one heading, two sections", "### Features & Bug Fixes\n\n- a\n- b\n", counts{2, 2, 0}},
		{"thematic break after heading", "### Features\n\n---\n\n- a\n", counts{0, 0, 0}},
		{"image alt text is not heading text", "### ![Features](x.png) Other\n\n- a\n", counts{0, 0, 0}},
		{"nbsp entity inside the match", "### Bug&nbsp;Fixes\n\n- a\n", counts{0, 0, 0}},
		{"autolink in heading", "### Features <https://example.com>\n\n- a\n", counts{1, 0, 0}},
		{"fenced code after heading", "### Features\n\n```\n- not an item\n```\n\n- a\n", counts{0, 0, 0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, features, bugFixes, breaking := parseBody([]byte(tt.body))
			assert.Equal(t, tt.want, counts{features, bugFixes, breaking})
		})
	}
}

func readTestdata(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

package web

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEntryAssets(t *testing.T) {
	var manifest map[string]any
	err := json.Unmarshal([]byte(`{
		"assets/main.ts": {
			"file": "assets/main-abc.js",
			"css": ["assets/main-abc.css"],
			"imports": ["_ContainerIcon-def.js", "_shared-ghi.js"],
			"dynamicImports": ["assets/pages/index.vue"]
		},
		"_ContainerIcon-def.js": {
			"file": "assets/ContainerIcon-def.js",
			"css": ["assets/ContainerIcon-def.css"],
			"imports": ["_shared-ghi.js"]
		},
		"_shared-ghi.js": {
			"file": "assets/shared-ghi.js",
			"css": ["assets/shared-ghi.css"]
		},
		"assets/pages/index.vue": {
			"file": "assets/index-jkl.js",
			"css": ["assets/pages-jkl.css"]
		}
	}`), &manifest)
	assert.NoError(t, err)

	entry, styles := entryAssets(manifest, "assets/main.ts")

	assert.Equal(t, "assets/main-abc.js", entry)
	// Statically imported chunks first, entry's own stylesheet last. Lazy page CSS is left
	// to the preload helper.
	assert.Equal(t, []string{"assets/shared-ghi.css", "assets/ContainerIcon-def.css", "assets/main-abc.css"}, styles)
}

func TestEntryAssetsMissingEntry(t *testing.T) {
	entry, styles := entryAssets(map[string]any{}, "assets/main.ts")

	assert.Empty(t, entry)
	assert.Empty(t, styles)
}

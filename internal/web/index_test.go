package web

import (
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/container/docker"
	"github.com/amir20/dozzle/internal/hostservice"
	"github.com/amir20/dozzle/internal/notification"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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

	entry, styles, preloads := entryAssets(manifest, "assets/main.ts")

	assert.Equal(t, "assets/main-abc.js", entry)
	// Statically imported chunks first, entry's own stylesheet last. Lazy page CSS is left
	// to the preload helper.
	assert.Equal(t, []string{"assets/shared-ghi.css", "assets/ContainerIcon-def.css", "assets/main-abc.css"}, styles)
	// Every statically imported chunk gets a modulepreload hint. The entry is excluded,
	// since the <script> tag already requests it, as are lazy pages.
	assert.Equal(t, []string{"assets/ContainerIcon-def.js", "assets/shared-ghi.js"}, preloads)
}

func TestEntryAssetsMissingEntry(t *testing.T) {
	entry, styles, preloads := entryAssets(map[string]any{}, "assets/main.ts")

	assert.Empty(t, entry)
	assert.Empty(t, styles)
	assert.Empty(t, preloads)
}

// The page decides whether to ask for the cloud status and the recent-alerts
// history at all from `linked`, so fetching it put a round trip in front of both.
// The server knows it at render time; this pins that the shell says so.
func Test_index_inlines_cloud_config(t *testing.T) {
	render := func(t *testing.T, cc *notification.CloudConfig) (map[string]any, string) {
		t.Helper()

		memfs := afero.NewMemMapFs()
		// The same script context the real shell uses: html/template escapes for the
		// context it is in, so the config only survives as JSON inside this tag.
		const tmpl = `<script type="application/json" id="config__json">{{ marshal .Config }}</script>`
		require.NoError(t, afero.WriteFile(memfs, "index.html", []byte(tmpl), 0644))

		client := new(MockedClient)
		client.On("ListContainers", mock.Anything, mock.Anything).Return([]container.Container{}, nil)
		client.On("Host").Return(container.Host{ID: "localhost"})
		client.On("ContainerEvents", mock.Anything, mock.AnythingOfType("chan<- container.ContainerEvent")).Return(nil)

		manager := hostservice.NewRetriableClientManager(nil, 3*time.Second, tls.Certificate{}, docker.NewService(client, container.ContainerLabels{}))
		h := &handler{
			hostService: &cloudLinkedService{HostService: hostservice.NewMultiHostService(manager, 3*time.Second), cc: cc},
			content:     afero.NewIOFS(memfs),
			config:      &Config{Base: "/", Authorization: Authorization{Provider: NONE}},
		}

		rr := httptest.NewRecorder()
		createRouter(h).ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
		require.Equal(t, http.StatusOK, rr.Code)

		body := rr.Body.String()
		_, inner, ok := strings.Cut(body, `id="config__json">`)
		require.True(t, ok, "shell did not render the config script tag: %s", body)
		inner, _, ok = strings.Cut(inner, "</script>")
		require.True(t, ok)

		var config map[string]any
		require.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(inner)), &config))
		return config, body
	}

	t.Run("linked instance ships what /api/cloud/config would answer", func(t *testing.T) {
		config, body := render(t, &notification.CloudConfig{APIKey: "super-secret-key", Prefix: "acme"})

		assert.Equal(t, map[string]any{"prefix": "acme", "linked": true, "streamLogs": true}, config["cloudConfig"])
		// Knowing the instance is linked does not need the key that does the
		// linking, and this one rides along in HTML to every signed-in reader.
		assert.NotContains(t, body, "super-secret-key")
	})

	t.Run("unlinked instance says so rather than staying quiet", func(t *testing.T) {
		config, _ := render(t, nil)

		// Present and null, not absent: the page seeds from this key, and an absent
		// one is indistinguishable from an older shell that never carried it.
		require.Contains(t, config, "cloudConfig")
		assert.Nil(t, config["cloudConfig"])
	})
}

package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/auth"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func pwaHandler(t *testing.T, config Config) http.Handler {
	t.Helper()

	fs := afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "index.html", []byte("index page"), 0644))

	return createHandler(nil, afero.NewIOFS(fs), config)
}

func get(t *testing.T, handler http.Handler, url string) *httptest.ResponseRecorder {
	t.Helper()

	req, err := http.NewRequest("GET", url, nil)
	require.NoError(t, err)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	return rr
}

func Test_manifest(t *testing.T) {
	handler := pwaHandler(t, Config{Base: "/", Authorization: Authorization{Provider: NONE}})

	var manifest map[string]any
	require.NoError(t, json.Unmarshal(get(t, handler, "/manifest.webmanifest").Body.Bytes(), &manifest))

	// Without an id the installed app is identified by start_url, so changing
	// start_url later installs a second app beside the first.
	assert.Equal(t, "/", manifest["id"])
	// The splash screen is painted from background_color before the app renders,
	// so leaving it out is a white flash on every cold start of a dark app.
	assert.Equal(t, "#121212", manifest["background_color"])
	assert.Equal(t, "#121212", manifest["theme_color"])

	icons, ok := manifest["icons"].([]any)
	require.True(t, ok)
	require.Len(t, icons, 2, "android wants a 192 alongside the 512")
	assert.Equal(t, "/favicon.png", icons[0].(map[string]any)["src"])
	assert.Equal(t, "192x192", icons[0].(map[string]any)["sizes"])
	assert.Equal(t, "/apple-touch-icon.png", icons[1].(map[string]any)["src"])

	// The logo runs to the edges of its square, so a circular mask would cut the
	// antennae off. Claiming maskable would make the Android icon worse.
	for _, icon := range icons {
		assert.NotContains(t, icon.(map[string]any), "purpose")
	}
}

// Every URL in the manifest is resolved against the manifest's own location, but
// an installed app is pinned to the scope and start_url it was installed with, so
// a missing prefix is not something the browser recovers from.
func Test_manifest_under_a_base(t *testing.T) {
	handler := pwaHandler(t, Config{Base: "/foobar", Authorization: Authorization{Provider: NONE}})

	var manifest map[string]any
	require.NoError(t, json.Unmarshal(get(t, handler, "/foobar/manifest.webmanifest").Body.Bytes(), &manifest))

	assert.Equal(t, "/foobar/", manifest["id"])
	assert.Equal(t, "/foobar/", manifest["start_url"])
	assert.Equal(t, "/foobar/", manifest["scope"])

	icons := manifest["icons"].([]any)
	assert.Equal(t, "/foobar/favicon.png", icons[0].(map[string]any)["src"])
	assert.Equal(t, "/foobar/apple-touch-icon.png", icons[1].(map[string]any)["src"])
}

func Test_offlinePage(t *testing.T) {
	handler := pwaHandler(t, Config{Base: "/", Authorization: Authorization{Provider: NONE}})

	rr := get(t, handler, "/offline.html")

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `href="/"`)
	// It is shown exactly when the network is gone, so anything it had to fetch
	// is the one thing it cannot have.
	assert.NotContains(t, rr.Body.String(), "<link")
	assert.NotContains(t, rr.Body.String(), "<script")
}

func Test_offlinePage_under_a_base(t *testing.T) {
	handler := pwaHandler(t, Config{Base: "/foobar", Authorization: Authorization{Provider: NONE}})

	assert.Contains(t, get(t, handler, "/foobar/offline.html").Body.String(), `href="/foobar/"`)
}

// The service worker has to be able to cache it while the session is expired,
// which is one of the times it is most needed.
func Test_offlinePage_needs_no_session(t *testing.T) {
	handler := pwaHandler(t, Config{Base: "/",
		Authorization: Authorization{
			Provider:   SIMPLE,
			Authorizer: auth.NewSimpleAuth(auth.UserDatabase{Users: map[string]*auth.User{}}, time.Second*100, testSecret),
		},
	})

	rr := get(t, handler, "/offline.html")

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Dozzle is unreachable")
}

func Test_serviceWorker_falls_back_for_navigations(t *testing.T) {
	handler := pwaHandler(t, Config{Base: "/foobar", Version: "dev", Authorization: Authorization{Provider: NONE}})

	body := get(t, handler, "/foobar/sw.js").Body.String()

	assert.Contains(t, body, `const OFFLINE_URL = "/foobar/offline.html";`)
	assert.Contains(t, body, `event.request.mode === "navigate"`)
	// Network-first: the fallback is only reached once the network has rejected, so
	// a stale page is never shown to someone who is online. The network here is the
	// preloaded response when there is one, and a plain fetch when there is not.
	assert.Contains(t, body, "preloaded || fetch(event.request)")
	assert.Contains(t, body, ".catch(")
	// Without this the worker has to boot before it can ask for the page, and that
	// boot lands in front of every cold navigation.
	assert.Contains(t, body, "navigationPreload?.enable()")
	assert.Contains(t, body, `const CACHE_NAME = "dozzle-dev";`)
}

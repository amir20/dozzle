package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/amir20/dozzle/internal/auth"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func oidcHandler(t *testing.T) http.Handler {
	t.Helper()

	// The fixture renders the injected config so the login page's inputs can be
	// asserted on.
	fs := afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "index.html", []byte(`<script>window.__CONFIG__ = {{ marshal .Config }};</script>`), 0644))

	// The issuer is never contacted by these tests: discovery is lazy and only
	// the login start would trigger it.
	authorizer := auth.NewOIDCAuth(auth.OIDCConfig{
		Issuer:       "http://127.0.0.1:1",
		ClientID:     "dozzle",
		ClientSecret: "secret",
		DisplayName:  "Keycloak",
	}, "", 0, testSecret)

	return createHandler(nil, afero.NewIOFS(fs), Config{Base: "/", Authorization: Authorization{
		Provider:   OIDC,
		Authorizer: authorizer,
		LogoutUrl:  "https://id.example.com/logout",
	}})
}

func Test_createRoutes_oidc_redirects_to_login(t *testing.T) {
	handler := oidcHandler(t)

	req, err := http.NewRequest("GET", "/containers/abc", nil)
	require.NoError(t, err)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusTemporaryRedirect, rr.Code)
	assert.Equal(t, "/login?redirectUrl=%2Fcontainers%2Fabc", rr.Header().Get("Location"))
}

func Test_createRoutes_oidc_login_page_offers_only_the_provider(t *testing.T) {
	handler := oidcHandler(t)

	req, err := http.NewRequest("GET", "/login", nil)
	require.NoError(t, err)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	body := rr.Body.String()
	assert.Contains(t, body, `"passwordLogin":false`)
	assert.Contains(t, body, `"name":"Keycloak"`)
	assert.Contains(t, body, `/api/auth/login?provider=oidc`)
}

// There is no password to check under oidc, so the password endpoint does not
// exist, while logout and the OAuth legs do.
func Test_createRoutes_oidc_has_no_password_login(t *testing.T) {
	handler := oidcHandler(t)

	req, err := http.NewRequest("POST", "/api/token", nil)
	require.NoError(t, err)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusMethodNotAllowed, rr.Code, "POST /api/token must not be registered")

	req, err = http.NewRequest("DELETE", "/api/token", nil)
	require.NoError(t, err)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code, "logout must clear the session")
	assert.Contains(t, rr.Header().Get("Set-Cookie"), "jwt=;")

	// A callback with no state cookie is rejected, which proves the route exists.
	req, err = http.NewRequest("GET", "/api/auth/callback?code=x&state=y", nil)
	require.NoError(t, err)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusFound, rr.Code)
	assert.Equal(t, "/login?error=oauth", rr.Header().Get("Location"))
}

func Test_createRoutes_oidc_requires_auth_for_api(t *testing.T) {
	handler := oidcHandler(t)

	req, err := http.NewRequest("GET", "/api/version", nil)
	require.NoError(t, err)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

// A provider-asserted picture is served by redirect, so Dozzle never fetches a
// URL a user may have typed into their own profile.
func Test_avatar_redirects_to_provider_picture(t *testing.T) {
	h := &handler{config: &Config{}}

	req := httptest.NewRequest("GET", "/api/profile/avatar", nil)
	req = req.WithContext(auth.WithUser(req.Context(), auth.User{Username: "abc", Picture: "https://cdn.example.com/a.png"}))
	rr := httptest.NewRecorder()
	h.avatar(rr, req)

	assert.Equal(t, http.StatusFound, rr.Code)
	assert.Equal(t, "https://cdn.example.com/a.png", rr.Header().Get("Location"))
}

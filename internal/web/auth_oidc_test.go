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
		DataDir:      t.TempDir(),
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

func avatarRequest(picture string) *http.Request {
	req := httptest.NewRequest("GET", "/api/profile/avatar", nil)
	return req.WithContext(auth.WithUser(req.Context(), auth.User{Username: "abc", Email: "a@example.com", Picture: picture}))
}

// swapAvatarClients points both clients at test servers. The picture client
// keeps its redirect policy but loses the public-address dialer, since test
// servers listen on loopback.
func swapAvatarClients(t *testing.T, picture, gravatar *httptest.Server) {
	t.Helper()
	oldPicture, oldAvatar := pictureClient, avatarClient
	t.Cleanup(func() { pictureClient, avatarClient = oldPicture, oldAvatar })

	pictureClient = &http.Client{Transport: picture.Client().Transport, CheckRedirect: oldPicture.CheckRedirect}
	gravatarURL := gravatar.URL
	avatarClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		r2, _ := http.NewRequestWithContext(r.Context(), r.Method, gravatarURL, nil)
		return gravatar.Client().Transport.RoundTrip(r2)
	})}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func imageServer(contentType, body string) *httptest.Server {
	return httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", contentType)
		w.Write([]byte(body))
	}))
}

// The CSP only allows same-origin images, so the provider picture is proxied.
func Test_avatar_proxies_provider_picture(t *testing.T) {
	picture := imageServer("image/png", "picture")
	defer picture.Close()
	gravatar := imageServer("image/png", "gravatar")
	defer gravatar.Close()
	swapAvatarClients(t, picture, gravatar)

	rr := httptest.NewRecorder()
	(&handler{config: &Config{}}).avatar(rr, avatarRequest(picture.URL+"/a.png"))

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "image/png", rr.Header().Get("Content-Type"))
	assert.Equal(t, "picture", rr.Body.String())
}

func Test_avatar_falls_back_to_gravatar_for_unsafe_picture(t *testing.T) {
	gravatar := imageServer("image/png", "gravatar")
	defer gravatar.Close()

	for name, picture := range map[string]*httptest.Server{
		"svg":       imageServer("image/svg+xml", "<svg/>"),
		"not-image": imageServer("text/html", "<html>"),
	} {
		t.Run(name, func(t *testing.T) {
			defer picture.Close()
			swapAvatarClients(t, picture, gravatar)

			rr := httptest.NewRecorder()
			(&handler{config: &Config{}}).avatar(rr, avatarRequest(picture.URL))

			assert.Equal(t, http.StatusOK, rr.Code)
			assert.Equal(t, "gravatar", rr.Body.String())
		})
	}
}

// A user who controls their picture must not be able to read internal addresses.
func Test_picture_client_refuses_private_addresses(t *testing.T) {
	internal := imageServer("image/png", "secret")
	defer internal.Close()

	_, err := pictureClient.Get(internal.URL)
	require.ErrorIs(t, err, errPrivateAddress)

	for _, addr := range []string{"127.0.0.1:443", "10.0.0.1:443", "192.168.1.1:443", "169.254.169.254:80", "100.64.0.1:443", "[::1]:443", "[fd00::1]:443", "[::ffff:127.0.0.1]:443", "0.0.0.0:443"} {
		assert.ErrorIs(t, refuseNonPublicAddress("tcp", addr, nil), errPrivateAddress, addr)
	}
	assert.NoError(t, refuseNonPublicAddress("tcp", "140.82.112.3:443", nil))
}

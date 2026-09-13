package auth

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// logoutFor runs LogoutRedirect for the session in res, the way DELETE
// /api/token does behind the middleware.
func logoutFor(t *testing.T, a *oidcAuthContext, res *http.Response) *LogoutTarget {
	t.Helper()

	jwt := sessionCookie(res)
	require.NotNil(t, jwt, "expected a session cookie")

	var target *LogoutTarget
	handler := a.AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		target = a.LogoutRedirect(r)
	}))

	req := httptest.NewRequest(http.MethodDelete, "/api/token", nil)
	req.Host = "dozzle.example.com"
	req.AddCookie(jwt)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	return target
}

func storedIDToken(t *testing.T, a *oidcAuthContext, res *http.Response) string {
	t.Helper()

	decoded, err := a.tokenAuth.Decode(sessionCookie(res).Value)
	require.NoError(t, err)
	var session string
	require.NoError(t, decoded.Get("session", &session))

	return filepath.Join(a.idTokens.dir, "abc-123", "sessions", session)
}

func TestOIDCLogoutSendsTheIDTokenBackToTheIssuer(t *testing.T) {
	a := oidcUserAuth(t, oidcUserServer{idToken: keycloakClaims("shell"), endSession: true}, OIDCConfig{})
	res := signIn(t, a)

	path := storedIDToken(t, a, res)
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm(), "someone else's ID token must not be world readable")

	target := logoutFor(t, a, res)
	require.NotNil(t, target)
	require.Equal(t, a.provider.issuer+"/logout", target.URL)
	require.Equal(t, "dozzle", target.Params["client_id"])
	require.Equal(t, "http://dozzle.example.com/login", target.Params["post_logout_redirect_uri"])
	require.Equal(t, unsignedJWT(t, keycloakClaims("shell")), target.Params["id_token_hint"])

	require.NoFileExists(t, path, "the token is forgotten once used")
}

// The whole point of keeping it on disk: an ID token carrying a data: URL
// picture is far past what a cookie holds, and logout still gets it back.
func TestOIDCLogoutHandlesAnIDTokenTooLargeForACookie(t *testing.T) {
	claims := keycloakClaims("shell")
	claims["picture"] = "data:image/png;base64," + strings.Repeat("A", 8000)
	a := oidcUserAuth(t, oidcUserServer{idToken: claims, endSession: true}, OIDCConfig{})
	res := signIn(t, a)

	target := logoutFor(t, a, res)
	require.NotNil(t, target)
	require.Equal(t, unsignedJWT(t, claims), target.Params["id_token_hint"])
}

// A token that is gone, after the container was recreated without a volume,
// still logs out at the issuer, which then asks the user to confirm.
func TestOIDCLogoutWithoutAStoredTokenOmitsTheHint(t *testing.T) {
	a := oidcUserAuth(t, oidcUserServer{idToken: keycloakClaims("shell"), endSession: true}, OIDCConfig{})
	res := signIn(t, a)
	require.NoError(t, os.Remove(storedIDToken(t, a, res)))

	target := logoutFor(t, a, res)
	require.NotNil(t, target)
	require.NotContains(t, target.Params, "id_token_hint")
	require.Equal(t, "dozzle", target.Params["client_id"])
}

func TestOIDCLogoutWithoutAnEndSessionEndpointJustReloads(t *testing.T) {
	a := oidcUserAuth(t, oidcUserServer{idToken: keycloakClaims("shell")}, OIDCConfig{})

	require.Nil(t, logoutFor(t, a, signIn(t, a)))
}

func TestOIDCLogoutURLOverridesDiscovery(t *testing.T) {
	a := oidcUserAuth(t, oidcUserServer{idToken: keycloakClaims("shell"), endSession: true}, OIDCConfig{
		LogoutURL: "https://sso.example.com/flows/invalidate",
	})

	target := logoutFor(t, a, signIn(t, a))
	require.Equal(t, &LogoutTarget{URL: "https://sso.example.com/flows/invalidate"}, target)
}

// The docs used to say to set --auth-logout-url to the end_session_endpoint.
// Those deployments get the hint without changing anything.
func TestOIDCLogoutURLMatchingDiscoveryUsesRPInitiatedLogout(t *testing.T) {
	server := oidcUserServer{idToken: keycloakClaims("shell"), endSession: true}
	a := oidcUserAuth(t, server, OIDCConfig{})
	a.logoutURL = a.provider.issuer + "/logout/"

	target := logoutFor(t, a, signIn(t, a))
	require.NotNil(t, target)
	require.NotEmpty(t, target.Params["id_token_hint"])
}

func TestOIDCLoginPrunesStaleIDTokens(t *testing.T) {
	a := oidcUserAuth(t, oidcUserServer{idToken: keycloakClaims("shell"), endSession: true}, OIDCConfig{})
	a.idTokens.retention = time.Hour

	stale := storedIDToken(t, a, signIn(t, a))
	old := time.Now().Add(-2 * time.Hour)
	require.NoError(t, os.Chtimes(stale, old, old))

	fresh := storedIDToken(t, a, signIn(t, a))
	require.NoFileExists(t, stale)
	require.FileExists(t, fresh)
}

func TestIDTokenStoreRefusesUnsafePaths(t *testing.T) {
	store := newIDTokenStore(t.TempDir(), 0)

	for _, user := range []string{"", ".", "..", "a/b"} {
		require.Error(t, store.save(user, strings.Repeat("a", 32), "token"), user)
	}
	for _, session := range []string{"", "../../etc", strings.Repeat("z", 32)} {
		require.Error(t, store.save("abc-123", session, "token"), session)
	}
}

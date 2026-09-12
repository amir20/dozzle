package auth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// oidcUserServer is a fake issuer for the oidc provider. idToken and userInfo
// are the claim maps it serves, so a test can put roles in either.
type oidcUserServer struct {
	idToken  map[string]any
	userInfo map[string]any
}

func (s oidcUserServer) start(t *testing.T) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"issuer": %q, "authorization_endpoint": %q, "token_endpoint": %q, "userinfo_endpoint": %q}`,
			server.URL, server.URL+"/authorize", server.URL+"/token", server.URL+"/userinfo")
	})

	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		body := map[string]any{"access_token": "at", "token_type": "bearer"}
		if s.idToken != nil {
			body["id_token"] = unsignedJWT(t, s.idToken)
		}
		json.NewEncoder(w).Encode(body)
	})

	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		info := s.userInfo
		if info == nil {
			info = map[string]any{"sub": "abc-123"}
		}
		json.NewEncoder(w).Encode(info)
	})

	return server
}

func unsignedJWT(t *testing.T, claims map[string]any) string {
	t.Helper()

	payload, err := json.Marshal(claims)
	require.NoError(t, err)

	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none"}`))

	return header + "." + base64.RawURLEncoding.EncodeToString(payload) + ".sig"
}

func oidcUserAuth(t *testing.T, server oidcUserServer, config OIDCConfig) *oidcAuthContext {
	t.Helper()

	issuer := server.start(t)
	config.Issuer = issuer.URL
	if config.ClientID == "" {
		config.ClientID = "dozzle"
	}
	config.ClientSecret = "secret"

	return NewOIDCAuth(config, "", 0, testSecret)
}

// signIn runs the whole flow against the fake issuer and returns the callback
// response.
func signIn(t *testing.T, a *oidcAuthContext) *http.Response {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/api/auth/login", nil)
	req.Host = "dozzle.example.com"
	w := httptest.NewRecorder()
	a.LoginHandler(w, req)

	res := w.Result()
	require.Equal(t, http.StatusFound, res.StatusCode)

	redirect, err := url.Parse(res.Header.Get("Location"))
	require.NoError(t, err)
	state := redirect.Query().Get("state")
	require.NotEmpty(t, state)

	var cookie *http.Cookie
	for _, c := range res.Cookies() {
		if c.Name == stateCookieName {
			cookie = c
		}
	}
	require.NotNil(t, cookie)

	cb := httptest.NewRequest(http.MethodGet, "/api/auth/callback?code=the-code&state="+state, nil)
	cb.AddCookie(cookie)
	cw := httptest.NewRecorder()
	a.CallbackHandler(cw, cb)

	return cw.Result()
}

// userFor runs the session cookie back through the middleware and returns the
// user it resolves, which is what every authenticated request sees.
func userFor(t *testing.T, a *oidcAuthContext, res *http.Response) *User {
	t.Helper()

	jwt := sessionCookie(res)
	require.NotNil(t, jwt, "expected a session cookie")

	var user *User
	handler := a.AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user = UserFromContext(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(jwt)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	return user
}

func keycloakClaims(roles ...any) map[string]any {
	return map[string]any{
		"sub":                "abc-123",
		"preferred_username": "amir",
		"name":               "Amir",
		"email":              "amir@example.com",
		"resource_access": map[string]any{
			"dozzle": map[string]any{"roles": roles},
		},
	}
}

// The Keycloak shape from the issue signs in with nothing configured beyond
// the issuer and client: resource_access.<client-id>.roles is on the default
// search list.
func TestOIDCAuthSignsInFromClientRolesInIDToken(t *testing.T) {
	a := oidcUserAuth(t, oidcUserServer{idToken: keycloakClaims("shell", "actions")}, OIDCConfig{})

	res := signIn(t, a)
	require.Equal(t, http.StatusFound, res.StatusCode)
	require.Equal(t, "/", res.Header.Get("Location"))

	user := userFor(t, a, res)
	require.NotNil(t, user)
	require.Equal(t, "abc-123", user.Username, "sub keys the user and the profile directory")
	require.Equal(t, "Amir", user.Name)
	require.Equal(t, "amir@example.com", user.Email)
	require.True(t, user.Roles.Has(Shell))
	require.True(t, user.Roles.Has(Actions))
	require.False(t, user.Roles.Has(Download))
	require.False(t, user.ContainerLabels.Exists())
}

// Keycloak only mirrors resource_access into userinfo when a mapper says so,
// but a provider that puts roles only there has to work too.
func TestOIDCAuthReadsRolesFromUserInfoWhenIDTokenLacksThem(t *testing.T) {
	a := oidcUserAuth(t, oidcUserServer{
		idToken:  map[string]any{"sub": "abc-123"},
		userInfo: map[string]any{"sub": "abc-123", "roles": []any{"download"}},
	}, OIDCConfig{})

	user := userFor(t, a, signIn(t, a))
	require.NotNil(t, user)
	require.Equal(t, Download, user.Roles)
}

func TestOIDCAuthRejectsWhenNoRolesClaimResolves(t *testing.T) {
	a := oidcUserAuth(t, oidcUserServer{idToken: map[string]any{
		"sub": "abc-123",
		// groups is deliberately not searched: every user has one.
		"groups": []any{"admins"},
	}}, OIDCConfig{})

	requireRejected(t, signIn(t, a))
}

func TestOIDCAuthRejectsAnEmptyRolesClaim(t *testing.T) {
	a := oidcUserAuth(t, oidcUserServer{idToken: keycloakClaims()}, OIDCConfig{})

	requireRejected(t, signIn(t, a))
}

// Present but meaningless is roles: none, not a refusal. Realm roles carry
// offline_access and friends for everyone, which is exactly this case.
func TestOIDCAuthSignsInWithNoPrivilegesWhenNoRoleMaps(t *testing.T) {
	a := oidcUserAuth(t, oidcUserServer{idToken: keycloakClaims("offline_access", "uma_authorization")}, OIDCConfig{})

	user := userFor(t, a, signIn(t, a))
	require.NotNil(t, user)
	require.Equal(t, None, user.Roles)
}

func TestOIDCAuthRejectsATokenWithoutSub(t *testing.T) {
	a := oidcUserAuth(t, oidcUserServer{
		idToken:  map[string]any{"roles": []any{"shell"}},
		userInfo: map[string]any{"roles": []any{"shell"}},
	}, OIDCConfig{})

	requireRejected(t, signIn(t, a))
}

func TestOIDCAuthExplicitClaimReplacesTheSearch(t *testing.T) {
	claims := keycloakClaims("shell")
	claims["custom"] = map[string]any{"dozzle": map[string]any{"perms": "actions"}}

	a := oidcUserAuth(t, oidcUserServer{idToken: claims}, OIDCConfig{RolesClaim: "custom.dozzle.perms"})

	user := userFor(t, a, signIn(t, a))
	require.NotNil(t, user)
	require.Equal(t, Actions, user.Roles, "the explicit path wins and the defaults are not consulted")

	// And when the explicit path is missing, the defaults do not rescue it.
	b := oidcUserAuth(t, oidcUserServer{idToken: keycloakClaims("shell")}, OIDCConfig{RolesClaim: "custom.dozzle.perms"})
	requireRejected(t, signIn(t, b))
}

func TestOIDCAuthFiltersFromClaim(t *testing.T) {
	claims := keycloakClaims("shell")
	claims["resource_access"].(map[string]any)["dozzle"].(map[string]any)["filters"] = []any{"label=com.example.app", "name=web"}

	a := oidcUserAuth(t, oidcUserServer{idToken: claims}, OIDCConfig{})

	user := userFor(t, a, signIn(t, a))
	require.NotNil(t, user)
	require.Equal(t, []string{"com.example.app"}, user.ContainerLabels["label"])
	require.Equal(t, []string{"web"}, user.ContainerLabels["name"])
}

// A filter the IdP got wrong must fail the login rather than grant every
// container, which is what silently dropping it would do.
func TestOIDCAuthRejectsAnInvalidFilter(t *testing.T) {
	claims := keycloakClaims("shell")
	claims["dozzle_filters"] = []any{"not-a-filter"}

	a := oidcUserAuth(t, oidcUserServer{idToken: claims}, OIDCConfig{})

	requireRejected(t, signIn(t, a))
}

// Zitadel encodes project roles as an object keyed by role name.
func TestOIDCAuthAcceptsObjectShapedRoles(t *testing.T) {
	a := oidcUserAuth(t, oidcUserServer{idToken: map[string]any{
		"sub":          "abc-123",
		"dozzle_roles": map[string]any{"shell": map[string]any{"org": "example.com"}},
	}}, OIDCConfig{})

	user := userFor(t, a, signIn(t, a))
	require.NotNil(t, user)
	require.Equal(t, Shell, user.Roles)
}

func TestOIDCAuthNegatedRolesWork(t *testing.T) {
	a := oidcUserAuth(t, oidcUserServer{idToken: keycloakClaims("all", "^shell")}, OIDCConfig{})

	user := userFor(t, a, signIn(t, a))
	require.NotNil(t, user)
	require.Equal(t, All&^Shell, user.Roles)
}

// An unverified or missing email is fine here: sub is the key and the email is
// only display.
func TestOIDCAuthDoesNotRequireAVerifiedEmail(t *testing.T) {
	a := oidcUserAuth(t, oidcUserServer{idToken: map[string]any{
		"sub":            "abc-123",
		"email":          "amir@example.com",
		"email_verified": false,
		"roles":          []any{"shell"},
	}}, OIDCConfig{})

	user := userFor(t, a, signIn(t, a))
	require.NotNil(t, user)
	require.Equal(t, "amir@example.com", user.Email)

	b := oidcUserAuth(t, oidcUserServer{idToken: map[string]any{"sub": "abc-123", "roles": []any{"shell"}}}, OIDCConfig{})
	user = userFor(t, b, signIn(t, b))
	require.NotNil(t, user)
	require.Equal(t, "abc-123", user.Name, "with nothing else to show, the name falls back to sub")
}

func TestOIDCAuthDisplayNameFallsBackThroughUsernameAndEmail(t *testing.T) {
	a := oidcUserAuth(t, oidcUserServer{idToken: map[string]any{
		"sub":                "abc-123",
		"preferred_username": "amir",
		"roles":              []any{"shell"},
	}}, OIDCConfig{})
	require.Equal(t, "amir", userFor(t, a, signIn(t, a)).Name)

	b := oidcUserAuth(t, oidcUserServer{idToken: map[string]any{
		"sub":   "abc-123",
		"email": "amir@example.com",
		"roles": []any{"shell"},
	}}, OIDCConfig{})
	require.Equal(t, "amir@example.com", userFor(t, b, signIn(t, b)).Name)
}

func TestOIDCAuthPictureFeedsTheAvatar(t *testing.T) {
	a := oidcUserAuth(t, oidcUserServer{idToken: map[string]any{
		"sub":     "abc-123",
		"picture": "https://cdn.example.com/amir.png",
		"roles":   []any{"shell"},
	}}, OIDCConfig{})

	user := userFor(t, a, signIn(t, a))
	require.Equal(t, "https://cdn.example.com/amir.png", user.PictureURL())
	require.Contains(t, user.AvatarURL(), "gravatar.com", "the gravatar fallback is untouched")

	// Anything but an https URL is dropped rather than handed to the browser.
	for _, picture := range []string{"javascript:alert(1)", "http://cdn.example.com/amir.png", "/relative.png", "https://"} {
		user.Picture = picture
		require.Empty(t, user.PictureURL(), picture)
	}
}

// Roles and filters ride in the session and are re-parsed per request, so the
// middleware needs nothing but the cookie.
func TestOIDCAuthSessionCarriesRolesAndFilters(t *testing.T) {
	claims := keycloakClaims("shell")
	claims["dozzle_filters"] = "label=team=a"
	a := oidcUserAuth(t, oidcUserServer{idToken: claims}, OIDCConfig{})

	jwt := sessionCookie(signIn(t, a))
	decoded, err := a.tokenAuth.Decode(jwt.Value)
	require.NoError(t, err)

	var roles, filters, sub string
	require.NoError(t, decoded.Get("roles", &roles))
	require.NoError(t, decoded.Get("filters", &filters))
	require.NoError(t, decoded.Get("sub", &sub))
	require.Equal(t, "shell", roles)
	require.Equal(t, "label=team=a", filters)
	require.Equal(t, "abc-123", sub)
}

func TestOIDCAuthSessionExpiresWithTTL(t *testing.T) {
	issuer := oidcUserServer{idToken: keycloakClaims("shell")}.start(t)
	a := NewOIDCAuth(OIDCConfig{Issuer: issuer.URL, ClientID: "dozzle", ClientSecret: "secret"}, "", time.Hour, testSecret)

	res := signIn(t, a)
	jwt := sessionCookie(res)
	require.NotNil(t, jwt)
	require.WithinDuration(t, time.Now().Add(time.Hour), jwt.Expires, time.Minute)

	decoded, err := a.tokenAuth.Decode(jwt.Value)
	require.NoError(t, err)
	exp, ok := decoded.Expiration()
	require.True(t, ok)
	require.WithinDuration(t, time.Now().Add(time.Hour), exp, time.Minute)
}

// Repointing Dozzle at another issuer must invalidate sessions minted under the
// old one: the same sub at a different issuer is a different person.
func TestOIDCAuthSessionsDoNotSurviveAnIssuerChange(t *testing.T) {
	a := oidcUserAuth(t, oidcUserServer{idToken: keycloakClaims("shell")}, OIDCConfig{})
	jwt := sessionCookie(signIn(t, a))

	b := NewOIDCAuth(OIDCConfig{Issuer: "https://other.example.com", ClientID: "dozzle", ClientSecret: "secret"}, "", 0, testSecret)

	var user *User
	handler := b.AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user = UserFromContext(r.Context())
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(jwt)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	require.Nil(t, user)
}

func TestOIDCAuthHasNoPasswordLogin(t *testing.T) {
	a := oidcUserAuth(t, oidcUserServer{}, OIDCConfig{DisplayName: "Keycloak"})

	require.False(t, a.PasswordLoginEnabled())
	_, err := a.CreateToken("amir", "password")
	require.ErrorIs(t, err, ErrPasswordLoginUnavailable)

	providers := a.Providers()
	require.Len(t, providers, 1)
	require.Equal(t, "Keycloak", providers[0].Name)
	require.Equal(t, "/api/auth/login?provider=oidc", providers[0].LoginURL)
}

func TestOIDCAuthDescribesItsClaimSearch(t *testing.T) {
	a := oidcUserAuth(t, oidcUserServer{}, OIDCConfig{ClientID: "my-client"})
	require.Equal(t, "dozzle_roles, resource_access.my-client.roles, roles", a.RolesClaims())

	b := oidcUserAuth(t, oidcUserServer{}, OIDCConfig{RolesClaim: "realm_access.roles"})
	require.Equal(t, "realm_access.roles", b.RolesClaims())
}

// A userinfo response for a different subject than the ID token is not the
// same user, and the spec has the client check that.
func TestOIDCAuthRejectsUserInfoSubMismatch(t *testing.T) {
	a := oidcUserAuth(t, oidcUserServer{
		idToken:  keycloakClaims("shell"),
		userInfo: map[string]any{"sub": "someone-else"},
	}, OIDCConfig{})

	requireRejected(t, signIn(t, a))
}

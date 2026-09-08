package auth

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

type oidcOptions struct {
	// issuer overrides the issuer the discovery document claims.
	issuer string
	// emailVerified is written into userinfo verbatim, so a test can send the
	// string "true" some issuers use instead of a boolean.
	emailVerified string
	email         string
	// discoveryStatus overrides the discovery response code.
	discoveryStatus int
}

// oidcServer stands in for an OpenID provider and records the redirect_uri it
// was handed at the token exchange.
func oidcServer(t *testing.T, opts oidcOptions) (*httptest.Server, *atomic.Value, *atomic.Int32) {
	t.Helper()

	var tokenRedirectURI atomic.Value
	tokenRedirectURI.Store("")
	var discoveryHits atomic.Int32

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		discoveryHits.Add(1)

		if opts.discoveryStatus != 0 && opts.discoveryStatus != http.StatusOK {
			w.WriteHeader(opts.discoveryStatus)
			return
		}

		issuer := server.URL
		if opts.issuer != "" {
			issuer = opts.issuer
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{
			"issuer": %q,
			"authorization_endpoint": %q,
			"token_endpoint": %q,
			"userinfo_endpoint": %q,
			"jwks_uri": %q
		}`, issuer, server.URL+"/authorize", server.URL+"/token", server.URL+"/userinfo", server.URL+"/jwks")
	})

	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		tokenRedirectURI.Store(r.Form.Get("redirect_uri"))
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"access_token":"at","token_type":"bearer","id_token":"unused"}`)
	})

	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, r *http.Request) {
		verified := opts.emailVerified
		if verified == "" {
			verified = "true"
		}
		email := opts.email
		if email == "" {
			email = "amir@example.com"
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"sub":"abc-123","email":%q,"email_verified":%s,"name":"Amir","preferred_username":"amir"}`, email, verified)
	})

	return server, &tokenRedirectURI, &discoveryHits
}

func testOIDCProvider(t *testing.T, opts oidcOptions) (*oidcProvider, *atomic.Value, *atomic.Int32) {
	t.Helper()

	server, redirectURI, hits := oidcServer(t, opts)
	p := NewOIDCProvider(server.URL, "client-id", "client-secret", "Keycloak")

	return p, redirectURI, hits
}

// oidcAuth builds an oauth context whose single user is linked by email.
func oidcAuth(t *testing.T, p IdentityProvider) *oauthAuthContext {
	t.Helper()

	user := &User{Username: "amir", Email: "amir@example.com", RolesConfigured: "all"}
	db := UserDatabase{
		Users:    map[string]*User{"amir": user},
		byEmail:  map[string]*User{normalizeEmail(user.Email): user},
		byGithub: map[string]*User{},
	}

	return NewOAuthAuth(NewSimpleAuth(db, 0, testSecret), "", p)
}

// startOIDCLogin drives LoginHandler with a real Host so the callback URL is
// derived the way it is in production.
func startOIDCLogin(t *testing.T, a *oauthAuthContext, https bool) (*http.Cookie, string, string) {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/api/auth/login?provider=oidc", nil)
	req.Host = "dozzle.example.com"
	if https {
		req.Header.Set("X-Forwarded-Proto", "https")
	}

	w := httptest.NewRecorder()
	a.LoginHandler(w, req)

	res := w.Result()
	require.Equal(t, http.StatusFound, res.StatusCode)

	redirect, err := url.Parse(res.Header.Get("Location"))
	require.NoError(t, err)

	var cookie *http.Cookie
	for _, c := range res.Cookies() {
		if c.Name == stateCookieName {
			cookie = c
		}
	}
	require.NotNil(t, cookie)

	return cookie, redirect.Query().Get("state"), redirect.Query().Get("redirect_uri")
}

// OIDC, unlike GitHub, must send redirect_uri, and the token exchange has to
// replay the identical value or the provider rejects it.
func TestOIDCSendsAndReplaysRedirectURI(t *testing.T) {
	p, tokenRedirectURI, _ := testOIDCProvider(t, oidcOptions{})
	a := oidcAuth(t, p)

	cookie, state, authRedirectURI := startOIDCLogin(t, a, true)
	require.Equal(t, "https://dozzle.example.com/api/auth/callback", authRedirectURI)

	res := callback(t, a, cookie, "code=the-code&state="+state)
	require.NotNil(t, sessionCookie(res), "a verified, linked email should sign in")
	require.Equal(t, authRedirectURI, tokenRedirectURI.Load(), "the token exchange must replay the authorize redirect_uri")
}

// X-Forwarded-Proto is what tells Dozzle the browser used HTTPS when TLS is
// terminated upstream; getting it wrong builds an http:// callback the provider
// has never seen.
func TestOIDCCallbackURLRespectsForwardedProto(t *testing.T) {
	p, _, _ := testOIDCProvider(t, oidcOptions{})
	a := oidcAuth(t, p)

	_, _, secure := startOIDCLogin(t, a, true)
	require.Equal(t, "https://dozzle.example.com/api/auth/callback", secure)

	_, _, plain := startOIDCLogin(t, a, false)
	require.Equal(t, "http://dozzle.example.com/api/auth/callback", plain)
}

func TestOIDCCallbackURLCarriesBase(t *testing.T) {
	p, _, _ := testOIDCProvider(t, oidcOptions{})
	user := &User{Username: "amir", Email: "amir@example.com"}
	db := UserDatabase{Users: map[string]*User{"amir": user}, byEmail: map[string]*User{"amir@example.com": user}}
	a := NewOAuthAuth(NewSimpleAuth(db, 0, testSecret), "/dozzle", p)

	_, _, redirectURI := startOIDCLogin(t, a, true)
	require.Equal(t, "https://dozzle.example.com/dozzle/api/auth/callback", redirectURI)
}

// Matching an unverified address would let anyone who can register at a sloppy
// IdP claim a Dozzle account by typing in someone else's email.
func TestOIDCRejectsUnverifiedEmail(t *testing.T) {
	p, _, _ := testOIDCProvider(t, oidcOptions{emailVerified: "false"})
	a := oidcAuth(t, p)

	cookie, state, _ := startOIDCLogin(t, a, true)
	requireRejected(t, callback(t, a, cookie, "code=the-code&state="+state))
}

// Some issuers send email_verified as a string.
func TestOIDCAcceptsStringEmailVerified(t *testing.T) {
	p, _, _ := testOIDCProvider(t, oidcOptions{emailVerified: `"true"`})
	a := oidcAuth(t, p)

	cookie, state, _ := startOIDCLogin(t, a, true)
	require.NotNil(t, sessionCookie(callback(t, a, cookie, "code=the-code&state="+state)))
}

// users.yml is the allowlist for OIDC exactly as it is for GitHub.
func TestOIDCRejectsEmailNotInUsersYml(t *testing.T) {
	p, _, _ := testOIDCProvider(t, oidcOptions{email: "stranger@example.com"})
	a := oidcAuth(t, p)

	cookie, state, _ := startOIDCLogin(t, a, true)
	requireRejected(t, callback(t, a, cookie, "code=the-code&state="+state))
}

func TestOIDCMatchesEmailCaseInsensitively(t *testing.T) {
	p, _, _ := testOIDCProvider(t, oidcOptions{email: "AMIR@Example.COM"})
	a := oidcAuth(t, p)

	cookie, state, _ := startOIDCLogin(t, a, true)
	require.NotNil(t, sessionCookie(callback(t, a, cookie, "code=the-code&state="+state)))
}

// A discovery document naming a different issuer means the URL is not the
// authority it claims to be, which is the classic mix-up setup.
func TestOIDCRejectsIssuerMismatch(t *testing.T) {
	p, _, _ := testOIDCProvider(t, oidcOptions{issuer: "https://evil.example.com"})

	_, err := p.discover(context.Background())
	require.ErrorContains(t, err, "does not match the configured issuer")
}

// An IdP that is down at boot must not disable SSO until the next restart.
func TestOIDCDiscoveryFailureIsNotCached(t *testing.T) {
	server, _, hits := oidcServer(t, oidcOptions{discoveryStatus: http.StatusInternalServerError})
	p := NewOIDCProvider(server.URL, "id", "secret", "SSO")

	_, err := p.discover(context.Background())
	require.Error(t, err)
	_, err = p.discover(context.Background())
	require.Error(t, err)

	require.EqualValues(t, 2, hits.Load(), "a failed discovery must be retried, not cached")
}

func TestOIDCDiscoverySucceedsOnceAndCaches(t *testing.T) {
	p, _, hits := testOIDCProvider(t, oidcOptions{})

	for range 3 {
		_, err := p.discover(context.Background())
		require.NoError(t, err)
	}

	require.EqualValues(t, 1, hits.Load(), "a successful discovery should be cached")
}

func TestOIDCIdentityReadsUserInfo(t *testing.T) {
	p, _, _ := testOIDCProvider(t, oidcOptions{})

	id, err := p.identity(context.Background(), &oauth2.Token{AccessToken: "at"})
	require.NoError(t, err)

	require.Equal(t, "abc-123", id.Sub)
	require.Equal(t, "amir@example.com", id.Email)
	require.Equal(t, "Amir", id.Name)
	require.Equal(t, "amir", id.Login)
}

func TestOIDCProviderDescribesItsButton(t *testing.T) {
	p, _, _ := testOIDCProvider(t, oidcOptions{})
	a := oidcAuth(t, p)

	providers := a.Providers()
	require.Len(t, providers, 1)
	require.Equal(t, "Keycloak", providers[0].Name, "the button label is operator-configurable")
	require.Equal(t, "/api/auth/login?provider=oidc", providers[0].LoginURL)
}

// Both vendors have to coexist, each on its own ?provider= value.
func TestGithubAndOIDCCoexist(t *testing.T) {
	oidc, _, _ := testOIDCProvider(t, oidcOptions{})
	github := testGithubProvider(t, "amir20")

	user := &User{Username: "amir", Email: "amir@example.com", Github: "amir20"}
	db := UserDatabase{
		Users:    map[string]*User{"amir": user},
		byEmail:  map[string]*User{"amir@example.com": user},
		byGithub: map[string]*User{"amir20": user},
	}
	a := NewOAuthAuth(NewSimpleAuth(db, 0, testSecret), "", github, oidc)

	providers := a.Providers()
	require.Len(t, providers, 2)
	require.Equal(t, "/api/auth/login?provider=github", providers[0].LoginURL)
	require.Equal(t, "/api/auth/login?provider=oidc", providers[1].LoginURL)

	// With more than one configured, an omitted ?provider= is ambiguous.
	w := httptest.NewRecorder()
	a.LoginHandler(w, httptest.NewRequest(http.MethodGet, "/api/auth/login", nil))
	require.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
}

// A provider that is unreachable when the first login starts must fail closed.
//
// oauth2Config used to swallow the discovery error and hand back a config with
// a zero-value Endpoint. AuthCodeURL on that emits a bare query string, so
// http.Redirect sent the browser to a relative URL back inside Dozzle, carrying
// the state and PKCE parameters, instead of showing the login error. Discovery
// is lazy, so this was reachable on the very first click after a restart.
func TestOIDCLoginFailsClosedWhenDiscoveryIsDown(t *testing.T) {
	// Port 1 refuses connections immediately.
	p := NewOIDCProvider("http://127.0.0.1:1", "id", "secret", "SSO")
	a := oidcAuth(t, p)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/auth/login?provider=oidc", nil)
	req.Host = "dozzle.example.com"
	a.LoginHandler(w, req)

	res := w.Result()
	require.Equal(t, http.StatusFound, res.StatusCode)
	require.Equal(t, "/login?error=oauth", res.Header.Get("Location"),
		"an unreachable issuer must land on the login page, not a relative OAuth URL")

	for _, c := range res.Cookies() {
		require.NotEqual(t, stateCookieName, c.Name, "no state should be minted for a login that cannot start")
	}
}

// The same guard on the callback leg, where a config is rebuilt to exchange.
func TestOIDCCallbackFailsClosedWhenDiscoveryIsDown(t *testing.T) {
	p, _, _ := testOIDCProvider(t, oidcOptions{})
	a := oidcAuth(t, p)

	cookie, state, _ := startOIDCLogin(t, a, true)

	// Point the provider at a dead issuer and drop the cached discovery, which
	// is what a restart between the two legs looks like.
	p.mu.Lock()
	p.discovery = nil
	p.issuer = "http://127.0.0.1:1"
	p.mu.Unlock()

	requireRejected(t, callback(t, a, cookie, "code=the-code&state="+state))
}

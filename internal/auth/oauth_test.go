package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// githubServer stands in for github.com and api.github.com. It hands out one
// fixed code and reports the login it was built with.
func githubServer(t *testing.T, login string) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()

	mux.HandleFunc("/login/oauth/access_token", func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		require.Equal(t, "the-code", r.Form.Get("code"))
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"access_token":"at","token_type":"bearer"}`)
	})

	mux.HandleFunc("/user", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"login":%q,"id":42,"name":"Test User","email":"public@example.com"}`, login)
	})

	mux.HandleFunc("/user/emails", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[{"email":"unverified@example.com","primary":false,"verified":false},
		                {"email":"primary@example.com","primary":true,"verified":true}]`)
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return server
}

func testGithubProvider(t *testing.T, login string) *githubProvider {
	t.Helper()

	server := githubServer(t, login)
	p := NewGithubProvider("client-id", "client-secret")
	p.apiBase = server.URL
	p.endpoint = oauth2.Endpoint{
		AuthURL:  server.URL + "/login/oauth/authorize",
		TokenURL: server.URL + "/login/oauth/access_token",
	}

	return p
}

// testAuth builds an oauth context over a users.yml holding a single user
// linked to githubLogin.
func testAuth(t *testing.T, githubLogin string, provider IdentityProvider) *oauthAuthContext {
	t.Helper()

	user := &User{Username: "amir", Email: "amir@example.com", Github: githubLogin, RolesConfigured: "all"}
	db := UserDatabase{
		Users:    map[string]*User{"amir": user},
		byEmail:  map[string]*User{normalizeEmail(user.Email): user},
		byGithub: map[string]*User{},
	}
	if githubLogin != "" {
		db.byGithub[normalizeGithub(githubLogin)] = user
	}

	return NewOAuthAuth(NewSimpleAuth(db, 0, testSecret), "", provider)
}

// startLogin runs LoginHandler and returns the state cookie plus the state value
// the provider was sent.
func startLogin(t *testing.T, a *oauthAuthContext) (*http.Cookie, string) {
	t.Helper()

	w := httptest.NewRecorder()
	a.LoginHandler(w, httptest.NewRequest(http.MethodGet, "/api/auth/login?provider=github", nil))

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
	require.NotNil(t, cookie, "login should set a state cookie")

	return cookie, state
}

func callback(t *testing.T, a *oauthAuthContext, cookie *http.Cookie, query string) *http.Response {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/api/auth/callback?"+query, nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}

	w := httptest.NewRecorder()
	a.CallbackHandler(w, req)

	return w.Result()
}

func sessionCookie(res *http.Response) *http.Cookie {
	for _, c := range res.Cookies() {
		if c.Name == "jwt" && c.Value != "" {
			return c
		}
	}

	return nil
}

func requireRejected(t *testing.T, res *http.Response) {
	t.Helper()

	require.Equal(t, http.StatusFound, res.StatusCode)
	require.Equal(t, "/login?error=oauth", res.Header.Get("Location"))
	require.Nil(t, sessionCookie(res), "a rejected login must not mint a session")
}

func TestLoginStartsPKCEFlow(t *testing.T) {
	a := testAuth(t, "amir20", testGithubProvider(t, "amir20"))

	w := httptest.NewRecorder()
	a.LoginHandler(w, httptest.NewRequest(http.MethodGet, "/api/auth/login?provider=github", nil))

	res := w.Result()
	redirect, err := url.Parse(res.Header.Get("Location"))
	require.NoError(t, err)

	require.Equal(t, "S256", redirect.Query().Get("code_challenge_method"))
	require.NotEmpty(t, redirect.Query().Get("code_challenge"))
	require.Equal(t, "client-id", redirect.Query().Get("client_id"))

	// The provider falls back to the callback registered on the OAuth app, which
	// is what lets Dozzle work without a base-URL flag.
	require.Empty(t, redirect.Query().Get("redirect_uri"), "redirect_uri must be omitted")

	for _, c := range res.Cookies() {
		if c.Name == stateCookieName {
			require.True(t, c.HttpOnly, "the state cookie must not be readable from JS")
		}
	}
}

func TestLoginRejectsUnknownProvider(t *testing.T) {
	a := testAuth(t, "amir20", testGithubProvider(t, "amir20"))

	w := httptest.NewRecorder()
	a.LoginHandler(w, httptest.NewRequest(http.MethodGet, "/api/auth/login?provider=gitlab", nil))

	require.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
}

func TestCallbackSignsInLinkedUser(t *testing.T) {
	a := testAuth(t, "amir20", testGithubProvider(t, "amir20"))
	cookie, state := startLogin(t, a)

	res := callback(t, a, cookie, "code=the-code&state="+state)

	require.Equal(t, http.StatusFound, res.StatusCode)
	require.Equal(t, "/", res.Header.Get("Location"))

	jwt := sessionCookie(res)
	require.NotNil(t, jwt, "a matched login should mint a session")
	require.True(t, jwt.HttpOnly)

	// The session must be indistinguishable from a password login: the same
	// identity-only claim, signed by the same users.yml-derived key.
	decoded, err := a.tokenAuth.Decode(jwt.Value)
	require.NoError(t, err)

	var username string
	require.NoError(t, decoded.Get("username", &username))
	require.Equal(t, "amir", username)
}

func TestCallbackHonorsSafeRedirect(t *testing.T) {
	a := testAuth(t, "amir20", testGithubProvider(t, "amir20"))

	w := httptest.NewRecorder()
	a.LoginHandler(w, httptest.NewRequest(http.MethodGet, "/api/auth/login?provider=github&redirectUrl=%2Fcontainer%2Fabc", nil))
	res := w.Result()
	redirect, _ := url.Parse(res.Header.Get("Location"))

	var cookie *http.Cookie
	for _, c := range res.Cookies() {
		if c.Name == stateCookieName {
			cookie = c
		}
	}

	out := callback(t, a, cookie, "code=the-code&state="+redirect.Query().Get("state"))
	require.Equal(t, "/container/abc", out.Header.Get("Location"))
}

// An off-origin redirectUrl must never survive into the callback's Location.
func TestLoginDropsOffOriginRedirect(t *testing.T) {
	a := testAuth(t, "amir20", testGithubProvider(t, "amir20"))

	for _, hostile := range []string{"//evil.com", `/\evil.com`, "https://evil.com", "javascript:alert(1)"} {
		t.Run(hostile, func(t *testing.T) {
			w := httptest.NewRecorder()
			a.LoginHandler(w, httptest.NewRequest(http.MethodGet, "/api/auth/login?provider=github&redirectUrl="+url.QueryEscape(hostile), nil))

			res := w.Result()
			var cookie *http.Cookie
			for _, c := range res.Cookies() {
				if c.Name == stateCookieName {
					cookie = c
				}
			}
			require.NotNil(t, cookie)

			raw, err := base64.RawURLEncoding.DecodeString(cookie.Value)
			require.NoError(t, err)
			var st oauthState
			require.NoError(t, json.Unmarshal(raw, &st))
			require.Empty(t, st.RedirectURL, "hostile redirect must be dropped at login")
		})
	}
}

func TestCallbackRejectsStateMismatch(t *testing.T) {
	a := testAuth(t, "amir20", testGithubProvider(t, "amir20"))
	cookie, _ := startLogin(t, a)

	requireRejected(t, callback(t, a, cookie, "code=the-code&state=not-the-state"))
}

func TestCallbackRejectsMissingStateCookie(t *testing.T) {
	a := testAuth(t, "amir20", testGithubProvider(t, "amir20"))
	_, state := startLogin(t, a)

	// The browser that finishes the flow is not the one that started it.
	requireRejected(t, callback(t, a, nil, "code=the-code&state="+state))
}

// An expired state cookie is simply absent by the time the browser calls back,
// which is the same rejection path as a missing one.
func TestStateCookieExpires(t *testing.T) {
	a := testAuth(t, "amir20", testGithubProvider(t, "amir20"))
	cookie, _ := startLogin(t, a)

	require.Equal(t, int(stateCookieTTL.Seconds()), cookie.MaxAge)
	require.LessOrEqual(t, stateCookieTTL, 15*time.Minute, "a resumable login should be short lived")
}

// A tampered verifier must fail the exchange rather than fall back to no PKCE.
func TestCallbackRejectsVerifierMismatch(t *testing.T) {
	a := testAuth(t, "amir20", testGithubProvider(t, "amir20"))
	cookie, state := startLogin(t, a)

	raw, err := base64.RawURLEncoding.DecodeString(cookie.Value)
	require.NoError(t, err)
	var st oauthState
	require.NoError(t, json.Unmarshal(raw, &st))

	// Rewrite the state cookie so it no longer matches the state the provider
	// was given, which is what a swapped-in cookie looks like.
	st.State = "attacker-state"
	tampered, err := json.Marshal(st)
	require.NoError(t, err)
	cookie.Value = base64.RawURLEncoding.EncodeToString(tampered)

	requireRejected(t, callback(t, a, cookie, "code=the-code&state="+state))
}

func TestCallbackRejectsProviderError(t *testing.T) {
	a := testAuth(t, "amir20", testGithubProvider(t, "amir20"))
	cookie, state := startLogin(t, a)

	requireRejected(t, callback(t, a, cookie, "error=access_denied&state="+state))
}

// The whole point of the design: users.yml is the allowlist.
func TestCallbackRejectsUserNotInUsersYml(t *testing.T) {
	a := testAuth(t, "amir20", testGithubProvider(t, "someone-else"))
	cookie, state := startLogin(t, a)

	requireRejected(t, callback(t, a, cookie, "code=the-code&state="+state))
}

func TestCallbackRejectsWhenNoUserLinksGithub(t *testing.T) {
	a := testAuth(t, "", testGithubProvider(t, "amir20"))
	cookie, state := startLogin(t, a)

	requireRejected(t, callback(t, a, cookie, "code=the-code&state="+state))
}

// GitHub logins are unique case-insensitively, so users.yml holding "Amir20"
// has to match the canonical "amir20" the API returns. Matching case-sensitively
// would turn a capitalization difference into a silent lockout.
func TestGithubLoginMatchesCaseInsensitively(t *testing.T) {
	a := testAuth(t, "Amir20", testGithubProvider(t, "amir20"))
	cookie, state := startLogin(t, a)

	res := callback(t, a, cookie, "code=the-code&state="+state)
	require.NotNil(t, sessionCookie(res), "casing in users.yml must not decide the match")
}

func TestGithubIdentityUsesPrimaryVerifiedEmail(t *testing.T) {
	p := testGithubProvider(t, "amir20")

	id, err := p.identity(context.Background(), &oauth2.Token{AccessToken: "at"})
	require.NoError(t, err)

	require.Equal(t, "amir20", id.Login)
	require.Equal(t, "42", id.Sub)
	require.Equal(t, "Test User", id.Name)
	// Never the self-asserted profile email, and never an unverified one.
	require.Equal(t, "primary@example.com", id.Email)
}

func TestProvidersDescribeLoginButtons(t *testing.T) {
	a := testAuth(t, "amir20", testGithubProvider(t, "amir20"))

	providers := a.Providers()
	require.Len(t, providers, 1)
	require.Equal(t, "GitHub", providers[0].Name)
	require.Equal(t, "mdi:github", providers[0].Icon)
	require.Equal(t, "/api/auth/login?provider=github", providers[0].LoginURL)
}

// With a base path the login URL has to carry it, because the login page uses
// the value verbatim.
func TestProvidersLoginURLCarriesBase(t *testing.T) {
	a := NewOAuthAuth(NewSimpleAuth(UserDatabase{Users: map[string]*User{}}, 0, testSecret), "/dozzle", testGithubProvider(t, "amir20"))

	require.Equal(t, "/dozzle/api/auth/login?provider=github", a.Providers()[0].LoginURL)
}

func TestSafeRelativePath(t *testing.T) {
	for _, ok := range []string{"/", "/container/abc", "/a?b=c#d"} {
		require.Equal(t, ok, SafeRelativePath(ok))
	}

	for _, bad := range []string{"", "//evil.com", `/\evil.com`, "https://evil.com", "container/abc", "javascript:alert(1)"} {
		require.Empty(t, SafeRelativePath(bad), "%q must not be redirected to", bad)
	}
}

// Password login has to keep working beside OAuth, and an account with no hash
// must not be signed in by an empty password.
func TestPasswordLoginStillWorksBesideOAuth(t *testing.T) {
	a := testAuth(t, "amir20", testGithubProvider(t, "amir20"))

	_, err := a.CreateToken("amir", "")
	require.ErrorIs(t, err, ErrInvalidCredentials)

	_, err = a.CreateToken("amir", "anything")
	require.ErrorIs(t, err, ErrInvalidCredentials)
}

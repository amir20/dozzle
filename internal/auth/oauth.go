package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2"
)

// externalIdentity is what a provider tells us about whoever just signed in.
//
// Under simple auth it is deliberately not a User: users.yml is the only place a
// Dozzle user is defined, and this exists only to look one up there. Under the
// oidc provider there is no users.yml, and Claims is what the user is built from.
type externalIdentity struct {
	Sub     string
	Login   string
	Email   string
	Name    string
	Picture string
	// Claims is every claim the provider returned, ID token first and userinfo
	// second, for the oidc provider to read roles and filters out of. Empty for
	// GitHub, which publishes no claims.
	Claims claimSet
}

// userLookup is the locked view of users.yml handed to a provider, so a provider
// decides which key it matches on without also owning the reload lock.
type userLookup interface {
	byGithubLogin(login string) (User, bool)
	byVerifiedEmail(email string) (User, bool)
}

// IdentityProvider is one OAuth vendor. The handlers below know nothing about
// GitHub, so the generic OIDC provider slots in beside it without touching them.
type IdentityProvider interface {
	// ID is the value of the ?provider= query parameter.
	ID() string
	// DisplayName and Icon are what the login page renders on the button.
	DisplayName() string
	Icon() string

	// oauth2Config builds the exchange config. callbackURI is this request's own
	// callback URL; GitHub ignores it and lets the OAuth app decide, while OIDC
	// must send it and replay the identical value at the token exchange.
	//
	// It returns an error rather than a partially built config, because OIDC
	// resolves its endpoints from a discovery document that can be unreachable.
	// A zero-value Endpoint makes AuthCodeURL emit a bare query string, which
	// redirects the browser back into Dozzle instead of to the provider.
	oauth2Config(callbackURI string) (*oauth2.Config, error)
	identity(ctx context.Context, token *oauth2.Token) (externalIdentity, error)
	// match resolves an identity against users.yml. GitHub matches on the login;
	// generic OIDC will match on the verified email.
	match(users userLookup, id externalIdentity) (User, bool)
}

// OAuthProviderInfo is the shape the login page renders a button from.
type OAuthProviderInfo struct {
	Name     string `json:"name"`
	LoginURL string `json:"loginUrl"`
	Icon     string `json:"icon"`
}

// oauthFlow is the authorization code dance shared by simple auth with OAuth
// bolted on and by the oidc provider: state cookie, PKCE, code exchange, and
// reading the identity back. What happens to that identity is the one thing the
// two disagree on, which is why it is a callback rather than a method.
type oauthFlow struct {
	providers []IdentityProvider
	base      string
	ttl       time.Duration
	// login turns a verified identity into a session JWT, or refuses it. A refusal
	// has already been logged with whatever detail the owner wanted to give.
	login func(provider IdentityProvider, id externalIdentity) (string, bool)
}

func newOAuthFlow(base string, ttl time.Duration, login func(IdentityProvider, externalIdentity) (string, bool), providers ...IdentityProvider) *oauthFlow {
	if base == "/" {
		base = ""
	}

	return &oauthFlow{providers: providers, base: base, ttl: ttl, login: login}
}

// oauthAuthContext is simple auth with OAuth bolted on as a second way to prove
// you are one of the users already in users.yml. It embeds the simple context so
// password login, the middleware, and per-request role resolution are untouched.
type oauthAuthContext struct {
	*simpleAuthContext
	*oauthFlow
}

// NewOAuthAuth wraps simple auth. base is the router base ("" when Dozzle is
// mounted at /), used to build the login URLs and the post-login redirect.
func NewOAuthAuth(simple *simpleAuthContext, base string, providers ...IdentityProvider) *oauthAuthContext {
	a := &oauthAuthContext{simpleAuthContext: simple}
	a.oauthFlow = newOAuthFlow(base, simple.ttl, a.loginUser, providers...)

	return a
}

func (a *oauthAuthContext) byGithubLogin(login string) (User, bool) {
	return a.findByGithub(login)
}

func (a *oauthAuthContext) byVerifiedEmail(email string) (User, bool) {
	return a.findByEmail(email)
}

// loginUser is the simple-auth half of the flow: users.yml is the allowlist. No
// match means no login, and no account is ever created here.
func (a *oauthAuthContext) loginUser(provider IdentityProvider, identity externalIdentity) (string, bool) {
	user, ok := provider.match(a, identity)
	if !ok {
		log.Warn().
			Str("provider", provider.ID()).
			Str("login", identity.Login).
			Msg("OAuth login rejected: no user in the user database is linked to this account")
		return "", false
	}

	jwt, err := a.issueToken(user)
	if err != nil {
		log.Error().Err(err).Msg("Could not create token after OAuth login")
		return "", false
	}

	log.Info().Str("user", user.Username).Str("provider", provider.ID()).Msg("Token created")

	return jwt, true
}

// Providers describes the configured providers to the frontend. The URLs are
// already base-prefixed, so the login page uses them verbatim.
func (f *oauthFlow) Providers() []OAuthProviderInfo {
	infos := make([]OAuthProviderInfo, 0, len(f.providers))
	for _, p := range f.providers {
		infos = append(infos, OAuthProviderInfo{
			Name:     p.DisplayName(),
			LoginURL: fmt.Sprintf("%s/api/auth/login?provider=%s", f.base, url.QueryEscape(p.ID())),
			Icon:     p.Icon(),
		})
	}

	return infos
}

func (f *oauthFlow) provider(id string) IdentityProvider {
	// Omitting ?provider= is unambiguous when only one is configured, which is
	// the common single-vendor setup.
	if id == "" && len(f.providers) == 1 {
		return f.providers[0]
	}

	for _, p := range f.providers {
		if p.ID() == id {
			return p
		}
	}

	return nil
}

const stateCookieName = "dozzle_oauth_state"

// stateCookieTTL bounds how long a half-finished login stays resumable.
const stateCookieTTL = 10 * time.Minute

// oauthState is the CSRF state and PKCE verifier for one in-flight login. It
// rides in an HttpOnly cookie rather than server memory so the flow survives a
// restart and works across replicas, and the browser that started the login is
// the only one that can finish it.
type oauthState struct {
	Provider    string `json:"p"`
	State       string `json:"s"`
	Verifier    string `json:"v"`
	RedirectURL string `json:"r"`
	// CallbackURI is the redirect_uri sent at authorize time. OIDC requires the
	// token exchange to replay it byte for byte, and pinning it here rather than
	// recomputing it means a proxy that rewrites Host between the two legs
	// cannot break the exchange.
	CallbackURI string `json:"c"`
}

// callbackURL derives this deployment's callback from the request.
//
// A spoofed Host cannot leak a code: the provider only honours a redirect_uri
// that is on the OAuth app's registered list, so a forged value fails the login
// instead. That is what lets Dozzle work without a base-URL flag.
func (f *oauthFlow) callbackURL(r *http.Request) string {
	scheme := "http"
	if IsHTTPS(r) {
		scheme = "https"
	}

	host := r.Host
	if forwarded := r.Header.Get("X-Forwarded-Host"); forwarded != "" {
		host, _, _ = strings.Cut(forwarded, ",")
		host = strings.TrimSpace(host)
	}

	return scheme + "://" + host + f.base + "/api/auth/callback"
}

func randomString() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func (f *oauthFlow) setStateCookie(w http.ResponseWriter, r *http.Request, state oauthState) error {
	encoded, err := json.Marshal(state)
	if err != nil {
		return err
	}

	http.SetCookie(w, &http.Cookie{
		Name:     stateCookieName,
		Value:    base64.RawURLEncoding.EncodeToString(encoded),
		HttpOnly: true,
		Path:     f.base + "/",
		// The callback is a top-level navigation from the provider, which is
		// cross-site; Lax still sends the cookie on that, Strict would not.
		SameSite: http.SameSiteLaxMode,
		Secure:   IsHTTPS(r),
		MaxAge:   int(stateCookieTTL.Seconds()),
	})

	return nil
}

func (f *oauthFlow) clearStateCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     stateCookieName,
		Value:    "",
		HttpOnly: true,
		Path:     f.base + "/",
		SameSite: http.SameSiteLaxMode,
		Secure:   IsHTTPS(r),
		MaxAge:   -1,
	})
}

func (f *oauthFlow) readStateCookie(r *http.Request) (oauthState, bool) {
	cookie, err := r.Cookie(stateCookieName)
	if err != nil {
		return oauthState{}, false
	}

	raw, err := base64.RawURLEncoding.DecodeString(cookie.Value)
	if err != nil {
		return oauthState{}, false
	}

	var state oauthState
	if err := json.Unmarshal(raw, &state); err != nil {
		return oauthState{}, false
	}

	if state.State == "" || state.Provider == "" {
		return oauthState{}, false
	}

	return state, true
}

// LoginHandler starts the flow: it mints state and a PKCE verifier, parks them
// in a short-lived cookie, and bounces the browser to the provider.
func (f *oauthFlow) LoginHandler(w http.ResponseWriter, r *http.Request) {
	provider := f.provider(r.URL.Query().Get("provider"))
	if provider == nil {
		log.Warn().Str("provider", r.URL.Query().Get("provider")).Msg("Unknown OAuth provider requested")
		http.Error(w, "Unknown provider", http.StatusBadRequest)
		return
	}

	state, err := randomString()
	if err != nil {
		log.Error().Err(err).Msg("Could not generate OAuth state")
		f.failLogin(w, r)
		return
	}

	verifier := oauth2.GenerateVerifier()
	callbackURI := f.callbackURL(r)

	// Built before the cookie is set, so a provider that cannot start a login
	// leaves no half-open state behind for the browser to carry around.
	config, err := provider.oauth2Config(callbackURI)
	if err != nil {
		log.Error().Err(err).Str("provider", provider.ID()).Msg("Could not build the OAuth config")
		f.failLogin(w, r)
		return
	}

	if err := f.setStateCookie(w, r, oauthState{
		Provider:    provider.ID(),
		State:       state,
		Verifier:    verifier,
		RedirectURL: SafeRelativePath(r.URL.Query().Get("redirectUrl")),
		CallbackURI: callbackURI,
	}); err != nil {
		log.Error().Err(err).Msg("Could not set OAuth state cookie")
		f.failLogin(w, r)
		return
	}

	// GitHub returns a config with no RedirectURL, and oauth2 omits redirect_uri
	// entirely in that case, so the OAuth app's registered callback is used.
	authURL := config.AuthCodeURL(state,
		oauth2.S256ChallengeOption(verifier),
	)

	http.Redirect(w, r, authURL, http.StatusFound)
}

// CallbackHandler finishes the flow. Every failure path lands on the login page
// with ?error=oauth rather than leaking which step failed.
func (f *oauthFlow) CallbackHandler(w http.ResponseWriter, r *http.Request) {
	// The state is single use whatever happens next.
	f.clearStateCookie(w, r)

	state, ok := f.readStateCookie(r)
	if !ok {
		log.Warn().Msg("OAuth callback without a valid state cookie")
		f.failLogin(w, r)
		return
	}

	query := r.URL.Query()

	if errCode := query.Get("error"); errCode != "" {
		log.Warn().Str("error", errCode).Str("description", query.Get("error_description")).Msg("OAuth provider returned an error")
		f.failLogin(w, r)
		return
	}

	if subtle.ConstantTimeCompare([]byte(state.State), []byte(query.Get("state"))) != 1 {
		log.Warn().Msg("OAuth callback state did not match the state cookie")
		f.failLogin(w, r)
		return
	}

	provider := f.provider(state.Provider)
	if provider == nil {
		log.Warn().Str("provider", state.Provider).Msg("OAuth callback for a provider that is no longer configured")
		f.failLogin(w, r)
		return
	}

	code := query.Get("code")
	if code == "" {
		log.Warn().Msg("OAuth callback without a code")
		f.failLogin(w, r)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	config, err := provider.oauth2Config(state.CallbackURI)
	if err != nil {
		log.Error().Err(err).Str("provider", provider.ID()).Msg("Could not build the OAuth config")
		f.failLogin(w, r)
		return
	}

	token, err := config.Exchange(ctx, code, oauth2.VerifierOption(state.Verifier))
	if err != nil {
		log.Error().Err(err).Msg("Could not exchange OAuth code")
		f.failLogin(w, r)
		return
	}

	identity, err := provider.identity(ctx, token)
	if err != nil {
		log.Error().Err(err).Msg("Could not read identity from OAuth provider")
		f.failLogin(w, r)
		return
	}

	jwt, ok := f.login(provider, identity)
	if !ok {
		f.failLogin(w, r)
		return
	}

	SetSessionCookie(w, r, jwt, f.ttl)

	target := state.RedirectURL
	if target == "" {
		target = "/"
	}

	http.Redirect(w, r, f.base+target, http.StatusFound)
}

func (f *oauthFlow) failLogin(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, f.base+"/login?error=oauth", http.StatusFound)
}

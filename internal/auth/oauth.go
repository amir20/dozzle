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

// externalIdentity is what a provider tells us about whoever just signed in. It
// is deliberately not a User: users.yml is the only place a Dozzle user is
// defined, and this exists only to look one up there.
type externalIdentity struct {
	Sub   string
	Login string
	Email string
	Name  string
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
	oauth2Config(callbackURI string) *oauth2.Config
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

// oauthAuthContext is simple auth with OAuth bolted on as a second way to prove
// you are one of the users already in users.yml. It embeds the simple context so
// password login, the middleware, and per-request role resolution are untouched.
type oauthAuthContext struct {
	*simpleAuthContext
	providers []IdentityProvider
	base      string
}

// NewOAuthAuth wraps simple auth. base is the router base ("" when Dozzle is
// mounted at /), used to build the login URLs and the post-login redirect.
func NewOAuthAuth(simple *simpleAuthContext, base string, providers ...IdentityProvider) *oauthAuthContext {
	if base == "/" {
		base = ""
	}

	return &oauthAuthContext{simpleAuthContext: simple, providers: providers, base: base}
}

func (a *oauthAuthContext) byGithubLogin(login string) (User, bool) {
	return a.findByGithub(login)
}

func (a *oauthAuthContext) byVerifiedEmail(email string) (User, bool) {
	return a.findByEmail(email)
}

// Providers describes the configured providers to the frontend. The URLs are
// already base-prefixed, so the login page uses them verbatim.
func (a *oauthAuthContext) Providers() []OAuthProviderInfo {
	infos := make([]OAuthProviderInfo, 0, len(a.providers))
	for _, p := range a.providers {
		infos = append(infos, OAuthProviderInfo{
			Name:     p.DisplayName(),
			LoginURL: fmt.Sprintf("%s/api/auth/login?provider=%s", a.base, url.QueryEscape(p.ID())),
			Icon:     p.Icon(),
		})
	}

	return infos
}

func (a *oauthAuthContext) provider(id string) IdentityProvider {
	// Omitting ?provider= is unambiguous when only one is configured, which is
	// the common single-vendor setup.
	if id == "" && len(a.providers) == 1 {
		return a.providers[0]
	}

	for _, p := range a.providers {
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
func (a *oauthAuthContext) callbackURL(r *http.Request) string {
	scheme := "http"
	if IsHTTPS(r) {
		scheme = "https"
	}

	host := r.Host
	if forwarded := r.Header.Get("X-Forwarded-Host"); forwarded != "" {
		host, _, _ = strings.Cut(forwarded, ",")
		host = strings.TrimSpace(host)
	}

	return scheme + "://" + host + a.base + "/api/auth/callback"
}

func randomString() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func (a *oauthAuthContext) setStateCookie(w http.ResponseWriter, r *http.Request, state oauthState) error {
	encoded, err := json.Marshal(state)
	if err != nil {
		return err
	}

	http.SetCookie(w, &http.Cookie{
		Name:     stateCookieName,
		Value:    base64.RawURLEncoding.EncodeToString(encoded),
		HttpOnly: true,
		Path:     a.base + "/",
		// The callback is a top-level navigation from the provider, which is
		// cross-site; Lax still sends the cookie on that, Strict would not.
		SameSite: http.SameSiteLaxMode,
		Secure:   IsHTTPS(r),
		MaxAge:   int(stateCookieTTL.Seconds()),
	})

	return nil
}

func (a *oauthAuthContext) clearStateCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     stateCookieName,
		Value:    "",
		HttpOnly: true,
		Path:     a.base + "/",
		SameSite: http.SameSiteLaxMode,
		Secure:   IsHTTPS(r),
		MaxAge:   -1,
	})
}

func (a *oauthAuthContext) readStateCookie(r *http.Request) (oauthState, bool) {
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
func (a *oauthAuthContext) LoginHandler(w http.ResponseWriter, r *http.Request) {
	provider := a.provider(r.URL.Query().Get("provider"))
	if provider == nil {
		log.Warn().Str("provider", r.URL.Query().Get("provider")).Msg("Unknown OAuth provider requested")
		http.Error(w, "Unknown provider", http.StatusBadRequest)
		return
	}

	state, err := randomString()
	if err != nil {
		log.Error().Err(err).Msg("Could not generate OAuth state")
		a.failLogin(w, r)
		return
	}

	verifier := oauth2.GenerateVerifier()
	callbackURI := a.callbackURL(r)

	if err := a.setStateCookie(w, r, oauthState{
		Provider:    provider.ID(),
		State:       state,
		Verifier:    verifier,
		RedirectURL: SafeRelativePath(r.URL.Query().Get("redirectUrl")),
		CallbackURI: callbackURI,
	}); err != nil {
		log.Error().Err(err).Msg("Could not set OAuth state cookie")
		a.failLogin(w, r)
		return
	}

	config := provider.oauth2Config(callbackURI)
	// GitHub returns a config with no RedirectURL, and oauth2 omits redirect_uri
	// entirely in that case, so the OAuth app's registered callback is used.
	authURL := config.AuthCodeURL(state,
		oauth2.S256ChallengeOption(verifier),
	)

	http.Redirect(w, r, authURL, http.StatusFound)
}

// CallbackHandler finishes the flow. Every failure path lands on the login page
// with ?error=oauth rather than leaking which step failed.
func (a *oauthAuthContext) CallbackHandler(w http.ResponseWriter, r *http.Request) {
	// The state is single use whatever happens next.
	a.clearStateCookie(w, r)

	state, ok := a.readStateCookie(r)
	if !ok {
		log.Warn().Msg("OAuth callback without a valid state cookie")
		a.failLogin(w, r)
		return
	}

	query := r.URL.Query()

	if errCode := query.Get("error"); errCode != "" {
		log.Warn().Str("error", errCode).Str("description", query.Get("error_description")).Msg("OAuth provider returned an error")
		a.failLogin(w, r)
		return
	}

	if subtle.ConstantTimeCompare([]byte(state.State), []byte(query.Get("state"))) != 1 {
		log.Warn().Msg("OAuth callback state did not match the state cookie")
		a.failLogin(w, r)
		return
	}

	provider := a.provider(state.Provider)
	if provider == nil {
		log.Warn().Str("provider", state.Provider).Msg("OAuth callback for a provider that is no longer configured")
		a.failLogin(w, r)
		return
	}

	code := query.Get("code")
	if code == "" {
		log.Warn().Msg("OAuth callback without a code")
		a.failLogin(w, r)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	token, err := provider.oauth2Config(state.CallbackURI).Exchange(ctx, code, oauth2.VerifierOption(state.Verifier))
	if err != nil {
		log.Error().Err(err).Msg("Could not exchange OAuth code")
		a.failLogin(w, r)
		return
	}

	identity, err := provider.identity(ctx, token)
	if err != nil {
		log.Error().Err(err).Msg("Could not read identity from OAuth provider")
		a.failLogin(w, r)
		return
	}

	// users.yml is the allowlist. No match means no login, and no account is
	// ever created here.
	user, ok := provider.match(a, identity)
	if !ok {
		log.Warn().
			Str("provider", provider.ID()).
			Str("login", identity.Login).
			Msg("OAuth login rejected: no user in the user database is linked to this account")
		a.failLogin(w, r)
		return
	}

	jwt, err := a.issueToken(user)
	if err != nil {
		log.Error().Err(err).Msg("Could not create token after OAuth login")
		a.failLogin(w, r)
		return
	}

	SetSessionCookie(w, r, jwt, a.ttl)
	log.Info().Str("user", user.Username).Str("provider", provider.ID()).Msg("Token created")

	target := state.RedirectURL
	if target == "" {
		target = "/"
	}

	http.Redirect(w, r, a.base+target, http.StatusFound)
}

func (a *oauthAuthContext) failLogin(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, a.base+"/login?error=oauth", http.StatusFound)
}

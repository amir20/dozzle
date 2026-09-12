package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/go-chi/jwtauth/v5"
	"github.com/rs/zerolog/log"
)

// OIDCConfig is everything --auth-provider oidc needs. Unlike simple auth with
// OIDC bolted on, there is no users.yml behind it: the token is the user
// database, so the config also says where in the token roles and filters live.
type OIDCConfig struct {
	Issuer       string
	ClientID     string
	ClientSecret string
	// DisplayName is the label on the login button.
	DisplayName string
	// RolesClaim and FiltersClaim are dot-separated paths that replace the
	// default search with the one path given. Empty means search the defaults.
	RolesClaim   string
	FiltersClaim string
}

// oidcAuthContext is the oidc provider: the IdP proves who you are and also
// says what you may do. Nothing about a user is read from disk.
type oidcAuthContext struct {
	*oauthFlow
	provider  *oidcProvider
	tokenAuth *jwtauth.JWTAuth
	ttl       time.Duration

	rolesClaims   []claimPath
	filtersClaims []claimPath
}

// ErrPasswordLoginUnavailable is what CreateToken returns under the oidc
// provider. The route is not registered there, so it is only reachable from
// code, but the Authorizer interface still has to answer.
var ErrPasswordLoginUnavailable = errors.New("password login is not available with the oidc auth provider")

// NewOIDCAuth builds the oidc provider. base is the router base, secret is the
// persisted key from SessionSecret.
func NewOIDCAuth(config OIDCConfig, base string, ttl time.Duration, secret []byte) *oidcAuthContext {
	provider := NewOIDCProvider(config.Issuer, config.ClientID, config.ClientSecret, config.DisplayName)
	// sub is the key here and the email is only for display, so an issuer that
	// grants no email scope, or one that never marks addresses verified, is fine.
	provider.requireVerifiedEmail = false

	// The issuer and client id are hashed in with the secret so that repointing
	// Dozzle at a different IdP invalidates every session minted under the old
	// one: the same sub at a different issuer is a different person.
	h := sha256.New()
	h.Write(secret)
	h.Write([]byte(provider.issuer))
	h.Write([]byte(config.ClientID))

	a := &oidcAuthContext{
		provider:      provider,
		tokenAuth:     jwtauth.New("HS256", h.Sum(nil), nil),
		ttl:           ttl,
		rolesClaims:   claimSearch(config.RolesClaim, "roles", config.ClientID),
		filtersClaims: claimSearch(config.FiltersClaim, "filters", config.ClientID),
	}
	a.oauthFlow = newOAuthFlow(base, ttl, a.loginUser, provider)

	return a
}

// claimSearch is the list of paths tried for a claim, in order. An explicit
// path replaces the search outright.
//
// groups is deliberately absent from the defaults. At Authentik or Google every
// user has one, so including it would turn "denied" into "signed in and can
// read every container".
func claimSearch(explicit, name, clientID string) []claimPath {
	if path := parseClaimPath(explicit); len(path) > 0 {
		return []claimPath{path}
	}

	return []claimPath{
		{"dozzle_" + name},
		{"resource_access", clientID, name},
		{name},
	}
}

// RolesClaims describes the paths roles are read from, for the startup log.
func (a *oidcAuthContext) RolesClaims() string {
	return describePaths(a.rolesClaims)
}

func describePaths(paths []claimPath) string {
	names := make([]string, 0, len(paths))
	for _, path := range paths {
		names = append(names, path.String())
	}

	return strings.Join(names, ", ")
}

// PasswordLoginEnabled is always false: there is no password to check against.
func (a *oidcAuthContext) PasswordLoginEnabled() bool { return false }

func (a *oidcAuthContext) CreateToken(username, password string) (string, error) {
	return "", ErrPasswordLoginUnavailable
}

// loginUser is where the token becomes a user. A token with no roles claim, or
// an empty one, is refused: the IdP has not said this person may use Dozzle.
// A claim that is present but names no Dozzle role signs in with no
// privileges, the same as roles: none in users.yml.
func (a *oidcAuthContext) loginUser(provider IdentityProvider, identity externalIdentity) (string, bool) {
	if identity.Sub == "" {
		log.Warn().Msg("OIDC login rejected: the issuer returned no sub claim")
		return "", false
	}

	roles, rolesPath, ok := identity.Claims.strings(a.rolesClaims)
	if !ok || len(roles) == 0 {
		log.Warn().
			Str("sub", identity.Sub).
			Str("login", identity.Login).
			Str("tried", describePaths(a.rolesClaims)).
			Msg("OIDC login rejected: no roles claim found in the ID token or userinfo, or it was empty")
		return "", false
	}

	var filters []string
	if values, path, ok := identity.Claims.strings(a.filtersClaims); ok {
		filters = values
		log.Debug().Str("claim", path.String()).Strs("filters", filters).Msg("Resolved OIDC filters claim")
	}

	rolesRaw := strings.Join(roles, ",")
	filtersRaw := strings.Join(filters, ",")

	// Validated here so a typo in the IdP fails the login loudly rather than
	// every request quietly.
	if _, err := container.ParseContainerFilter(filtersRaw); err != nil {
		log.Warn().Err(err).Str("sub", identity.Sub).Str("filters", filtersRaw).Msg("OIDC login rejected: the filters claim is not a valid container filter")
		return "", false
	}

	user := a.newUser(identity.Sub, identity.Login, identity.Email, identity.Name, identity.Picture, rolesRaw, filtersRaw)

	// Roles and filters ride in the session as the raw claim strings and are
	// parsed again on every request. There is no users.yml to re-read, so the
	// session is the only copy; a change at the IdP lands at the next login.
	claims := map[string]any{
		"sub":      identity.Sub,
		"username": identity.Login,
		"email":    identity.Email,
		"name":     identity.Name,
		"picture":  identity.Picture,
		"roles":    rolesRaw,
		"filters":  filtersRaw,
	}
	jwtauth.SetIssuedNow(claims)
	if a.ttl > 0 {
		jwtauth.SetExpiryIn(claims, a.ttl)
	}

	_, token, err := a.tokenAuth.Encode(claims)
	if err != nil {
		log.Error().Err(err).Msg("Could not create token after OIDC login")
		return "", false
	}

	log.Info().
		Str("user", user.Username).
		Str("login", identity.Login).
		Str("claim", rolesPath.String()).
		Strs("roles", roles).
		Msg("Token created")

	return token, true
}

// newUser builds the request-scoped user from what the session carries.
//
// sub keys the user, and with it the profile directory, because it is the one
// claim the spec guarantees stable and unique. preferred_username is not: an
// IdP may let it change or be reused. The display name falls back through
// preferred_username and email so the menu never shows a bare UUID unless the
// issuer gave nothing else.
func (a *oidcAuthContext) newUser(sub, login, email, name, picture, rolesRaw, filtersRaw string) User {
	if name == "" {
		name = login
	}
	if name == "" {
		name = email
	}
	if name == "" {
		name = sub
	}

	labels, _ := container.ParseContainerFilter(filtersRaw)

	user := newUser(profileKey(sub), email, name, labels, ParseRole(rolesRaw))
	user.Picture = picture

	return user
}

// profileKey is the directory name under /data for a subject. sub is opaque
// and some issuers put a slash in it, which the profile store rightly refuses
// as a path, so such a value is hashed instead of written as is.
func profileKey(sub string) string {
	if filepath.Base(sub) == sub && sub != "." && sub != ".." {
		return sub
	}

	sum := sha256.Sum256([]byte(sub))

	return "oidc-" + hex.EncodeToString(sum[:8])
}

func (a *oidcAuthContext) AuthMiddleware(next http.Handler) http.Handler {
	return jwtauth.Verifier(a.tokenAuth)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if user := a.userFromToken(r.Context()); user != nil {
			r = r.WithContext(WithUser(r.Context(), *user))
		}

		next.ServeHTTP(w, r)
	}))
}

// userFromToken rebuilds the user from the verified session. It returns nil for
// a missing or invalid token so the request falls through to
// RequireAuthentication as unauthenticated.
func (a *oidcAuthContext) userFromToken(ctx context.Context) *User {
	_, claims, err := jwtauth.FromContext(ctx)
	if err != nil {
		return nil
	}

	str := func(key string) string {
		s, _ := claims[key].(string)
		return s
	}

	sub := str("sub")
	if sub == "" {
		return nil
	}

	user := a.newUser(sub, str("username"), str("email"), str("name"), str("picture"), str("roles"), str("filters"))

	return &user
}

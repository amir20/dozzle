package auth

import (
	"context"
	"crypto/sha256"
	"net/http"
	"strings"

	"github.com/go-chi/jwtauth/v5"
)

// TokenUseClaim marks a JWT that is signed with the session key but is not a
// session. Only tokens without it are accepted as a browser session, so an MCP
// access token cannot be replayed against the rest of the API.
const TokenUseClaim = "token_use"

// TokenAuthorizer is an authorizer whose sessions are JWTs it signs itself. The
// MCP authorization server mints its tokens through it, so they are signed with
// the session key and are invalidated by whatever rotates it.
type TokenAuthorizer interface {
	// SignToken signs claims as they are. The caller sets exp and TokenUseClaim.
	SignToken(claims map[string]any) (string, error)
	// VerifyToken checks the signature and expiry and returns the claims.
	VerifyToken(token string) (map[string]any, error)
	// UserFromClaims rebuilds the user the way the session middleware does, or
	// returns nil when the claims no longer name one.
	UserFromClaims(claims map[string]any) *User
	// DerivedKey is a key for purpose that depends only on the persisted session
	// secret, for things that must survive users.yml or issuer changes.
	DerivedKey(purpose string) []byte
}

func isSessionClaims(claims map[string]any) bool {
	_, ok := claims[TokenUseClaim]
	return !ok
}

func verifyClaims(ja *jwtauth.JWTAuth, token string) (map[string]any, error) {
	t, err := jwtauth.VerifyToken(ja, token)
	if err != nil {
		return nil, err
	}

	_, claims, err := jwtauth.FromContext(jwtauth.NewContext(context.Background(), t, nil))

	return claims, err
}

func deriveKey(secret []byte, purpose string) []byte {
	h := sha256.New()
	h.Write(secret)
	h.Write([]byte("\x00" + purpose))

	return h.Sum(nil)
}

// Origin is scheme://host as the browser reached this request, honouring
// X-Forwarded-Proto and X-Forwarded-Host from a reverse proxy.
func Origin(r *http.Request) string {
	scheme := "http"
	if IsHTTPS(r) {
		scheme = "https"
	}

	host := r.Host
	if forwarded := r.Header.Get("X-Forwarded-Host"); forwarded != "" {
		host, _, _ = strings.Cut(forwarded, ",")
		host = strings.TrimSpace(host)
	}

	return scheme + "://" + host
}

var (
	_ TokenAuthorizer = (*simpleAuthContext)(nil)
	_ TokenAuthorizer = (*oauthAuthContext)(nil)
	_ TokenAuthorizer = (*oidcAuthContext)(nil)
)

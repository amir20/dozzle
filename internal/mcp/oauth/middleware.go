package oauth

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/amir20/dozzle/internal/auth"
	mcpauth "github.com/modelcontextprotocol/go-sdk/auth"
)

// Middleware guards the MCP endpoint. A request that already carries a browser
// session, from the cookie or a session JWT sent as a Bearer token the way the
// docs used to describe, passes as before. Anything else needs an access token
// from this server, and a request without one gets the 401 whose
// WWW-Authenticate header starts an MCP client's login.
func (s *Server) Middleware(next http.Handler) http.Handler {
	withUser := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if info := mcpauth.TokenInfoFromContext(r.Context()); info != nil {
			if user, ok := info.Extra["user"].(auth.User); ok {
				r = r.WithContext(auth.WithUser(r.Context(), user))
			}
		}
		next.ServeHTTP(w, r)
	})

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if auth.UserFromContext(r.Context()) != nil {
			next.ServeHTTP(w, r)
			return
		}

		mcpauth.RequireBearerToken(s.verifyAccessToken, &mcpauth.RequireBearerTokenOptions{
			ResourceMetadataURL: s.ResourceMetadataURL(r),
			Scopes:              []string{Scope},
		})(withUser).ServeHTTP(w, r)
	})
}

func (s *Server) verifyAccessToken(_ context.Context, token string, r *http.Request) (*mcpauth.TokenInfo, error) {
	claims, err := s.tokens.VerifyToken(token)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", mcpauth.ErrInvalidToken, err)
	}
	if claims[auth.TokenUseClaim] != tokenUseAccess {
		return nil, fmt.Errorf("%w: not an MCP access token", mcpauth.ErrInvalidToken)
	}
	// Audience binding: a token issued for this server under another hostname,
	// or for a different resource, is not accepted here.
	if !slices.Contains(stringList(claims["aud"]), s.resource(r)) {
		return nil, fmt.Errorf("%w: token audience does not match %s", mcpauth.ErrInvalidToken, s.resource(r))
	}

	user := s.tokens.UserFromClaims(claims)
	if user == nil {
		return nil, fmt.Errorf("%w: the user is no longer allowed to sign in", mcpauth.ErrInvalidToken)
	}

	expiry, _ := unixClaim(claims["exp"])
	scope, _ := claims["scope"].(string)

	return &mcpauth.TokenInfo{
		Scopes:     strings.Fields(scope),
		Expiration: expiry,
		UserID:     user.Username,
		Extra:      map[string]any{"user": *user},
	}, nil
}

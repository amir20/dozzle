package oauth

import (
	"context"
	"encoding/json"
	"maps"
	"net/http"
	"slices"
	"time"

	"github.com/go-chi/jwtauth/v5"
	"github.com/rs/zerolog/log"
)

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

// Token is the token endpoint: authorization_code and refresh_token grants for
// public clients. POST /api/oauth/token.
func (s *Server) Token(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")

	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	if err := r.ParseForm(); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "could not parse the form")
		return
	}

	c, ok := s.client(r.PostForm.Get("client_id"))
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid_client", "unknown client_id")
		return
	}

	switch r.PostForm.Get("grant_type") {
	case "authorization_code":
		s.exchangeCode(w, r, c)
	case "refresh_token":
		s.refresh(w, r, c)
	default:
		writeError(w, http.StatusBadRequest, "unsupported_grant_type", "")
	}
}

func (s *Server) exchangeCode(w http.ResponseWriter, r *http.Request, c client) {
	code, ok := s.takeCode(r.PostForm.Get("code"))
	if !ok || code.clientID != c.ID || code.redirectURI != r.PostForm.Get("redirect_uri") {
		writeError(w, http.StatusBadRequest, "invalid_grant", "the code is invalid, expired, or was issued to another client")
		return
	}
	if !verifyPKCE(r.PostForm.Get("code_verifier"), code.challenge) {
		writeError(w, http.StatusBadRequest, "invalid_grant", "code_verifier does not match the code_challenge")
		return
	}
	if resource := r.PostForm.Get("resource"); resource != "" && resource != code.resource {
		writeError(w, http.StatusBadRequest, "invalid_target", "")
		return
	}

	s.issue(w, code.identity, c.ID, code.resource, s.now())
}

func (s *Server) refresh(w http.ResponseWriter, r *http.Request, c client) {
	claims, err := s.tokens.VerifyToken(r.PostForm.Get("refresh_token"))
	if err != nil || claims["token_use"] != tokenUseRefresh || claims["client_id"] != c.ID {
		writeError(w, http.StatusBadRequest, "invalid_grant", "the refresh token is invalid or expired")
		return
	}

	audience := stringList(claims["aud"])
	if len(audience) != 1 {
		writeError(w, http.StatusBadRequest, "invalid_grant", "")
		return
	}
	if resource := r.PostForm.Get("resource"); resource != "" && resource != audience[0] {
		writeError(w, http.StatusBadRequest, "invalid_target", "")
		return
	}

	authTime, ok := unixClaim(claims["auth_time"])
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid_grant", "")
		return
	}

	s.issue(w, identityClaims(claims), c.ID, audience[0], authTime)
}

// issue mints a new access and refresh token pair. authTime is when the user
// approved the client, carried across refreshes so the refresh token's absolute
// lifetime never moves.
func (s *Server) issue(w http.ResponseWriter, identity map[string]any, clientID, resource string, authTime time.Time) {
	// A user removed from users.yml since, or a session that no longer names
	// anyone, gets nothing.
	if s.tokens.UserFromClaims(identity) == nil {
		writeError(w, http.StatusBadRequest, "invalid_grant", "the user is no longer allowed to sign in")
		return
	}

	now := s.now()
	refreshExpiry := authTime.Add(refreshTokenTTL)
	if !now.Before(refreshExpiry) {
		writeError(w, http.StatusBadRequest, "invalid_grant", "the grant has expired, approve the client again")
		return
	}

	mint := func(use string, expiry time.Time, extra map[string]any) (string, error) {
		claims := make(map[string]any, len(identity)+len(extra)+6)
		maps.Copy(claims, identity)
		maps.Copy(claims, extra)
		claims["token_use"] = use
		claims["aud"] = resource
		claims["client_id"] = clientID
		claims["scope"] = Scope
		jwtauth.SetIssuedAt(claims, now)
		jwtauth.SetExpiry(claims, expiry)

		return s.tokens.SignToken(claims)
	}

	accessExpiry := now.Add(accessTokenTTL)
	access, err := mint(tokenUseAccess, accessExpiry, nil)
	if err != nil {
		log.Error().Err(err).Msg("Could not sign an MCP access token")
		writeError(w, http.StatusInternalServerError, "server_error", "")
		return
	}

	refresh, err := mint(tokenUseRefresh, refreshExpiry, map[string]any{"auth_time": authTime.Unix()})
	if err != nil {
		log.Error().Err(err).Msg("Could not sign an MCP refresh token")
		writeError(w, http.StatusInternalServerError, "server_error", "")
		return
	}

	writeJSON(w, http.StatusOK, tokenResponse{
		AccessToken:  access,
		TokenType:    "Bearer",
		ExpiresIn:    int64(accessExpiry.Sub(now).Seconds()),
		RefreshToken: refresh,
		Scope:        Scope,
	})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, code, description string) {
	body := map[string]string{"error": code}
	if description != "" {
		body["error_description"] = description
	}
	writeJSON(w, status, body)
}

func verifyWith(ja *jwtauth.JWTAuth, token string) (map[string]any, error) {
	t, err := jwtauth.VerifyToken(ja, token)
	if err != nil {
		return nil, err
	}
	_, claims, err := jwtauth.FromContext(jwtauth.NewContext(context.Background(), t, nil))

	return claims, err
}

// stringList reads a claim that is a string or a list of them, which is how
// aud and other array claims come back depending on how many values they hold.
func stringList(v any) []string {
	switch v := v.(type) {
	case string:
		return []string{v}
	case []string:
		return slices.Clone(v)
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}

	return nil
}

func unixClaim(v any) (time.Time, bool) {
	switch v := v.(type) {
	case float64:
		return time.Unix(int64(v), 0), true
	case int64:
		return time.Unix(v, 0), true
	case int:
		return time.Unix(int64(v), 0), true
	case json.Number:
		n, err := v.Int64()
		return time.Unix(n, 0), err == nil
	case time.Time:
		return v, true
	}

	return time.Time{}, false
}

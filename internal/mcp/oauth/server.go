// Package oauth is the OAuth 2.1 authorization server MCP clients sign in
// through.
//
// Dozzle is its own authorization server rather than pointing clients at the
// configured IdP. A client registers itself here, the user approves it on a
// consent page after signing in however the instance is configured (password,
// GitHub or OIDC), and the token that comes back is a Dozzle JWT. That keeps the
// MCP spec's requirements (dynamic registration, PKCE, audience-bound tokens)
// off the IdP, most of which support only some of them.
package oauth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"sync"
	"time"

	"github.com/amir20/dozzle/internal/auth"
	"github.com/go-chi/jwtauth/v5"
)

const (
	// Scope is the one scope there is: read access through the MCP tools.
	Scope = "mcp"

	accessTokenTTL = time.Hour
	// refreshTokenTTL is absolute from the original consent, not sliding, so a
	// client is asked to be approved again at least this often. Under oidc the
	// roles in the token are the ones the IdP gave at that login.
	refreshTokenTTL = 30 * 24 * time.Hour
	codeTTL         = time.Minute

	tokenUseAccess  = "mcp_access"
	tokenUseRefresh = "mcp_refresh"
	tokenUseClient  = "mcp_client"
)

// Server is the authorization server. It is stateless apart from the
// authorization codes, which live for a minute.
type Server struct {
	tokens auth.TokenAuthorizer
	// base is the router base, "" when Dozzle is mounted at /.
	base string
	// clients signs client ids. Registration is stateless: the client id is a
	// signed copy of what the client registered, so nothing is stored and there
	// is nothing to fill up. Its key depends only on the session secret, so a
	// users.yml edit does not orphan every registered client.
	clients *jwtauth.JWTAuth

	mu    sync.Mutex
	codes map[string]pendingCode

	now func() time.Time
}

type pendingCode struct {
	identity    map[string]any
	clientID    string
	redirectURI string
	challenge   string
	resource    string
	expires     time.Time
}

func New(tokens auth.TokenAuthorizer, base string) *Server {
	if base == "/" {
		base = ""
	}

	return &Server{
		tokens:  tokens,
		base:    base,
		clients: jwtauth.New("HS256", tokens.DerivedKey("mcp-oauth-clients"), nil),
		codes:   make(map[string]pendingCode),
		now:     time.Now,
	}
}

func (s *Server) saveCode(code pendingCode) (string, error) {
	id, err := randomString()
	if err != nil {
		return "", err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now()
	for k, c := range s.codes {
		if now.After(c.expires) {
			delete(s.codes, k)
		}
	}
	s.codes[id] = code

	return id, nil
}

// takeCode is single use whether or not the exchange that follows succeeds.
func (s *Server) takeCode(id string) (pendingCode, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	code, ok := s.codes[id]
	delete(s.codes, id)
	if !ok || s.now().After(code.expires) {
		return pendingCode{}, false
	}

	return code, true
}

func verifyPKCE(verifier, challenge string) bool {
	if verifier == "" {
		return false
	}
	sum := sha256.Sum256([]byte(verifier))

	return subtle.ConstantTimeCompare([]byte(base64.RawURLEncoding.EncodeToString(sum[:])), []byte(challenge)) == 1
}

func randomString() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(buf), nil
}

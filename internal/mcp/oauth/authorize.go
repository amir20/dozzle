package oauth

import (
	"encoding/json"
	"maps"
	"mime"
	"net/http"
	"net/url"

	"github.com/amir20/dozzle/internal/auth"
	"github.com/go-chi/jwtauth/v5"
	"github.com/rs/zerolog/log"
)

// The authorization endpoint the metadata advertises is the SPA's consent page
// (/mcp/authorize). The page is served like any other route, so an unsigned-in
// visitor goes through the normal login and lands back on it with the query
// intact. The page then talks to the two handlers below.

type authorizeRequest struct {
	ResponseType        string
	ClientID            string
	RedirectURI         string
	CodeChallenge       string
	CodeChallengeMethod string
	State               string
	Resource            string
}

func parseAuthorizeRequest(query string) authorizeRequest {
	v, _ := url.ParseQuery(query)

	return authorizeRequest{
		ResponseType:        v.Get("response_type"),
		ClientID:            v.Get("client_id"),
		RedirectURI:         v.Get("redirect_uri"),
		CodeChallenge:       v.Get("code_challenge"),
		CodeChallengeMethod: v.Get("code_challenge_method"),
		State:               v.Get("state"),
		Resource:            v.Get("resource"),
	}
}

// authorizeError is what the consent page shows or follows. RedirectURL is set
// only once the client and redirect_uri are known to be good: before that there
// is nowhere safe to send the browser (RFC 6749 4.1.2.1).
type authorizeError struct {
	Error       string `json:"error"`
	Description string `json:"description"`
	RedirectURL string `json:"redirectUrl,omitempty"`
}

// validate checks an authorization request and returns the client, or the
// error to show.
func (s *Server) validate(r *http.Request, req authorizeRequest) (client, *authorizeError) {
	c, ok := s.client(req.ClientID)
	if !ok {
		return client{}, &authorizeError{Error: "invalid_client", Description: "This client is not registered with this Dozzle instance."}
	}
	if req.RedirectURI == "" || !c.allows(req.RedirectURI) {
		return client{}, &authorizeError{Error: "invalid_request", Description: "The redirect_uri was not registered by this client."}
	}

	fail := func(code, description string) *authorizeError {
		return &authorizeError{Error: code, Description: description, RedirectURL: s.redirect(r, req, url.Values{"error": {code}, "error_description": {description}})}
	}

	if req.ResponseType != "code" {
		return client{}, fail("unsupported_response_type", "only response_type=code is supported")
	}
	if req.CodeChallenge == "" || req.CodeChallengeMethod != "S256" {
		return client{}, fail("invalid_request", "PKCE with code_challenge_method=S256 is required")
	}
	if req.Resource != "" && req.Resource != s.resource(r) {
		return client{}, fail("invalid_target", "resource must be "+s.resource(r))
	}

	return c, nil
}

// redirect builds the response the client is sent back with. iss is RFC 9207,
// which the metadata advertises, so a client can tell this server's response
// from a mix-up.
func (s *Server) redirect(r *http.Request, req authorizeRequest, params url.Values) string {
	u, err := url.Parse(req.RedirectURI)
	if err != nil {
		return ""
	}

	q := u.Query()
	maps.Copy(q, params)
	if req.State != "" {
		q.Set("state", req.State)
	}
	q.Set("iss", s.issuer(r))
	u.RawQuery = q.Encode()

	return u.String()
}

type consentInfo struct {
	ClientName   string `json:"clientName"`
	RedirectHost string `json:"redirectHost"`
	Scope        string `json:"scope"`
}

// DescribeRequest validates the query the consent page was opened with and says
// what to show. GET /api/oauth/authorize?<the client's query>.
func (s *Server) DescribeRequest(w http.ResponseWriter, r *http.Request) {
	req := parseAuthorizeRequest(r.URL.RawQuery)
	c, failure := s.validate(r, req)
	if failure != nil {
		writeJSON(w, http.StatusBadRequest, failure)
		return
	}

	// Only an https host is a claim about who receives the code. Any other
	// scheme is shown in full: evilapp://claude.ai is not claude.ai.
	host := req.RedirectURI
	if u, err := url.Parse(req.RedirectURI); err == nil {
		switch {
		case u.Scheme == "https":
			host = u.Host
		case u.Host != "":
			host = u.Scheme + "://" + u.Host
		default:
			host = u.Scheme + ":"
		}
	}

	writeJSON(w, http.StatusOK, consentInfo{ClientName: c.Name, RedirectHost: host, Scope: Scope})
}

type decision struct {
	// Query is the authorization request exactly as the consent page got it.
	Query   string `json:"query"`
	Approve bool   `json:"approve"`
}

// Decide records the user's answer and returns where to send the browser.
// POST /api/oauth/authorize. It is behind the session like the rest of the API,
// and a JSON body under a SameSite=Lax cookie cannot be forged cross-site.
func (s *Server) Decide(w http.ResponseWriter, r *http.Request) {
	// A text/plain form post is the one cross-origin POST a browser sends
	// without a preflight, so insisting on JSON keeps that door shut too.
	if mediaType, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type")); mediaType != "application/json" {
		http.Error(w, "expected application/json", http.StatusUnsupportedMediaType)
		return
	}

	var d decision
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&d); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	req := parseAuthorizeRequest(d.Query)
	c, failure := s.validate(r, req)
	if failure != nil {
		writeJSON(w, http.StatusBadRequest, failure)
		return
	}

	if !d.Approve {
		writeJSON(w, http.StatusOK, map[string]string{"redirectUrl": s.redirect(r, req, url.Values{"error": {"access_denied"}})})
		return
	}

	user := auth.UserFromContext(r.Context())
	_, claims, err := jwtauth.FromContext(r.Context())
	if user == nil || err != nil || claims[auth.TokenUseClaim] != nil {
		http.Error(w, "a browser session is required to approve a client", http.StatusUnauthorized)
		return
	}

	resource := req.Resource
	if resource == "" {
		resource = s.resource(r)
	}

	code, err := s.saveCode(pendingCode{
		identity:    identityClaims(claims),
		clientID:    c.ID,
		redirectURI: req.RedirectURI,
		challenge:   req.CodeChallenge,
		resource:    resource,
		expires:     s.now().Add(codeTTL),
	})
	if err != nil {
		log.Error().Err(err).Msg("Could not create an MCP authorization code")
		http.Error(w, "could not create an authorization code", http.StatusInternalServerError)
		return
	}

	log.Info().Str("user", user.Username).Str("client", c.Name).Str("redirect", req.RedirectURI).Msg("MCP client approved")

	writeJSON(w, http.StatusOK, map[string]string{"redirectUrl": s.redirect(r, req, url.Values{"code": {code}})})
}

// identityClaims is the session minus everything about the session token
// itself. It is what the user is rebuilt from on every MCP request, exactly as
// the session middleware would rebuild them.
func identityClaims(claims map[string]any) map[string]any {
	identity := make(map[string]any, len(claims))
	for k, v := range claims {
		switch k {
		// session names the browser session's stored ID token under oidc, which
		// an MCP token has no business reaching.
		case "iat", "exp", "nbf", "aud", "iss", "jti", auth.TokenUseClaim, "client_id", "scope", "auth_time", "session":
			continue
		}
		identity[k] = v
	}

	return identity
}

package oauth

import (
	"encoding/json"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"unicode"

	"github.com/go-chi/jwtauth/v5"
)

type client struct {
	ID           string
	Name         string
	RedirectURIs []string
}

type registrationRequest struct {
	RedirectURIs []string `json:"redirect_uris"`
	ClientName   string   `json:"client_name"`
}

type registrationResponse struct {
	ClientID                string   `json:"client_id"`
	ClientIDIssuedAt        int64    `json:"client_id_issued_at"`
	ClientName              string   `json:"client_name,omitempty"`
	RedirectURIs            []string `json:"redirect_uris"`
	GrantTypes              []string `json:"grant_types"`
	ResponseTypes           []string `json:"response_types"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
	Scope                   string   `json:"scope"`
}

const (
	maxRedirectURIs  = 10
	maxClientNameLen = 100
)

// Register is dynamic client registration (RFC 7591). It is open, as the MCP
// spec expects: a registered client can do nothing until a signed-in user
// approves it on the consent page, which is what names where the code goes.
//
// Every client is public. Whatever auth method was asked for, the response says
// "none", which RFC 7591 lets the server decide.
func (s *Server) Register(w http.ResponseWriter, r *http.Request) {
	var req registrationRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_client_metadata", "request body is not valid JSON")
		return
	}

	if len(req.RedirectURIs) == 0 || len(req.RedirectURIs) > maxRedirectURIs {
		writeError(w, http.StatusBadRequest, "invalid_redirect_uri", "between 1 and 10 redirect_uris are required")
		return
	}
	for _, uri := range req.RedirectURIs {
		if !validRedirectURI(uri) {
			writeError(w, http.StatusBadRequest, "invalid_redirect_uri", "redirect_uri must be https, http on a loopback address, or a private-use scheme: "+uri)
			return
		}
	}

	// jti keeps two registrations with the same metadata apart, so a refresh
	// token bound to one is not accepted from the other.
	jti, err := randomString()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", "")
		return
	}

	name := cleanClientName(req.ClientName)
	now := s.now()
	claims := map[string]any{
		"jti":           jti,
		"token_use":     tokenUseClient,
		"client_name":   name,
		"redirect_uris": req.RedirectURIs,
	}
	jwtauth.SetIssuedAt(claims, now)

	_, id, err := s.clients.Encode(claims)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", "")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(registrationResponse{
		ClientID:                id,
		ClientIDIssuedAt:        now.Unix(),
		ClientName:              name,
		RedirectURIs:            req.RedirectURIs,
		GrantTypes:              []string{"authorization_code", "refresh_token"},
		ResponseTypes:           []string{"code"},
		TokenEndpointAuthMethod: "none",
		Scope:                   Scope,
	})
}

func (s *Server) client(id string) (client, bool) {
	if id == "" {
		return client{}, false
	}

	claims, err := verifyWith(s.clients, id)
	if err != nil || claims["token_use"] != tokenUseClient {
		return client{}, false
	}

	name, _ := claims["client_name"].(string)

	return client{ID: id, Name: name, RedirectURIs: stringList(claims["redirect_uris"])}, true
}

// allows reports whether redirectURI is one the client registered. Loopback
// redirects match on any port (RFC 8252 7.3): native clients bind an ephemeral
// port per login and cannot know it at registration.
func (c client) allows(redirectURI string) bool {
	if slices.Contains(c.RedirectURIs, redirectURI) {
		return true
	}

	requested, err := url.Parse(redirectURI)
	if err != nil || !isLoopbackHTTP(requested) {
		return false
	}

	for _, registered := range c.RedirectURIs {
		u, err := url.Parse(registered)
		if err != nil || !isLoopbackHTTP(u) {
			continue
		}
		if u.Hostname() == requested.Hostname() && u.Path == requested.Path && u.RawQuery == requested.RawQuery {
			return true
		}
	}

	return false
}

func validRedirectURI(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Fragment != "" || strings.Contains(raw, "#") {
		return false
	}

	switch strings.ToLower(u.Scheme) {
	case "https":
		return u.Host != ""
	case "http":
		return isLoopbackHTTP(u)
	case "javascript", "data", "file", "vbscript", "blob", "about", "ftp", "ws", "wss":
		return false
	default:
		// A private-use scheme such as cursor:// or vscode://, owned by the app
		// that registered it.
		return true
	}
}

func isLoopbackHTTP(u *url.URL) bool {
	if !strings.EqualFold(u.Scheme, "http") {
		return false
	}

	host := u.Hostname()
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)

	return ip != nil && ip.IsLoopback()
}

// cleanClientName is shown on the consent page. It is chosen by whoever
// registered the client, so it is never trusted to say who the client is; the
// page leads with where the code will be sent instead.
func cleanClientName(name string) string {
	name = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, strings.TrimSpace(name))

	if runes := []rune(name); len(runes) > maxClientNameLen {
		name = string(runes[:maxClientNameLen])
	}

	return name
}

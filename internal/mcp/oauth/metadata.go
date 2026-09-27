package oauth

import (
	"net/http"

	"github.com/amir20/dozzle/internal/auth"
	mcpauth "github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/oauthex"
)

// Everything is derived from the request, the same way the OAuth login callback
// is, so Dozzle needs no base-URL flag. A forged Host only produces metadata
// that points the forger at themselves.

// issuer is this deployment's URL, which is also where its metadata lives.
func (s *Server) issuer(r *http.Request) string {
	return auth.Origin(r) + s.base
}

// resource is the MCP endpoint's canonical URL, the audience of every token.
func (s *Server) resource(r *http.Request) string {
	return s.issuer(r) + "/api/mcp"
}

// ResourceMetadataURL is where the WWW-Authenticate challenge points. It sits
// under the base so it is reachable behind a proxy that only forwards the base.
func (s *Server) ResourceMetadataURL(r *http.Request) string {
	return s.issuer(r) + "/.well-known/oauth-protected-resource"
}

// ResourceMetadata is RFC 9728 protected resource metadata.
func (s *Server) ResourceMetadata(w http.ResponseWriter, r *http.Request) {
	mcpauth.ProtectedResourceMetadataHandler(&oauthex.ProtectedResourceMetadata{
		Resource:               s.resource(r),
		AuthorizationServers:   []string{s.issuer(r)},
		ScopesSupported:        []string{Scope},
		BearerMethodsSupported: []string{"header"},
		ResourceName:           "Dozzle",
	}).ServeHTTP(w, r)
}

type authServerMetadata struct {
	Issuer                                 string   `json:"issuer"`
	AuthorizationEndpoint                  string   `json:"authorization_endpoint"`
	TokenEndpoint                          string   `json:"token_endpoint"`
	RegistrationEndpoint                   string   `json:"registration_endpoint"`
	ScopesSupported                        []string `json:"scopes_supported"`
	ResponseTypesSupported                 []string `json:"response_types_supported"`
	GrantTypesSupported                    []string `json:"grant_types_supported"`
	TokenEndpointAuthMethodsSupported      []string `json:"token_endpoint_auth_methods_supported"`
	CodeChallengeMethodsSupported          []string `json:"code_challenge_methods_supported"`
	AuthorizationResponseIssParamSupported bool     `json:"authorization_response_iss_parameter_supported"`
}

// AuthServerMetadata is RFC 8414 authorization server metadata.
func (s *Server) AuthServerMetadata(w http.ResponseWriter, r *http.Request) {
	issuer := s.issuer(r)
	allowCORS(w)
	writeJSON(w, http.StatusOK, authServerMetadata{
		Issuer:                                 issuer,
		AuthorizationEndpoint:                  issuer + "/mcp/authorize",
		TokenEndpoint:                          issuer + "/api/oauth/token",
		RegistrationEndpoint:                   issuer + "/api/oauth/register",
		ScopesSupported:                        []string{Scope},
		ResponseTypesSupported:                 []string{"code"},
		GrantTypesSupported:                    []string{"authorization_code", "refresh_token"},
		TokenEndpointAuthMethodsSupported:      []string{"none"},
		CodeChallengeMethodsSupported:          []string{"S256"},
		AuthorizationResponseIssParamSupported: true,
	})
}

// CORS lets a browser-based client (the MCP inspector, a web IDE) use the
// public endpoints. None of them read a cookie, so allowing any origin grants
// nothing a native client could not already do.
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allowCORS(w)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func allowCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Mcp-Protocol-Version")
}

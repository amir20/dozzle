package web

import (
	"bytes"
	"context"
	"encoding/json"
	"maps"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/auth"
	mcpauth "github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/modelcontextprotocol/go-sdk/oauthex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func newMCPOAuthServer(t *testing.T, base string) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(createHandler(nil, nil, Config{
		Base:      base,
		EnableMCP: true,
		Authorization: Authorization{
			Provider: SIMPLE,
			Authorizer: auth.NewSimpleAuth(auth.UserDatabase{
				Users: map[string]*auth.User{
					"amir": {
						Username: "amir",
						Password: "$2a$10$4Tvzu0ms9shlv4B8pIfqI.TM9CoqsamsAznP91A1NGuwg/68SGS1m",
					},
				},
			}, time.Hour, testSecret),
		},
	}))
	t.Cleanup(server.Close)

	return server
}

// sessionToken logs in with the password form and returns the session JWT.
func sessionToken(t *testing.T, apiBase string) string {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.WriteField("username", "amir")
	writer.WriteField("password", "password")
	writer.Close()

	resp, err := http.Post(apiBase+"/api/token", writer.FormDataContentType(), body)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	for _, c := range resp.Cookies() {
		if c.Name == "jwt" {
			return c.Value
		}
	}
	t.Fatal("no jwt cookie after login")

	return ""
}

// approve plays the browser on the consent page: it answers the authorization
// request with the user's session and follows the redirect back to the client.
func approve(t *testing.T, apiBase, session string, authURL string, allow bool) url.Values {
	t.Helper()

	u, err := url.Parse(authURL)
	require.NoError(t, err)
	assert.True(t, strings.HasSuffix(u.Path, "/mcp/authorize"), "authorization endpoint is the consent page, got %s", u.Path)

	describe, err := http.NewRequest(http.MethodGet, apiBase+"/api/oauth/authorize?"+u.RawQuery, nil)
	require.NoError(t, err)
	describe.AddCookie(&http.Cookie{Name: "jwt", Value: session})
	resp, err := http.DefaultClient.Do(describe)
	require.NoError(t, err)
	var info map[string]string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&info))
	resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, info)
	assert.True(t, strings.HasPrefix(info["redirectHost"], "http://127.0.0.1:"), info["redirectHost"])

	payload, _ := json.Marshal(map[string]any{"query": u.RawQuery, "approve": allow})
	decide, err := http.NewRequest(http.MethodPost, apiBase+"/api/oauth/authorize", bytes.NewReader(payload))
	require.NoError(t, err)
	decide.Header.Set("Content-Type", "application/json")
	decide.AddCookie(&http.Cookie{Name: "jwt", Value: session})
	resp, err = http.DefaultClient.Do(decide)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	back, err := url.Parse(result["redirectUrl"])
	require.NoError(t, err)

	return back.Query()
}

func connectWithOAuth(t *testing.T, apiBase string) *mcp.ClientSession {
	t.Helper()

	session := sessionToken(t, apiBase)
	handler, err := mcpauth.NewAuthorizationCodeHandler(&mcpauth.AuthorizationCodeHandlerConfig{
		DynamicClientRegistrationConfig: &mcpauth.DynamicClientRegistrationConfig{
			Metadata: &oauthex.ClientRegistrationMetadata{
				ClientName:   "test client",
				RedirectURIs: []string{"http://127.0.0.1:1/callback"},
				GrantTypes:   []string{"authorization_code", "refresh_token"},
			},
		},
		AuthorizationCodeFetcher: func(ctx context.Context, args *mcpauth.AuthorizationArgs) (*mcpauth.AuthorizationResult, error) {
			q := approve(t, apiBase, session, args.URL, true)
			return &mcpauth.AuthorizationResult{Code: q.Get("code"), State: q.Get("state"), Iss: q.Get("iss")}, nil
		},
	})
	require.NoError(t, err)

	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil)
	cs, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{
		Endpoint:     apiBase + "/api/mcp",
		OAuthHandler: handler,
	}, nil)
	require.NoError(t, err)
	t.Cleanup(func() { cs.Close() })

	return cs
}

func Test_mcpOAuth_fullFlow(t *testing.T) {
	for _, base := range []string{"/", "/dozzle"} {
		t.Run(base, func(t *testing.T) {
			server := newMCPOAuthServer(t, base)
			apiBase := strings.TrimSuffix(server.URL+base, "/")

			cs := connectWithOAuth(t, apiBase)

			tools, err := cs.ListTools(context.Background(), nil)
			require.NoError(t, err)
			assert.NotEmpty(t, tools.Tools)

			res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: "list_hosts"})
			require.NoError(t, err)
			assert.False(t, res.IsError)
		})
	}
}

func Test_mcpOAuth_unauthenticatedGetsChallenge(t *testing.T) {
	server := newMCPOAuthServer(t, "/")

	resp, err := http.Post(server.URL+"/api/mcp", "application/json", strings.NewReader(`{}`))
	require.NoError(t, err)
	resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("WWW-Authenticate"), `resource_metadata="`+server.URL+`/.well-known/oauth-protected-resource"`)
}

const initializeBody = `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"t","version":"1"}}}`

func mcpInitialize(t *testing.T, endpoint, bearer string) int {
	t.Helper()

	req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(initializeBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Authorization", "Bearer "+bearer)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()

	return resp.StatusCode
}

// The docs told simple-auth users to send the session JWT as a Bearer token.
func Test_mcpOAuth_sessionBearerStillWorks(t *testing.T) {
	server := newMCPOAuthServer(t, "/")

	assert.Equal(t, http.StatusOK, mcpInitialize(t, server.URL+"/api/mcp", sessionToken(t, server.URL)))
}

// tokenPair runs the flow by hand and returns the token response.
func tokenPair(t *testing.T, apiBase string) (clientID string, tokens map[string]any) {
	t.Helper()

	reg, err := http.Post(apiBase+"/api/oauth/register", "application/json",
		strings.NewReader(`{"client_name":"x","redirect_uris":["http://127.0.0.1:1/callback"]}`))
	require.NoError(t, err)
	var registered map[string]any
	require.NoError(t, json.NewDecoder(reg.Body).Decode(&registered))
	reg.Body.Close()
	require.Equal(t, http.StatusCreated, reg.StatusCode)
	clientID = registered["client_id"].(string)

	verifier := "a-verifier-that-is-long-enough-to-be-a-real-pkce-verifier-123"
	authURL := apiBase + "/mcp/authorize?" + url.Values{
		"response_type":         {"code"},
		"client_id":             {clientID},
		"redirect_uri":          {"http://127.0.0.1:54321/callback"},
		"code_challenge":        {pkceChallenge(verifier)},
		"code_challenge_method": {"S256"},
		"state":                 {"s"},
	}.Encode()
	q := approve(t, apiBase, sessionToken(t, apiBase), authURL, true)
	require.NotEmpty(t, q.Get("code"))
	assert.Equal(t, "s", q.Get("state"))

	exchange := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {q.Get("code")},
		"client_id":     {clientID},
		"redirect_uri":  {"http://127.0.0.1:54321/callback"},
		"code_verifier": {verifier},
	}
	resp, err := http.PostForm(apiBase+"/api/oauth/token", exchange)
	require.NoError(t, err)
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&tokens))
	resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, tokens)

	// Codes are single use.
	resp, err = http.PostForm(apiBase+"/api/oauth/token", exchange)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	return clientID, tokens
}

func pkceChallenge(verifier string) string {
	return oauth2.S256ChallengeFromVerifier(verifier)
}

func Test_mcpOAuth_accessTokenOnlyWorksForMCP(t *testing.T) {
	server := newMCPOAuthServer(t, "/")
	_, tokens := tokenPair(t, server.URL)
	access := tokens["access_token"].(string)

	assert.Equal(t, http.StatusOK, mcpInitialize(t, server.URL+"/api/mcp", access))

	req, _ := http.NewRequest(http.MethodGet, server.URL+"/api/version", nil)
	req.Header.Set("Authorization", "Bearer "+access)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, "an MCP token must not be a session")

	// Nor is the refresh token an access token.
	assert.Equal(t, http.StatusUnauthorized, mcpInitialize(t, server.URL+"/api/mcp", tokens["refresh_token"].(string)))
}

func Test_mcpOAuth_refresh(t *testing.T) {
	server := newMCPOAuthServer(t, "/")
	clientID, tokens := tokenPair(t, server.URL)

	resp, err := http.PostForm(server.URL+"/api/oauth/token", url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {tokens["refresh_token"].(string)},
		"client_id":     {clientID},
	})
	require.NoError(t, err)
	var refreshed map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&refreshed))
	resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, refreshed)

	assert.Equal(t, http.StatusOK, mcpInitialize(t, server.URL+"/api/mcp", refreshed["access_token"].(string)))

	// A refresh token is bound to the client it was issued to.
	_, other := tokenPair(t, server.URL)
	resp, err = http.PostForm(server.URL+"/api/oauth/token", url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {other["refresh_token"].(string)},
		"client_id":     {clientID},
	})
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func Test_mcpOAuth_consent(t *testing.T) {
	server := newMCPOAuthServer(t, "/")

	reg, err := http.Post(server.URL+"/api/oauth/register", "application/json",
		strings.NewReader(`{"client_name":"x","redirect_uris":["https://client.example/cb"]}`))
	require.NoError(t, err)
	var registered map[string]any
	require.NoError(t, json.NewDecoder(reg.Body).Decode(&registered))
	reg.Body.Close()
	clientID := registered["client_id"].(string)

	query := url.Values{
		"response_type":         {"code"},
		"client_id":             {clientID},
		"redirect_uri":          {"https://client.example/cb"},
		"code_challenge":        {"c"},
		"code_challenge_method": {"S256"},
	}

	t.Run("requires a session", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/api/oauth/authorize?" + query.Encode())
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	session := sessionToken(t, server.URL)
	describe := func(q url.Values) (int, map[string]string) {
		req, _ := http.NewRequest(http.MethodGet, server.URL+"/api/oauth/authorize?"+q.Encode(), nil)
		req.AddCookie(&http.Cookie{Name: "jwt", Value: session})
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		var body map[string]string
		json.NewDecoder(resp.Body).Decode(&body)
		return resp.StatusCode, body
	}

	t.Run("unregistered redirect is never followed", func(t *testing.T) {
		q := url.Values{}
		maps.Copy(q, query)
		q.Set("redirect_uri", "https://evil.example/cb")
		status, body := describe(q)
		assert.Equal(t, http.StatusBadRequest, status)
		assert.Empty(t, body["redirectUrl"])
	})

	t.Run("tampered client id", func(t *testing.T) {
		q := url.Values{}
		maps.Copy(q, query)
		q.Set("client_id", clientID+"x")
		status, body := describe(q)
		assert.Equal(t, http.StatusBadRequest, status)
		assert.Equal(t, "invalid_client", body["error"])
	})

	t.Run("missing PKCE goes back to the client", func(t *testing.T) {
		q := url.Values{}
		maps.Copy(q, query)
		q.Del("code_challenge")
		status, body := describe(q)
		assert.Equal(t, http.StatusBadRequest, status)
		assert.True(t, strings.HasPrefix(body["redirectUrl"], "https://client.example/cb?"), body["redirectUrl"])
	})

	t.Run("rejects a non-JSON approval", func(t *testing.T) {
		payload, _ := json.Marshal(map[string]any{"query": query.Encode(), "approve": true})
		req, _ := http.NewRequest(http.MethodPost, server.URL+"/api/oauth/authorize", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "text/plain")
		req.AddCookie(&http.Cookie{Name: "jwt", Value: session})
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusUnsupportedMediaType, resp.StatusCode)
	})

	t.Run("deny", func(t *testing.T) {
		payload, _ := json.Marshal(map[string]any{"query": query.Encode(), "approve": false})
		req, _ := http.NewRequest(http.MethodPost, server.URL+"/api/oauth/authorize", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{Name: "jwt", Value: session})
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		var body map[string]string
		json.NewDecoder(resp.Body).Decode(&body)
		resp.Body.Close()
		assert.Contains(t, body["redirectUrl"], "error=access_denied")
	})
}

func Test_mcpOAuth_registerRejectsUnsafeRedirects(t *testing.T) {
	server := newMCPOAuthServer(t, "/")

	for _, uri := range []string{"http://example.com/cb", "javascript:alert(1)", "https://ok.example/cb#frag", "data:text/html,x"} {
		body, _ := json.Marshal(map[string]any{"redirect_uris": []string{uri}})
		resp, err := http.Post(server.URL+"/api/oauth/register", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode, uri)
	}
}

func Test_mcpOAuth_consentPageCannotBeFramed(t *testing.T) {
	server := newMCPOAuthServer(t, "/")

	req, _ := http.NewRequest(http.MethodGet, server.URL+"/mcp/authorize?x=1", nil)
	req.AddCookie(&http.Cookie{Name: "jwt", Value: sessionToken(t, server.URL)})
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()

	assert.Equal(t, "DENY", resp.Header.Get("X-Frame-Options"))
	assert.Contains(t, resp.Header.Get("Content-Security-Policy"), "frame-ancestors 'none'")

	req, _ = http.NewRequest(http.MethodGet, server.URL+"/MCP/Authorize/?x=1", nil)
	req.AddCookie(&http.Cookie{Name: "jwt", Value: sessionToken(t, server.URL)})
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, "DENY", resp.Header.Get("X-Frame-Options"), "every spelling vue-router renders")
}

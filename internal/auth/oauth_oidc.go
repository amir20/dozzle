package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"
)

// oidcProvider is a generic OpenID Connect relying party. One implementation
// covers Google, Keycloak, Pocket ID, Zitadel, Authentik and anything else that
// publishes a discovery document.
type oidcProvider struct {
	issuer       string
	clientID     string
	clientSecret string
	displayName  string
	scopes       []string
	client       *http.Client

	// Discovery is fetched once and cached, but a failure is not cached: an IdP
	// that is down while Dozzle boots must not disable SSO until the next
	// restart.
	mu        sync.Mutex
	discovery *oidcDiscovery
}

type oidcDiscovery struct {
	Issuer        string   `json:"issuer"`
	AuthURL       string   `json:"authorization_endpoint"`
	TokenURL      string   `json:"token_endpoint"`
	UserInfoURL   string   `json:"userinfo_endpoint"`
	JWKSURL       string   `json:"jwks_uri"`
	ScopesSupport []string `json:"scopes_supported"`
}

// NewOIDCProvider builds a provider for an issuer URL such as
// https://accounts.google.com or https://keycloak.example.com/realms/main.
func NewOIDCProvider(issuer, clientID, clientSecret, displayName string) *oidcProvider {
	if displayName == "" {
		displayName = "SSO"
	}

	return &oidcProvider{
		issuer:       strings.TrimSuffix(strings.TrimSpace(issuer), "/"),
		clientID:     clientID,
		clientSecret: clientSecret,
		displayName:  displayName,
		scopes:       []string{"openid", "profile", "email"},
		client:       &http.Client{Timeout: 10 * time.Second},
	}
}

func (o *oidcProvider) ID() string          { return "oidc" }
func (o *oidcProvider) DisplayName() string { return o.displayName }
func (o *oidcProvider) Icon() string        { return "mdi:shield-account" }

// discover fetches and caches the issuer's OpenID configuration.
func (o *oidcProvider) discover(ctx context.Context) (*oidcDiscovery, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.discovery != nil {
		return o.discovery, nil
	}

	url := o.issuer + "/.well-known/openid-configuration"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := o.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not reach the OIDC issuer at %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OIDC discovery at %s returned %s", url, resp.Status)
	}

	var discovery oidcDiscovery
	if err := json.NewDecoder(resp.Body).Decode(&discovery); err != nil {
		return nil, fmt.Errorf("could not parse the OIDC discovery document: %w", err)
	}

	// Mix-up defence: a document that names a different issuer than the one
	// configured means the URL is not the authority it claims to be.
	if strings.TrimSuffix(discovery.Issuer, "/") != o.issuer {
		return nil, fmt.Errorf("OIDC discovery issuer %q does not match the configured issuer %q", discovery.Issuer, o.issuer)
	}

	if discovery.AuthURL == "" || discovery.TokenURL == "" {
		return nil, fmt.Errorf("OIDC discovery document is missing an authorization or token endpoint")
	}

	// Identity is read from userinfo, so an issuer without one cannot be used.
	if discovery.UserInfoURL == "" {
		return nil, fmt.Errorf("OIDC issuer %s does not publish a userinfo endpoint", o.issuer)
	}

	o.discovery = &discovery

	return o.discovery, nil
}

// oauth2Config builds the exchange config from the discovery document. It is
// called on the request path, where discovery is already warm after the first
// login; a cold failure surfaces as a failed login rather than a panic.
func (o *oidcProvider) oauth2Config(callbackURI string) *oauth2.Config {
	config := &oauth2.Config{
		ClientID:     o.clientID,
		ClientSecret: o.clientSecret,
		Scopes:       o.scopes,
		// Unlike GitHub, OIDC requires redirect_uri on the authorize request and
		// the identical value again at the token exchange.
		RedirectURL: callbackURI,
	}

	if discovery, err := o.discover(context.Background()); err == nil {
		config.Endpoint = oauth2.Endpoint{AuthURL: discovery.AuthURL, TokenURL: discovery.TokenURL}
	}

	return config
}

// match resolves on the verified email.
//
// OIDC has no equivalent of GitHub's stable login: `sub` is stable but opaque and
// nobody wants to paste a UUID into users.yml, and `preferred_username` is not
// guaranteed unique or stable. The email is the one claim operators actually
// know, which is why identity() refuses to return an unverified one.
func (o *oidcProvider) match(users userLookup, id externalIdentity) (User, bool) {
	if id.Email == "" {
		return User{}, false
	}

	return users.byVerifiedEmail(id.Email)
}

type oidcUserInfo struct {
	Sub               string `json:"sub"`
	Email             string `json:"email"`
	EmailVerified     any    `json:"email_verified"`
	Name              string `json:"name"`
	PreferredUsername string `json:"preferred_username"`
}

// verified normalizes email_verified, which some issuers send as the string
// "true" rather than a boolean.
func (u oidcUserInfo) verified() bool {
	switch v := u.EmailVerified.(type) {
	case bool:
		return v
	case string:
		return strings.EqualFold(v, "true")
	default:
		return false
	}
}

// identity reads the claims from the userinfo endpoint.
//
// The id_token's signature is deliberately not checked here, because nothing
// rests on it: the code was exchanged by this process, directly against the
// endpoint named in a discovery document we fetched over TLS from the configured
// issuer, authenticated with the client secret. userinfo is then read over that
// same TLS channel with the resulting access token, so its claims come from the
// issuer rather than through the browser. Hand-rolling JWKS fetching, caching
// and rotation to re-derive the same claims would add a place to get signature
// validation subtly wrong without adding a guarantee.
func (o *oidcProvider) identity(ctx context.Context, token *oauth2.Token) (externalIdentity, error) {
	discovery, err := o.discover(ctx)
	if err != nil {
		return externalIdentity{}, err
	}

	client := o.oauth2Config("").Client(ctx, token)
	client.Timeout = o.client.Timeout

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, discovery.UserInfoURL, nil)
	if err != nil {
		return externalIdentity{}, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return externalIdentity{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return externalIdentity{}, fmt.Errorf("OIDC userinfo returned %s", resp.Status)
	}

	var info oidcUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return externalIdentity{}, err
	}

	if info.Email == "" {
		return externalIdentity{}, fmt.Errorf("OIDC userinfo returned no email; the issuer must grant the email scope")
	}

	// Matching an unverified address would let anyone who can register at a
	// sloppy IdP claim a Dozzle account by typing in someone else's email.
	if !info.verified() {
		return externalIdentity{}, fmt.Errorf("OIDC userinfo reported email %q as unverified", info.Email)
	}

	return externalIdentity{
		Sub:   info.Sub,
		Login: info.PreferredUsername,
		Email: info.Email,
		Name:  info.Name,
	}, nil
}

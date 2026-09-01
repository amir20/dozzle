package imagecheck

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// acceptManifests lists every manifest media type we are willing to receive.
// Multi-arch images resolve to an index/manifest-list, which is the digest
// recorded in the local RepoDigests, so those come first.
var acceptManifests = strings.Join([]string{
	"application/vnd.oci.image.index.v1+json",
	"application/vnd.docker.distribution.manifest.list.v2+json",
	"application/vnd.docker.distribution.manifest.v2+json",
	"application/vnd.oci.image.manifest.v1+json",
}, ",")

var (
	// ErrAuthRequired means the registry rejected an anonymous request. Dozzle
	// has no credential store, so private images are reported rather than
	// retried.
	ErrAuthRequired = errors.New("registry requires authentication")
	// ErrNotFound means the tag no longer exists upstream.
	ErrNotFound = errors.New("image not found in registry")
	// ErrRateLimited means the registry asked us to back off.
	ErrRateLimited = errors.New("registry rate limited the request")
)

type cachedToken struct {
	token     string
	expiresAt time.Time
}

// Registry resolves the current manifest digest for an image reference using
// HEAD requests, which registries do not count against image pull rate limits.
type Registry struct {
	client *http.Client

	mu     sync.Mutex
	tokens map[string]cachedToken
}

func NewRegistry(timeout time.Duration) *Registry {
	return &Registry{
		client: &http.Client{Timeout: timeout},
		tokens: make(map[string]cachedToken),
	}
}

// Digest returns the manifest digest the registry currently serves for ref.
// It never downloads the manifest body: only the Docker-Content-Digest header
// is needed, so a HEAD is enough and costs no pull quota.
func (r *Registry) Digest(ctx context.Context, ref Reference) (string, error) {
	log.Debug().Str("url", ref.manifestURL()).Msg("image update check: HEAD manifest")

	resp, err := r.head(ctx, ref, "")
	if err != nil {
		log.Debug().Err(err).Str("url", ref.manifestURL()).Msg("image update check: manifest request failed")
		return "", err
	}

	// An anonymous HEAD is enough for registries that serve public images
	// without a token (for example mcr.microsoft.com).
	if resp.StatusCode == http.StatusUnauthorized {
		challenge := resp.Header.Get("WWW-Authenticate")
		resp.Body.Close()

		log.Debug().
			Str("repository", ref.Repository).
			Str("challenge", challenge).
			Msg("image update check: registry asked for a token")

		token, err := r.token(ctx, ref, challenge)
		if err != nil {
			log.Debug().Err(err).Str("repository", ref.Repository).Msg("image update check: could not get a token")
			return "", err
		}

		resp, err = r.head(ctx, ref, token)
		if err != nil {
			log.Debug().Err(err).Str("url", ref.manifestURL()).Msg("image update check: retry with token failed")
			return "", err
		}
	}
	defer resp.Body.Close()

	log.Debug().
		Str("repository", ref.Repository).
		Int("status", resp.StatusCode).
		Str("contentType", resp.Header.Get("Content-Type")).
		Msg("image update check: manifest response")

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized, http.StatusForbidden:
		return "", ErrAuthRequired
	case http.StatusNotFound:
		return "", ErrNotFound
	case http.StatusTooManyRequests:
		return "", ErrRateLimited
	default:
		return "", fmt.Errorf("registry returned %s for %s", resp.Status, ref.Repository)
	}

	digest := resp.Header.Get("Docker-Content-Digest")
	if digest == "" {
		return "", fmt.Errorf("registry omitted Docker-Content-Digest for %s", ref.Repository)
	}

	log.Debug().Str("repository", ref.Repository).Str("digest", digest).Msg("image update check: registry digest")

	return digest, nil
}

func (r *Registry) head(ctx context.Context, ref Reference, token string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, ref.manifestURL(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", acceptManifests)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	return r.client.Do(req)
}

// token fetches (and caches) a bearer token for ref by following the realm
// advertised in the registry's 401 challenge. This keeps the client generic
// across Docker Hub, ghcr.io, quay.io and self-hosted registries.
func (r *Registry) token(ctx context.Context, ref Reference, challenge string) (string, error) {
	key := ref.host() + "/" + ref.Repository

	r.mu.Lock()
	if cached, ok := r.tokens[key]; ok && time.Now().Before(cached.expiresAt) {
		r.mu.Unlock()
		log.Debug().Str("repository", ref.Repository).Msg("image update check: reusing cached token")
		return cached.token, nil
	}
	r.mu.Unlock()

	realm, service := parseChallenge(challenge)
	if realm == "" {
		return "", ErrAuthRequired
	}

	endpoint, err := url.Parse(realm)
	if err != nil {
		return "", fmt.Errorf("invalid auth realm %q: %w", realm, err)
	}

	// The realm is chosen by the registry, so it decides where Dozzle sends
	// its next request. Requiring TLS stops a hostile or compromised registry
	// from pointing that request at a plaintext internal address such as a
	// cloud metadata endpoint. Loopback registries are exempt for the same
	// reason they are allowed over HTTP at all.
	if err := validateRealm(endpoint, ref); err != nil {
		return "", err
	}
	query := endpoint.Query()
	if service != "" {
		query.Set("service", service)
	}
	query.Set("scope", "repository:"+ref.Repository+":pull")
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return "", err
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", ErrAuthRequired
	}

	// Registries disagree on the field name: Docker Hub returns "token",
	// ghcr.io returns "access_token". Accept either.
	var body struct {
		Token       string `json:"token"`
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", err
	}

	token := body.Token
	if token == "" {
		token = body.AccessToken
	}
	if token == "" {
		return "", ErrAuthRequired
	}

	ttl := time.Duration(body.ExpiresIn) * time.Second
	if ttl <= 0 {
		ttl = 60 * time.Second
	}

	// The token itself is a credential and is deliberately never logged.
	log.Debug().
		Str("repository", ref.Repository).
		Str("realm", realm).
		Dur("ttl", ttl).
		Msg("image update check: obtained registry token")

	r.mu.Lock()
	// Expire the token a little early so a request never races the deadline.
	r.tokens[key] = cachedToken{token: token, expiresAt: time.Now().Add(ttl - 10*time.Second)}
	r.mu.Unlock()

	return token, nil
}

// validateRealm restricts where a registry can send us for a token.
func validateRealm(endpoint *url.URL, ref Reference) error {
	if endpoint.Host == "" {
		return fmt.Errorf("auth realm %q has no host", endpoint)
	}

	if endpoint.Scheme == "https" {
		return nil
	}

	// A loopback registry is already trusted over plain HTTP, but only for
	// itself: it cannot send us to some other host in the clear.
	if endpoint.Scheme == "http" && ref.Insecure() && sameHost(endpoint.Host, ref.Registry) {
		return nil
	}

	return fmt.Errorf("refusing auth realm %q: must be https", endpoint)
}

func sameHost(a, b string) bool {
	return strings.EqualFold(a, b)
}

// parseChallenge pulls realm and service out of a Bearer WWW-Authenticate
// header, e.g. `Bearer realm="https://auth.docker.io/token",service="registry.docker.io"`.
func parseChallenge(header string) (realm string, service string) {
	if !strings.HasPrefix(strings.ToLower(header), "bearer ") {
		return "", ""
	}

	for part := range strings.SplitSeq(header[len("bearer "):], ",") {
		key, value, found := strings.Cut(strings.TrimSpace(part), "=")
		if !found {
			continue
		}
		value = strings.Trim(value, `"`)
		switch strings.ToLower(key) {
		case "realm":
			realm = value
		case "service":
			service = value
		}
	}

	return realm, service
}

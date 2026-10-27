package imagecheck

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
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
	// ErrLowQuota means a version lookup was skipped because the registry's
	// pull allowance is nearly spent. Those pulls are better left for the
	// update itself.
	ErrLowQuota = errors.New("registry pull allowance is low")
)

// minPullsLeft is how many pulls a registry must still allow before a version
// lookup may spend one. Updating a container needs pulls too.
const minPullsLeft = 20

type pullQuota struct {
	remaining int
	window    time.Duration
	seenAt    time.Time
}

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
	// quotas is the pull allowance each registry last reported, by host.
	quotas map[string]pullQuota
}

func NewRegistry(timeout time.Duration) *Registry {
	return &Registry{
		client: &http.Client{Timeout: timeout},
		tokens: make(map[string]cachedToken),
		quotas: make(map[string]pullQuota),
	}
}

// Digest returns the manifest digest the registry currently serves for ref.
// It never downloads the manifest body: only the Docker-Content-Digest header
// is needed, so a HEAD is enough and costs no pull quota.
func (r *Registry) Digest(ctx context.Context, ref Reference) (string, error) {
	log.Debug().Str("url", ref.manifestURL()).Msg("image update check: HEAD manifest")

	resp, err := r.send(ctx, ref, http.MethodHead, ref.manifestURL(), acceptManifests)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	log.Debug().
		Str("repository", ref.Repository).
		Int("status", resp.StatusCode).
		Str("contentType", resp.Header.Get("Content-Type")).
		Msg("image update check: manifest response")

	if err := statusError(resp, ref); err != nil {
		return "", err
	}

	digest := resp.Header.Get("Docker-Content-Digest")
	if digest == "" {
		return "", fmt.Errorf("registry omitted Docker-Content-Digest for %s", ref.Repository)
	}

	log.Debug().Str("repository", ref.Repository).Str("digest", digest).Msg("image update check: registry digest")

	return digest, nil
}

// send makes one registry request, fetching a token and retrying once when the
// registry asks for one. A token already held for the repository goes on the
// first request, so only the first request per token pays for the 401. An
// anonymous request is enough for registries that serve public images without
// a token (for example mcr.microsoft.com).
func (r *Registry) send(ctx context.Context, ref Reference, method, target, accept string) (resp *http.Response, err error) {
	defer func() {
		if resp != nil {
			r.noteQuota(ref, resp)
		}
	}()

	cached := r.cachedToken(ref)
	resp, err = r.request(ctx, method, target, accept, cached)
	if err != nil {
		log.Debug().Err(err).Str("url", target).Msg("image update check: registry request failed")
		return nil, err
	}

	if resp.StatusCode != http.StatusUnauthorized {
		return resp, nil
	}

	challenge := resp.Header.Get("WWW-Authenticate")
	resp.Body.Close()

	// The registry refused the token it issued (revoked, or its clock runs
	// ahead of ours), so it is not offered again.
	if cached != "" {
		r.dropToken(ref)
	}

	log.Debug().
		Str("repository", ref.Repository).
		Str("challenge", challenge).
		Msg("image update check: registry asked for a token")

	token, err := r.token(ctx, ref, challenge)
	if err != nil {
		log.Debug().Err(err).Str("repository", ref.Repository).Msg("image update check: could not get a token")
		return nil, err
	}

	resp, err = r.request(ctx, method, target, accept, token)
	if err != nil {
		log.Debug().Err(err).Str("url", target).Msg("image update check: retry with token failed")
		return nil, err
	}
	return resp, nil
}

func (r *Registry) request(ctx context.Context, method, target, accept, token string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, target, nil)
	if err != nil {
		return nil, err
	}
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	return r.client.Do(req)
}

// noteQuota records the pull allowance a registry reports. Docker Hub sends it
// on every manifest response, HEADs included, so it is known before a single
// pull is spent. Registries that do not send it are never held back.
func (r *Registry) noteQuota(ref Reference, resp *http.Response) {
	header := resp.Header.Get("RateLimit-Remaining")
	if header == "" {
		return
	}

	// "100;w=3600": pulls left, then the window in seconds.
	count, params, _ := strings.Cut(header, ";")
	remaining, err := strconv.Atoi(strings.TrimSpace(count))
	if err != nil {
		return
	}
	window := 6 * time.Hour
	if seconds, found := strings.CutPrefix(strings.TrimSpace(params), "w="); found {
		if n, err := strconv.Atoi(seconds); err == nil && n > 0 {
			window = time.Duration(n) * time.Second
		}
	}

	r.mu.Lock()
	r.quotas[ref.host()] = pullQuota{remaining: remaining, window: window, seenAt: time.Now()}
	r.mu.Unlock()
}

// quotaLow reports whether ref's registry recently said it has few pulls left.
// Once the window it reported has passed, the allowance has refilled.
func (r *Registry) quotaLow(ref Reference) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	q, ok := r.quotas[ref.host()]
	return ok && time.Since(q.seenAt) < q.window && q.remaining < minPullsLeft
}

// statusError maps a registry response status to the errors callers act on.
func statusError(resp *http.Response, ref Reference) error {
	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusUnauthorized, http.StatusForbidden:
		return ErrAuthRequired
	case http.StatusNotFound:
		return ErrNotFound
	case http.StatusTooManyRequests:
		return ErrRateLimited
	default:
		return fmt.Errorf("registry returned %s for %s", resp.Status, ref.Repository)
	}
}

// token fetches (and caches) a bearer token for ref by following the realm
// advertised in the registry's 401 challenge. This keeps the client generic
// across Docker Hub, ghcr.io, quay.io and self-hosted registries.
func (r *Registry) token(ctx context.Context, ref Reference, challenge string) (string, error) {
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
	r.tokens[tokenKey(ref)] = cachedToken{token: token, expiresAt: time.Now().Add(ttl - 10*time.Second)}
	r.mu.Unlock()

	return token, nil
}

func tokenKey(ref Reference) string {
	return ref.host() + "/" + ref.Repository
}

// cachedToken returns the unexpired token held for ref's repository, or "".
func (r *Registry) cachedToken(ref Reference) string {
	r.mu.Lock()
	defer r.mu.Unlock()

	if cached, ok := r.tokens[tokenKey(ref)]; ok && time.Now().Before(cached.expiresAt) {
		return cached.token
	}
	return ""
}

func (r *Registry) dropToken(ref Reference) {
	r.mu.Lock()
	delete(r.tokens, tokenKey(ref))
	r.mu.Unlock()
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

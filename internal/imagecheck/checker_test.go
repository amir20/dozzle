package imagecheck

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestChecker points a Checker at a fake registry. The returned counter
// tracks how many manifest requests actually reached the network.
func newTestChecker(t *testing.T, ttl time.Duration, handler http.HandlerFunc) (*Checker, *atomic.Int32, string) {
	t.Helper()

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/manifests/") {
			calls.Add(1)
		}
		handler(w, r)
	}))
	t.Cleanup(server.Close)

	host := strings.TrimPrefix(server.URL, "http://")
	registry := NewRegistry(5 * time.Second)
	// httptest serves plain HTTP; route the client's HTTPS URLs back to it.
	registry.client.Transport = rewriteToHTTP{host}

	return NewChecker(registry, ttl), &calls, host
}

// rewriteToHTTP sends the client's https:// requests to the test server.
type rewriteToHTTP struct{ host string }

func (t rewriteToHTTP) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	clone.URL.Scheme = "http"
	clone.URL.Host = t.host
	return http.DefaultTransport.RoundTrip(clone)
}

func manifestHandler(digest string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Docker-Content-Digest", digest)
		w.WriteHeader(http.StatusOK)
	}
}

func TestCheckUpToDate(t *testing.T) {
	checker, _, host := newTestChecker(t, time.Minute, manifestHandler("sha256:aaa"))

	result := checker.Check(context.Background(), host+"/app:latest", []string{host + "/app@sha256:aaa"}, false)

	assert.Equal(t, StatusUpToDate, result.Status)
	assert.False(t, result.UpdateAvailable())
}

func TestCheckUpdateAvailable(t *testing.T) {
	checker, _, host := newTestChecker(t, time.Minute, manifestHandler("sha256:bbb"))

	result := checker.Check(context.Background(), host+"/app:latest", []string{host + "/app@sha256:aaa"}, false)

	assert.Equal(t, StatusUpdateAvailable, result.Status)
	assert.True(t, result.UpdateAvailable())
	assert.Equal(t, "sha256:aaa", result.LocalDigest)
	assert.Equal(t, "sha256:bbb", result.RemoteDigest)
}

// A tag can map to more than one local RepoDigest, so a match against any of
// them means the container is current.
func TestCheckMatchesAnyLocalDigest(t *testing.T) {
	checker, _, host := newTestChecker(t, time.Minute, manifestHandler("sha256:bbb"))

	result := checker.Check(context.Background(), host+"/app:latest", []string{host + "/app@sha256:aaa", host + "/app@sha256:bbb"}, false)

	assert.Equal(t, StatusUpToDate, result.Status)
}

func TestCheckPinnedImageIsNeverStale(t *testing.T) {
	checker, calls, host := newTestChecker(t, time.Minute, manifestHandler("sha256:bbb"))

	result := checker.Check(context.Background(), host+"/app@sha256:aaa", []string{host + "/app@sha256:aaa"}, false)

	assert.Equal(t, StatusPinned, result.Status)
	assert.Zero(t, calls.Load(), "pinned references must not hit the registry")
}

func TestCheckLocallyBuiltImage(t *testing.T) {
	checker, calls, host := newTestChecker(t, time.Minute, manifestHandler("sha256:bbb"))

	result := checker.Check(context.Background(), host+"/app:latest", nil, false)

	assert.Equal(t, StatusNotCheckable, result.Status)
	assert.Zero(t, calls.Load(), "images without a RepoDigest must not hit the registry")
}

func TestCheckAuthRequired(t *testing.T) {
	checker, _, host := newTestChecker(t, time.Minute, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})

	result := checker.Check(context.Background(), host+"/app:latest", []string{host + "/app@sha256:aaa"}, false)

	assert.Equal(t, StatusAuthRequired, result.Status)
}

func TestCheckCachesRemoteDigest(t *testing.T) {
	checker, calls, host := newTestChecker(t, time.Minute, manifestHandler("sha256:aaa"))

	for range 5 {
		checker.Check(context.Background(), host+"/app:latest", []string{host + "/app@sha256:aaa"}, false)
	}

	assert.EqualValues(t, 1, calls.Load(), "repeated checks should be served from cache")
}

// The cache stores the remote digest, not the verdict, so pulling a new image
// locally flips the status immediately rather than waiting for the TTL.
func TestCacheStoresDigestNotVerdict(t *testing.T) {
	checker, calls, host := newTestChecker(t, time.Minute, manifestHandler("sha256:bbb"))

	stale := checker.Check(context.Background(), host+"/app:latest", []string{host + "/app@sha256:aaa"}, false)
	require.Equal(t, StatusUpdateAvailable, stale.Status)

	updated := checker.Check(context.Background(), host+"/app:latest", []string{host + "/app@sha256:bbb"}, false)
	assert.Equal(t, StatusUpToDate, updated.Status)
	assert.EqualValues(t, 1, calls.Load(), "the local change should not require a new fetch")
}

func TestForceBypassesCache(t *testing.T) {
	checker, calls, host := newTestChecker(t, time.Minute, manifestHandler("sha256:aaa"))

	checker.Check(context.Background(), host+"/app:latest", []string{host + "/app@sha256:aaa"}, false)
	checker.Check(context.Background(), host+"/app:latest", []string{host + "/app@sha256:aaa"}, true)

	assert.EqualValues(t, 2, calls.Load())
}

// A registry that is down should not be retried on every page view.
func TestFailuresAreCached(t *testing.T) {
	checker, calls, host := newTestChecker(t, time.Minute, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	for range 3 {
		result := checker.Check(context.Background(), host+"/app:latest", []string{host + "/app@sha256:aaa"}, false)
		assert.Equal(t, StatusUnknown, result.Status)
	}

	assert.EqualValues(t, 1, calls.Load())
}

func TestCacheExpires(t *testing.T) {
	checker, calls, host := newTestChecker(t, time.Millisecond, manifestHandler("sha256:aaa"))

	checker.Check(context.Background(), host+"/app:latest", []string{host + "/app@sha256:aaa"}, false)
	time.Sleep(5 * time.Millisecond)
	checker.Check(context.Background(), host+"/app:latest", []string{host + "/app@sha256:aaa"}, false)

	assert.EqualValues(t, 2, calls.Load())
}

// The token flow is exercised end to end: a 401 challenge, a realm fetch, then
// a retry carrying the bearer token.
func TestTokenAuthFlow(t *testing.T) {
	var authorized atomic.Bool
	checker, _, host := newTestChecker(t, time.Minute, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/token":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"token":"secret","expires_in":300}`))
		case r.Header.Get("Authorization") == "Bearer secret":
			authorized.Store(true)
			w.Header().Set("Docker-Content-Digest", "sha256:aaa")
			w.WriteHeader(http.StatusOK)
		default:
			w.Header().Set("WWW-Authenticate", `Bearer realm="https://`+r.Host+`/token",service="test"`)
			w.WriteHeader(http.StatusUnauthorized)
		}
	})

	result := checker.Check(context.Background(), host+"/app:latest", []string{host + "/app@sha256:aaa"}, false)

	assert.Equal(t, StatusUpToDate, result.Status)
	assert.True(t, authorized.Load(), "expected the retry to carry a bearer token")
}

// ghcr.io returns access_token where Docker Hub returns token.
func TestTokenAuthAcceptsAccessToken(t *testing.T) {
	checker, _, host := newTestChecker(t, time.Minute, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/token":
			w.Write([]byte(`{"access_token":"secret","expires_in":300}`))
		case r.Header.Get("Authorization") == "Bearer secret":
			w.Header().Set("Docker-Content-Digest", "sha256:aaa")
			w.WriteHeader(http.StatusOK)
		default:
			w.Header().Set("WWW-Authenticate", `Bearer realm="https://`+r.Host+`/token",service="test"`)
			w.WriteHeader(http.StatusUnauthorized)
		}
	})

	result := checker.Check(context.Background(), host+"/app:latest", []string{host + "/app@sha256:aaa"}, false)
	assert.Equal(t, StatusUpToDate, result.Status)
}

func TestSkippedLabel(t *testing.T) {
	assert.True(t, Skipped(map[string]string{SkipLabel: "false"}))
	assert.True(t, Skipped(map[string]string{SkipLabel: "off"}))
	assert.False(t, Skipped(map[string]string{SkipLabel: "true"}))
	assert.False(t, Skipped(nil))
}

func TestParseMode(t *testing.T) {
	for _, valid := range []string{"automatic", "manual", "off"} {
		mode, err := ParseMode(valid)
		require.NoError(t, err)
		assert.EqualValues(t, valid, mode)
	}

	_, err := ParseMode("sometimes")
	assert.Error(t, err)
}

// HEAD is what keeps the check free of Docker Hub pull quota, so assert we
// never fall back to GET.
func TestOnlyHeadRequestsAreMade(t *testing.T) {
	var methods []string
	checker, _, host := newTestChecker(t, time.Minute, func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method)
		w.Header().Set("Docker-Content-Digest", "sha256:aaa")
		w.WriteHeader(http.StatusOK)
	})

	checker.Check(context.Background(), host+"/app:latest", []string{host + "/app@sha256:aaa"}, false)

	require.NotEmpty(t, methods)
	for _, method := range methods {
		assert.Equal(t, http.MethodHead, method)
	}
}

// An image pulled from one repository and pushed to another carries digests
// for both. Only the repository being checked is comparable, otherwise the
// reported local digest belongs to an unrelated repository.
func TestIgnoresDigestsFromOtherRepositories(t *testing.T) {
	checker, _, host := newTestChecker(t, time.Minute, manifestHandler("sha256:pushed"))

	result := checker.Check(context.Background(), host+"/app:latest", []string{
		"alpine@sha256:unrelated",
		host + "/app@sha256:pushed",
	}, false)

	assert.Equal(t, StatusUpToDate, result.Status)
	assert.Equal(t, "sha256:pushed", result.LocalDigest)
}

func TestNotCheckableWhenNoDigestMatchesTheRepository(t *testing.T) {
	checker, _, host := newTestChecker(t, time.Minute, manifestHandler("sha256:remote"))

	result := checker.Check(context.Background(), host+"/app:latest", []string{"alpine@sha256:unrelated"}, false)

	assert.Equal(t, StatusNotCheckable, result.Status)
}

// Docker records official images without their library/ prefix, so the
// normalization has to line up on both sides.
func TestMatchesOfficialImageDigests(t *testing.T) {
	checker := NewChecker(NewRegistry(time.Second), time.Minute)
	ref, err := ParseReference("alpine:latest")
	require.NoError(t, err)

	assert.Equal(t, []string{"sha256:aaa"}, digestsForRepository([]string{"alpine@sha256:aaa"}, ref))
	_ = checker
}

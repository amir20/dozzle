package imagecheck

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeConfig points DOCKER_CONFIG at a temporary config.json.
func writeConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("DOCKER_CONFIG", dir)
	path := filepath.Join(dir, "config.json")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

func authEntry(registry, username, password string) string {
	encoded := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
	return fmt.Sprintf(`{"auths":{%q:{"auth":%q}}}`, registry, encoded)
}

func TestLookupLoginMatchesRegistrySpellings(t *testing.T) {
	cases := []struct {
		key      string
		registry string
		match    bool
	}{
		{"ghcr.io", "ghcr.io", true},
		{"https://ghcr.io", "ghcr.io", true},
		{"https://ghcr.io/v2/", "ghcr.io", true},
		{"GHCR.io", "ghcr.io", true},
		{"registry.example.com:5000", "registry.example.com:5000", true},
		{"https://index.docker.io/v1/", "docker.io", true},
		{"docker.io", "docker.io", true},
		{"index.docker.io", "docker.io", true},
		{"registry-1.docker.io", "docker.io", true},
		// None of these may hand a login to another host.
		{"ghcr.io", "evil.com", false},
		{"ghcr.io", "ghcr.io.evil.com", false},
		{"registry.example.com", "registry.example.com:5000", false},
		{"registry.example.com:5000", "registry.example.com", false},
		{"https://index.docker.io/v1/", "ghcr.io", false},
	}
	for _, tc := range cases {
		t.Run(tc.key+" for "+tc.registry, func(t *testing.T) {
			writeConfig(t, authEntry(tc.key, "user", "pass"))

			found := lookupLogin(tc.registry)
			if !tc.match {
				assert.Nil(t, found)
				return
			}
			require.NotNil(t, found)
			assert.Equal(t, "user", found.username)
			assert.Equal(t, "pass", found.password)
		})
	}
}

func TestLookupLoginDecoding(t *testing.T) {
	encode := func(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }

	t.Run("a password can contain colons", func(t *testing.T) {
		writeConfig(t, fmt.Sprintf(`{"auths":{"ghcr.io":{"auth":%q}}}`, encode("user:pa:ss")))
		found := lookupLogin("ghcr.io")
		require.NotNil(t, found)
		assert.Equal(t, "pa:ss", found.password)
	})

	t.Run("a trailing newline from a hand encoded value is dropped", func(t *testing.T) {
		writeConfig(t, fmt.Sprintf(`{"auths":{"ghcr.io":{"auth":%q}}}`, encode("user:pass\n")))
		found := lookupLogin("ghcr.io")
		require.NotNil(t, found)
		assert.Equal(t, "pass", found.password)
	})

	t.Run("a Kubernetes secret spells the fields out", func(t *testing.T) {
		writeConfig(t, `{"auths":{"ghcr.io":{"username":"user","password":"pass"}}}`)
		found := lookupLogin("ghcr.io")
		require.NotNil(t, found)
		assert.Equal(t, "user", found.username)
	})

	for name, config := range map[string]string{
		"bad base64":     `{"auths":{"ghcr.io":{"auth":"!!!"}}}`,
		"no colon":       fmt.Sprintf(`{"auths":{"ghcr.io":{"auth":%q}}}`, encode("nocolon")),
		"empty password": fmt.Sprintf(`{"auths":{"ghcr.io":{"auth":%q}}}`, encode("user:")),
		"identity token": fmt.Sprintf(`{"auths":{"ghcr.io":{"auth":%q,"identitytoken":"x"}}}`, encode("00000000-0000-0000-0000-000000000000:")),
		"helper only":    `{"auths":{"ghcr.io":{}},"credsStore":"osxkeychain"}`,
		"not json":       `{`,
		"empty":          `{}`,
	} {
		t.Run(name+" is no login", func(t *testing.T) {
			writeConfig(t, config)
			assert.Nil(t, lookupLogin("ghcr.io"))
		})
	}
}

func TestLookupLoginWithoutAFile(t *testing.T) {
	t.Setenv("DOCKER_CONFIG", t.TempDir())
	assert.Nil(t, lookupLogin("ghcr.io"))
}

// `docker login` takes effect on the next lookup, with no restart.
func TestLookupLoginReadsTheFileEachTime(t *testing.T) {
	path := writeConfig(t, authEntry("ghcr.io", "old", "pass"))
	require.Equal(t, "old", lookupLogin("ghcr.io").username)

	require.NoError(t, os.WriteFile(path, []byte(authEntry("ghcr.io", "new", "pass")), 0o600))
	assert.Equal(t, "new", lookupLogin("ghcr.io").username)
}

func TestPullAuth(t *testing.T) {
	writeConfig(t, authEntry("ghcr.io", "user", "pa+ss/w?rd"))

	encoded := PullAuth("ghcr.io/acme/app:latest")
	require.NotEmpty(t, encoded)

	// The daemon reads this with URL-safe base64, not the standard alphabet.
	decoded, err := base64.URLEncoding.DecodeString(encoded)
	require.NoError(t, err)
	var got map[string]string
	require.NoError(t, json.Unmarshal(decoded, &got))
	assert.Equal(t, map[string]string{"username": "user", "password": "pa+ss/w?rd", "serveraddress": "ghcr.io"}, got)

	assert.Empty(t, PullAuth("evil.com/acme/app:latest"))
	assert.Empty(t, PullAuth("nginx:latest"))
	assert.Empty(t, PullAuth(""))
}

func TestPullAuthUsesDockerHubLoginForBareNames(t *testing.T) {
	writeConfig(t, authEntry("https://index.docker.io/v1/", "user", "pass"))

	assert.NotEmpty(t, PullAuth("nginx:latest"))
	assert.NotEmpty(t, PullAuth("acme/app"))
	assert.Empty(t, PullAuth("ghcr.io/acme/app"))
}

// privateRegistry serves a manifest only with a token issued for user:pass.
func privateRegistry(tokenRequests *[]string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/token":
			*tokenRequests = append(*tokenRequests, r.Header.Get("Authorization"))
			if username, password, ok := r.BasicAuth(); !ok || username != "user" || password != "pass" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.Write([]byte(`{"token":"secret","expires_in":300}`))
		case r.Header.Get("Authorization") == "Bearer secret":
			w.Header().Set("Docker-Content-Digest", "sha256:aaa")
		default:
			w.Header().Set("WWW-Authenticate", `Bearer realm="https://`+r.Host+`/token",service="test"`)
			w.WriteHeader(http.StatusUnauthorized)
		}
	}
}

func TestCheckUsesLoginForToken(t *testing.T) {
	var requests []string
	checker, _, host := newTestChecker(t, time.Minute, privateRegistry(&requests))
	writeConfig(t, authEntry(host, "user", "pass"))

	result := checker.Check(context.Background(), host+"/app:latest", []string{host + "/app@sha256:aaa"}, false)

	assert.Equal(t, StatusUpToDate, result.Status)
	assert.Len(t, requests, 1)
}

func TestCheckSendsNoLoginToAnotherRegistry(t *testing.T) {
	var requests []string
	checker, _, host := newTestChecker(t, time.Minute, privateRegistry(&requests))
	writeConfig(t, authEntry("ghcr.io", "user", "pass"))

	result := checker.Check(context.Background(), host+"/app:latest", []string{host + "/app@sha256:aaa"}, false)

	assert.Equal(t, StatusAuthRequired, result.Status)
	assert.Equal(t, []string{""}, requests, "the token request must stay anonymous")
}

func TestCheckWithBasicChallenge(t *testing.T) {
	var seen []string
	checker, _, host := newTestChecker(t, time.Minute, func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.Header.Get("Authorization"))
		if username, password, ok := r.BasicAuth(); ok && username == "user" && password == "pass" {
			w.Header().Set("Docker-Content-Digest", "sha256:aaa")
			return
		}
		w.Header().Set("WWW-Authenticate", `Basic realm="registry"`)
		w.WriteHeader(http.StatusUnauthorized)
	})
	image := host + "/app:latest"
	digests := []string{host + "/app@sha256:aaa"}

	t.Run("without a login", func(t *testing.T) {
		t.Setenv("DOCKER_CONFIG", t.TempDir())
		assert.Equal(t, StatusAuthRequired, checker.Check(context.Background(), image, digests, true).Status)
	})

	t.Run("with a login", func(t *testing.T) {
		writeConfig(t, authEntry(host, "user", "pass"))
		assert.Equal(t, StatusUpToDate, checker.Check(context.Background(), image, digests, true).Status)
		assert.Empty(t, seen[len(seen)-2], "the first request stays anonymous")
	})

	t.Run("with a wrong login", func(t *testing.T) {
		writeConfig(t, authEntry(host, "user", "wrong"))
		assert.Equal(t, StatusAuthRequired, checker.Check(context.Background(), image, digests, true).Status)
	})
}

// An expired login must not stop a public image from being checked.
func TestRefusedLoginFallsBackToAnonymous(t *testing.T) {
	checker, _, host := newTestChecker(t, time.Minute, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/token":
			if _, _, hasLogin := r.BasicAuth(); hasLogin {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.Write([]byte(`{"token":"anonymous","expires_in":300}`))
		case r.Header.Get("Authorization") == "Bearer anonymous":
			w.Header().Set("Docker-Content-Digest", "sha256:aaa")
		default:
			w.Header().Set("WWW-Authenticate", `Bearer realm="https://`+r.Host+`/token",service="test"`)
			w.WriteHeader(http.StatusUnauthorized)
		}
	})
	writeConfig(t, authEntry(host, "user", "expired"))

	result := checker.Check(context.Background(), host+"/app:latest", []string{host + "/app@sha256:aaa"}, false)

	assert.Equal(t, StatusUpToDate, result.Status)
}

func TestAddingALoginEndsACachedRefusal(t *testing.T) {
	var requests []string
	checker, calls, host := newTestChecker(t, time.Hour, privateRegistry(&requests))
	path := writeConfig(t, `{}`)
	image := host + "/app:latest"
	digests := []string{host + "/app@sha256:aaa"}

	require.Equal(t, StatusAuthRequired, checker.Check(context.Background(), image, digests, false).Status)
	before := calls.Load()

	// With the file unchanged, the refusal stays cached.
	require.Equal(t, StatusAuthRequired, checker.Check(context.Background(), image, digests, false).Status)
	require.Equal(t, before, calls.Load())

	require.NoError(t, os.WriteFile(path, []byte(authEntry(host, "user", "pass")), 0o600))
	later := time.Now().Add(time.Minute)
	require.NoError(t, os.Chtimes(path, later, later))

	assert.Equal(t, StatusUpToDate, checker.Check(context.Background(), image, digests, false).Status)
}

// A changed config has no bearing on a digest that was fetched successfully.
func TestConfigChangeKeepsCachedDigests(t *testing.T) {
	checker, calls, host := newTestChecker(t, time.Hour, manifestHandler("sha256:aaa"))
	path := writeConfig(t, `{}`)
	image := host + "/app:latest"
	digests := []string{host + "/app@sha256:aaa"}

	require.Equal(t, StatusUpToDate, checker.Check(context.Background(), image, digests, false).Status)

	later := time.Now().Add(time.Minute)
	require.NoError(t, os.Chtimes(path, later, later))
	require.Equal(t, StatusUpToDate, checker.Check(context.Background(), image, digests, false).Status)

	assert.EqualValues(t, 1, calls.Load())
}

func TestRefuseDowngrade(t *testing.T) {
	parse := func(raw string) *http.Request {
		target, err := url.Parse(raw)
		require.NoError(t, err)
		return &http.Request{URL: target}
	}
	https, http2 := parse("https://registry.example.com/a"), parse("http://registry.example.com/a")

	assert.Error(t, refuseDowngrade(http2, []*http.Request{https}))
	assert.NoError(t, refuseDowngrade(https, []*http.Request{https}))
	assert.NoError(t, refuseDowngrade(http2, []*http.Request{http2}), "a loopback registry is plain http throughout")
	assert.Error(t, refuseDowngrade(https, make10(https)))
}

func make10(req *http.Request) []*http.Request {
	via := make([]*http.Request, 10)
	for i := range via {
		via[i] = req
	}
	return via
}

package imagecheck

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeRegistry serves JSON documents by the last path segment (a digest).
func fakeRegistry(docs map[string]any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		segments := strings.Split(r.URL.Path, "/")
		doc, ok := docs[segments[len(segments)-1]]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(doc)
	}
}

func platform(os, arch string) map[string]string {
	return map[string]string{"os": os, "architecture": arch}
}

// Official Docker Hub images annotate each platform in the index, so a
// single request answers.
func TestVersionFromIndexAnnotation(t *testing.T) {
	checker, calls, host := newTestChecker(t, time.Minute, fakeRegistry(map[string]any{
		"sha256:index": map[string]any{
			"mediaType": "application/vnd.oci.image.index.v1+json",
			"manifests": []any{
				map[string]any{
					"digest":      "sha256:attestation",
					"platform":    platform("unknown", "unknown"),
					"annotations": map[string]string{VersionLabel: "wrong"},
				},
				map[string]any{
					"digest":      "sha256:amd64",
					"platform":    platform("linux", "amd64"),
					"annotations": map[string]string{VersionLabel: "18.6-alpine3.24"},
				},
			},
		},
	}))

	version, err := checker.version(context.Background(), host+"/postgres:18-alpine", "sha256:index")

	require.NoError(t, err)
	assert.Equal(t, "18.6-alpine3.24", version)
	assert.Equal(t, int32(1), calls.Load())
}

// Without annotations the version comes from the image config's labels,
// through the platform manifest.
func TestVersionFromConfigLabel(t *testing.T) {
	checker, calls, host := newTestChecker(t, time.Minute, fakeRegistry(map[string]any{
		"sha256:index": map[string]any{
			"mediaType": "application/vnd.oci.image.index.v1+json",
			"manifests": []any{map[string]any{"digest": "sha256:image", "platform": platform("linux", "arm64")}},
		},
		"sha256:image": map[string]any{
			"mediaType": "application/vnd.oci.image.manifest.v1+json",
			"config":    map[string]string{"digest": "sha256:config"},
		},
		"sha256:config": map[string]any{
			"config": map[string]any{"Labels": map[string]string{VersionLabel: "12.3.8"}},
		},
	}))

	version, err := checker.version(context.Background(), host+"/app:latest", "sha256:index")

	require.NoError(t, err)
	assert.Equal(t, "12.3.8", version)
	assert.Equal(t, int32(2), calls.Load())
}

// A digest never changes, so an image without a version is not asked again.
func TestVersionMissingIsCached(t *testing.T) {
	checker, calls, host := newTestChecker(t, time.Minute, fakeRegistry(map[string]any{
		"sha256:image": map[string]any{
			"mediaType": "application/vnd.oci.image.manifest.v1+json",
			"config":    map[string]string{"digest": "sha256:config"},
		},
		"sha256:config": map[string]any{"config": map[string]any{}},
	}))

	for range 2 {
		version, err := checker.version(context.Background(), host+"/app:latest", "sha256:image")
		require.NoError(t, err)
		assert.Empty(t, version)
	}
	assert.Equal(t, int32(1), calls.Load())
}

// A failure is kept for a while, so a refusing registry is not asked on
// every check.
func TestVersionErrorIsCachedBriefly(t *testing.T) {
	checker, calls, host := newTestChecker(t, time.Minute, fakeRegistry(map[string]any{}))

	for range 2 {
		_, err := checker.version(context.Background(), host+"/app:latest", "sha256:gone")
		assert.ErrorIs(t, err, ErrNotFound)
	}
	assert.Equal(t, int32(1), calls.Load())
}

// Versions are read with GETs, never the HEAD the update check uses.
func TestVersionUsesGet(t *testing.T) {
	var methods []string
	checker, _, host := newTestChecker(t, time.Minute, func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method)
		_ = json.NewEncoder(w).Encode(map[string]any{"annotations": map[string]string{VersionLabel: "1.0"}})
	})

	version, err := checker.version(context.Background(), host+"/app:latest", "sha256:image")

	require.NoError(t, err)
	assert.Equal(t, "1.0", version)
	assert.Equal(t, []string{http.MethodGet}, methods)
}

// withVersionsRegistry answers the update check's HEAD with sha256:new and
// serves a version for both digests. remaining, when set, is the pull
// allowance Docker Hub style registries report.
func withVersionsRegistry(remaining string) http.HandlerFunc {
	versions := fakeRegistry(map[string]any{
		"sha256:old": map[string]any{"annotations": map[string]string{VersionLabel: "1.0"}},
		"sha256:new": map[string]any{"annotations": map[string]string{VersionLabel: "1.1"}},
	})
	return func(w http.ResponseWriter, r *http.Request) {
		if remaining != "" {
			w.Header().Set("RateLimit-Remaining", remaining)
		}
		if r.Method == http.MethodHead {
			w.Header().Set("Docker-Content-Digest", "sha256:new")
			return
		}
		versions(w, r)
	}
}

func TestWithVersionsFillsBothSides(t *testing.T) {
	checker, _, host := newTestChecker(t, time.Minute, withVersionsRegistry(""))
	result := checker.Check(context.Background(), host+"/app:latest", []string{host + "/app@sha256:old"}, false)

	result = checker.WithVersions(context.Background(), result, nil)

	assert.Equal(t, "1.0", result.CurrentVersion)
	assert.Equal(t, "1.1", result.LatestVersion)
}

// The running version comes from the container's labels when they have it,
// which saves a request.
func TestWithVersionsPrefersLabel(t *testing.T) {
	checker, calls, host := newTestChecker(t, time.Minute, withVersionsRegistry(""))
	result := checker.Check(context.Background(), host+"/app:latest", []string{host + "/app@sha256:old"}, false)

	result = checker.WithVersions(context.Background(), result, map[string]string{VersionLabel: "1.0-label"})

	assert.Equal(t, "1.0-label", result.CurrentVersion)
	assert.Equal(t, "1.1", result.LatestVersion)
	assert.Equal(t, int32(2), calls.Load(), "one HEAD for the check, one GET for the latest")
}

// Versions cost pulls, so nothing is fetched unless an update is waiting.
func TestWithVersionsOnlyForUpdates(t *testing.T) {
	checker, calls, host := newTestChecker(t, time.Minute, withVersionsRegistry(""))
	result := checker.Check(context.Background(), host+"/app:latest", []string{host + "/app@sha256:new"}, false)

	result = checker.WithVersions(context.Background(), result, nil)

	assert.Equal(t, StatusUpToDate, result.Status)
	assert.Empty(t, result.LatestVersion)
	assert.Equal(t, int32(1), calls.Load())
}

// A registry close to its pull limit keeps its pulls for the update itself,
// and the version is read once the allowance is back.
func TestWithVersionsSkipsWhenQuotaIsLow(t *testing.T) {
	remaining := "5;w=3600"
	checker, calls, host := newTestChecker(t, time.Minute, func(w http.ResponseWriter, r *http.Request) {
		withVersionsRegistry(remaining)(w, r)
	})
	image, local := host+"/app:latest", []string{host + "/app@sha256:old"}

	result := checker.WithVersions(context.Background(), checker.Check(context.Background(), image, local, false), nil)

	assert.Equal(t, StatusUpdateAvailable, result.Status)
	assert.Empty(t, result.LatestVersion)
	assert.Equal(t, int32(1), calls.Load(), "only the HEAD")

	remaining = "80;w=3600"
	result = checker.WithVersions(context.Background(), checker.Check(context.Background(), image, local, true), nil)

	assert.Equal(t, "1.1", result.LatestVersion)
}

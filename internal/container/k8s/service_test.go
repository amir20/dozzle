package k8s

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/imagecheck"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

// fakeRegistry serves one digest for every manifest. Loopback registries are
// reached over plain HTTP, so the real checker talks to it unmodified.
func fakeRegistry(t *testing.T, digest string) string {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Docker-Content-Digest", digest)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)
	return strings.TrimPrefix(server.URL, "http://")
}

func testImageService() *Service {
	return &Service{checker: imagecheck.NewChecker(imagecheck.NewRegistry(5*time.Second), time.Minute)}
}

func TestCheckImageUpdateComparesPodDigest(t *testing.T) {
	host := fakeRegistry(t, "sha256:new")
	k := testImageService()

	stale := container.Container{Image: host + "/api:latest", ImageDigest: host + "/api@sha256:old"}
	result, err := k.CheckImageUpdate(t.Context(), stale, false)
	require.NoError(t, err)
	assert.Equal(t, imagecheck.StatusUpdateAvailable, result.Status)

	current := container.Container{Image: host + "/api:latest", ImageDigest: host + "/api@sha256:new"}
	result, err = k.CheckImageUpdate(t.Context(), current, false)
	require.NoError(t, err)
	assert.Equal(t, imagecheck.StatusUpToDate, result.Status)
}

// An image loaded straight onto the node has no registry digest to compare.
func TestCheckImageUpdateWithoutDigestIsNotCheckable(t *testing.T) {
	result, err := testImageService().CheckImageUpdate(t.Context(), container.Container{Image: "app:dev"}, false)
	require.NoError(t, err)
	assert.Equal(t, imagecheck.StatusNotCheckable, result.Status)
}

func TestCheckImageUpdateHonorsSkipLabel(t *testing.T) {
	c := container.Container{Image: "nginx:latest", Labels: map[string]string{imagecheck.SkipLabel: "false"}}
	result, err := testImageService().CheckImageUpdate(t.Context(), c, false)
	require.NoError(t, err)
	assert.Equal(t, imagecheck.StatusSkipped, result.Status)
}

func TestImageDigest(t *testing.T) {
	assert.Equal(t, "docker.io/library/nginx@sha256:abc", imageDigest("docker.io/library/nginx@sha256:abc"))
	assert.Equal(t, "nginx@sha256:abc", imageDigest("docker-pullable://nginx@sha256:abc"))
	assert.Empty(t, imageDigest("sha256:abc"))
	assert.Empty(t, imageDigest(""))
}

func TestPodToContainersCarriesImageDigest(t *testing.T) {
	client := newTestK8sClient(t)
	pod := podWithOwner()
	pod.Status.ContainerStatuses = []corev1.ContainerStatus{
		{Name: "api", ImageID: "docker.io/example/api@sha256:abc", State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}},
	}

	containers := client.podToContainers(t.Context(), pod)
	require.Len(t, containers, 1)
	assert.Equal(t, "docker.io/example/api@sha256:abc", containers[0].ImageDigest)
}

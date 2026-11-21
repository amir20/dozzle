package swap

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	dcontainer "github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const podID = "81c6bd11dddfc26784f02e8a4e58972b7bbc7cb6fcbdc3564aae81df98c3ba7d"

// podClient serves the libpod inspect of "inpod" (in pod_app, the way
// podman-compose creates it) and "nopod" in Podman 6.2's shape, and the pod
// itself when podReadable. It counts every other request.
func podClient(t *testing.T, podReadable bool) (*client.Client, *int) {
	t.Helper()
	other := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v4.0.0/libpod/containers/inpod/json":
			io.WriteString(w, `{"Id":"inpod","Pod":"`+podID+`","HostConfig":{"UsernsMode":""}}`)
		case r.URL.Path == "/v4.0.0/libpod/containers/nopod/json":
			io.WriteString(w, `{"Id":"nopod","Pod":"","HostConfig":{"UsernsMode":""}}`)
		case r.URL.Path == "/v4.0.0/libpod/pods/"+podID+"/json" && podReadable:
			io.WriteString(w, `{"Id":"`+podID+`","Name":"pod_app"}`)
		case strings.Contains(r.URL.Path, "/libpod/"):
			w.WriteHeader(http.StatusNotFound)
		default:
			other++
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	t.Cleanup(server.Close)
	cli, err := client.New(client.WithHost(strings.Replace(server.URL, "http://", "tcp://", 1)))
	require.NoError(t, err)
	t.Cleanup(func() { cli.Close() })
	return cli, &other
}

func TestPodmanPod(t *testing.T) {
	cli, _ := podClient(t, true)
	assert.Equal(t, "pod_app", PodmanPod(t.Context(), cli, "inpod"))
	assert.Equal(t, "", PodmanPod(t.Context(), cli, "nopod"))
	assert.Equal(t, "", PodmanPod(t.Context(), cli, "missing"), "libpod can't answer: no pod")

	unnamed, _ := podClient(t, false)
	assert.Equal(t, podID[:12], PodmanPod(t.Context(), unnamed, "inpod"), "a pod that can't be read is named by its id")

	assert.Equal(t, "", PodmanPod(t.Context(), struct{}{}, "inpod"), "a client without a dialer: no pod")
}

func runningInspect(id string) dcontainer.InspectResponse {
	return dcontainer.InspectResponse{
		ID:         id,
		Name:       "/app",
		State:      &dcontainer.State{Running: true, Status: "running"},
		Config:     &dcontainer.Config{Image: "app:latest"},
		HostConfig: &dcontainer.HostConfig{},
	}
}

func TestSwapRefusesPodmanPod(t *testing.T) {
	cli, other := podClient(t, true)

	result, err := Swap(t.Context(), cli, runningInspect("inpod"), Options{Podman: true})

	require.ErrorIs(t, err, ErrInPod)
	var podErr *PodError
	require.ErrorAs(t, err, &podErr)
	assert.Equal(t, "pod_app", podErr.Pod)
	assert.Equal(t, Result{}, result, "nothing was touched")
	assert.Zero(t, *other, "no rename, create, stop or start was sent")
}

func TestSwapPodCheckOnlyOnPodman(t *testing.T) {
	cli, other := podClient(t, true)

	// On Docker libpod is never asked, and the swap goes ahead (failing here
	// only because the test server is not an engine).
	_, err := Swap(t.Context(), cli, runningInspect("inpod"), Options{})
	require.Error(t, err)
	assert.NotErrorIs(t, err, ErrInPod)
	assert.NotZero(t, *other)

	_, err = Swap(t.Context(), cli, runningInspect("nopod"), Options{Podman: true})
	require.Error(t, err)
	assert.NotErrorIs(t, err, ErrInPod, "a container outside a pod is not refused")
}

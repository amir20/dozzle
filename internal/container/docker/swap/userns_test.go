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

func TestKeepIDMode(t *testing.T) {
	// Rootless Podman 6.2, `podman run --userns keep-id` as uid 501, gid 1000.
	assert.Equal(t, "keep-id:uid=501,gid=1000", KeepIDMode(
		[]string{"0:1:501", "501:0:1", "502:502:999499"},
		[]string{"0:1:1000", "1000:0:1", "1001:1001:999000"},
	))
	// Plain rootless default: container root is the user, nothing to keep.
	assert.Equal(t, "", KeepIDMode([]string{"0:0:1", "1:1:65536"}, []string{"0:0:1", "1:1:65536"}))
	// --userns auto, or explicit --uidmap without the user in it.
	assert.Equal(t, "", KeepIDMode([]string{"0:2000:1024"}, []string{"0:2000:1024"}))
	assert.Equal(t, "", KeepIDMode(nil, nil))
}

// libpodClient serves the libpod inspect of a keep-id container, byte for byte
// in Podman 6.2's shape, and nothing else.
func libpodClient(t *testing.T) *client.Client {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v4.0.0/libpod/containers/abc123/json" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		io.WriteString(w, `{"Id":"abc123","HostConfig":{"UsernsMode":"private","IDMappings":{"UidMap":["0:1:501","501:0:1","502:502:999499"],"GidMap":["0:1:1000","1000:0:1","1001:1001:999000"]}}}`)
	}))
	t.Cleanup(server.Close)
	cli, err := client.New(client.WithHost(strings.Replace(server.URL, "http://", "tcp://", 1)))
	require.NoError(t, err)
	t.Cleanup(func() { cli.Close() })
	return cli
}

func TestWithPodmanKeepIDRestoresKeepID(t *testing.T) {
	old := dcontainer.InspectResponse{ID: "abc123", HostConfig: &dcontainer.HostConfig{UsernsMode: "private"}}

	got := WithPodmanKeepID(t.Context(), libpodClient(t), old)

	assert.Equal(t, dcontainer.UsernsMode("keep-id:uid=501,gid=1000"), got.HostConfig.UsernsMode)
	assert.Equal(t, dcontainer.UsernsMode("private"), old.HostConfig.UsernsMode, "the caller's inspect is not modified")
}

func TestWithPodmanKeepIDLeavesOtherModesAlone(t *testing.T) {
	cli := libpodClient(t)
	for _, mode := range []dcontainer.UsernsMode{"", "host", "keep-id"} {
		old := dcontainer.InspectResponse{ID: "abc123", HostConfig: &dcontainer.HostConfig{UsernsMode: mode}}
		assert.Equal(t, mode, WithPodmanKeepID(t.Context(), cli, old).HostConfig.UsernsMode)
	}
	// libpod unreachable for this id: unchanged.
	old := dcontainer.InspectResponse{ID: "missing", HostConfig: &dcontainer.HostConfig{UsernsMode: "private"}}
	assert.Equal(t, dcontainer.UsernsMode("private"), WithPodmanKeepID(t.Context(), cli, old).HostConfig.UsernsMode)
}

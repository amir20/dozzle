package docker

import (
	"net/netip"
	"testing"

	docker "github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/stretchr/testify/assert"
)

// inspect builds a response shaped like the daemon's, which always fills in
// hostname and ports even when the network mode forbids them.
func inspect(mode string) *docker.InspectResponse {
	port, err := network.ParsePort("8080/tcp")
	if err != nil {
		panic(err)
	}

	return &docker.InspectResponse{
		Config: &docker.Config{
			Hostname:     "abc123",
			ExposedPorts: network.PortSet{port: {}},
		},
		HostConfig: &docker.HostConfig{
			NetworkMode:     docker.NetworkMode(mode),
			Links:           []string{"other:alias"},
			DNS:             []netip.Addr{netip.MustParseAddr("1.1.1.1")},
			ExtraHosts:      []string{"example.com:10.0.0.1"},
			PortBindings:    network.PortMap{port: {{HostPort: "8080"}}},
			PublishAllPorts: true,
		},
		NetworkSettings: &docker.NetworkSettings{
			Networks: map[string]*network.EndpointSettings{"bridge": {Aliases: []string{"web"}}},
		},
	}
}

// A container sharing another container's namespace (a VPN sidecar, say)
// cannot declare any of this itself.
func Test_sanitizeForRecreate_containerNetworkMode(t *testing.T) {
	for _, mode := range []string{"container:abc", "container:some-name"} {
		resp := inspect(mode)

		shared := sanitizeForRecreate(resp)

		assert.True(t, shared, mode)
		assert.Empty(t, resp.Config.Hostname, mode)
		assert.Empty(t, resp.Config.ExposedPorts, mode)
		assert.Empty(t, resp.HostConfig.Links, mode)
		assert.Empty(t, resp.HostConfig.DNS, mode)
		assert.Empty(t, resp.HostConfig.ExtraHosts, mode)
		assert.Empty(t, resp.HostConfig.PortBindings, mode)
		assert.False(t, resp.HostConfig.PublishAllPorts, mode)
	}
}

func Test_sanitizeForRecreate_hostNetworkMode(t *testing.T) {
	resp := inspect("host")

	shared := sanitizeForRecreate(resp)

	assert.True(t, shared)
	assert.Empty(t, resp.Config.Hostname)
	assert.Empty(t, resp.HostConfig.Links)
	// Host networking publishes nothing, but the daemon does not reject these.
	assert.NotEmpty(t, resp.Config.ExposedPorts)
}

// An ordinary bridge container keeps everything it was created with.
func Test_sanitizeForRecreate_leavesBridgeAlone(t *testing.T) {
	resp := inspect("bridge")

	shared := sanitizeForRecreate(resp)

	assert.False(t, shared)
	assert.Equal(t, "abc123", resp.Config.Hostname)
	assert.NotEmpty(t, resp.Config.ExposedPorts)
	assert.NotEmpty(t, resp.HostConfig.PortBindings)
	assert.True(t, resp.HostConfig.PublishAllPorts)
	assert.NotEmpty(t, resp.HostConfig.Links)
}

// A host UTS namespace owns the hostname whatever the network mode is.
func Test_sanitizeForRecreate_hostUTSMode(t *testing.T) {
	resp := inspect("bridge")
	resp.HostConfig.UTSMode = docker.UTSMode("host")

	sanitizeForRecreate(resp)

	assert.Empty(t, resp.Config.Hostname)
}

func Test_sanitizeForRecreate_toleratesMissingConfig(t *testing.T) {
	assert.NotPanics(t, func() {
		sanitizeForRecreate(&docker.InspectResponse{})
	})
}

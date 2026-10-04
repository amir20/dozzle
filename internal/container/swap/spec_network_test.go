package swap

import (
	"net/netip"
	"testing"

	dcontainer "github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// networkInspect builds a response shaped like the daemon's, which always fills
// in hostname and ports even when the network mode forbids them. The hostname
// is not derived from the id, so only the network rules can clear it.
func networkInspect(mode string) dcontainer.InspectResponse {
	port, err := network.ParsePort("8080/tcp")
	if err != nil {
		panic(err)
	}

	return dcontainer.InspectResponse{
		ID: "bbbbbbbbbbbb2222222222222222222222222222222222222222222222222222",
		Config: &dcontainer.Config{
			Image:        "nginx:latest",
			Hostname:     "web",
			ExposedPorts: network.PortSet{port: {}},
		},
		HostConfig: &dcontainer.HostConfig{
			NetworkMode:     dcontainer.NetworkMode(mode),
			Links:           []string{"other:alias"},
			DNS:             []netip.Addr{netip.MustParseAddr("1.1.1.1")},
			ExtraHosts:      []string{"example.com:10.0.0.1"},
			PortBindings:    network.PortMap{port: {{HostPort: "8080"}}},
			PublishAllPorts: true,
		},
		NetworkSettings: &dcontainer.NetworkSettings{
			Networks: map[string]*network.EndpointSettings{"bridge": {Aliases: []string{"web"}}},
		},
	}
}

// A container sharing another container's namespace (a VPN sidecar, say)
// cannot declare any of this itself, and cannot be attached to networks.
func TestReplacementSpecContainerNetworkMode(t *testing.T) {
	for _, mode := range []string{"container:abc", "container:some-name"} {
		spec := ReplacementSpec(networkInspect(mode), nil, "web")

		assert.Nil(t, spec.NetworkingConfig, mode)
		assert.Empty(t, spec.Config.Hostname, mode)
		assert.Empty(t, spec.Config.ExposedPorts, mode)
		assert.Empty(t, spec.HostConfig.Links, mode)
		assert.Empty(t, spec.HostConfig.DNS, mode)
		assert.Empty(t, spec.HostConfig.ExtraHosts, mode)
		assert.Empty(t, spec.HostConfig.PortBindings, mode)
		assert.False(t, spec.HostConfig.PublishAllPorts, mode)
	}
}

func TestReplacementSpecHostNetworkMode(t *testing.T) {
	spec := ReplacementSpec(networkInspect("host"), nil, "web")

	assert.Nil(t, spec.NetworkingConfig)
	assert.Empty(t, spec.Config.Hostname)
	assert.Empty(t, spec.HostConfig.Links)
	// Host networking publishes nothing, but the daemon does not reject these.
	assert.NotEmpty(t, spec.Config.ExposedPorts)
}

// An ordinary bridge container keeps everything it was created with.
func TestReplacementSpecBridgeKeepsNetworking(t *testing.T) {
	spec := ReplacementSpec(networkInspect("bridge"), nil, "web")

	require.NotNil(t, spec.NetworkingConfig)
	assert.Equal(t, []string{"web"}, spec.NetworkingConfig.EndpointsConfig["bridge"].Aliases)
	assert.Equal(t, "web", spec.Config.Hostname)
	assert.NotEmpty(t, spec.Config.ExposedPorts)
	assert.NotEmpty(t, spec.HostConfig.PortBindings)
	assert.True(t, spec.HostConfig.PublishAllPorts)
	assert.NotEmpty(t, spec.HostConfig.Links)
}

// A host UTS namespace owns the hostname whatever the network mode is.
func TestReplacementSpecHostUTSMode(t *testing.T) {
	old := networkInspect("bridge")
	old.HostConfig.UTSMode = dcontainer.UTSMode("host")

	spec := ReplacementSpec(old, nil, "web")

	assert.Empty(t, spec.Config.Hostname)
}

func TestReplacementSpecToleratesMissingConfig(t *testing.T) {
	assert.NotPanics(t, func() {
		ReplacementSpec(dcontainer.InspectResponse{}, nil, "web")
	})
}

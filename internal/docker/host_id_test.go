package docker

import (
	"context"
	"errors"
	"testing"

	"github.com/amir20/dozzle/internal/container"
	"github.com/moby/moby/api/types/system"
	"github.com/moby/moby/client"
	"github.com/stretchr/testify/assert"
)

// stubCLI answers only the two calls newClient makes while working out a host
// id. Everything else is left to the embedded nil interface and would panic,
// which is the point: these tests should not reach any further.
type stubCLI struct {
	DockerCLI
	info     system.Info
	platform string
	infoErr  error
}

func (s *stubCLI) Info(context.Context, client.InfoOptions) (client.SystemInfoResult, error) {
	return client.SystemInfoResult{Info: s.info}, s.infoErr
}

func (s *stubCLI) ServerVersion(context.Context, client.ServerVersionOptions) (client.ServerVersionResult, error) {
	return client.ServerVersionResult{Platform: client.PlatformInfo{Name: s.platform}}, nil
}

// podmanID saves every case below repeating the runtime argument, which only
// the "not podman" test actually varies.
func podmanID(info system.Info) string {
	return podmanHostID(info, "podman")
}

func podmanInfo(hostname, graphRoot string) system.Info {
	return system.Info{
		// what Podman actually puts here: a fresh uuid on every single call
		ID:            "11111111-1111-1111-1111-111111111111",
		Name:          hostname,
		DockerRootDir: graphRoot,
	}
}

func TestPodmanHostID_StableAcrossCalls(t *testing.T) {
	info := podmanInfo("node-1", "/var/lib/containers/storage")

	first := podmanID(info)

	// the same host, one restart later: Podman hands out a different ID and
	// nothing else about the machine has moved
	info.ID = "22222222-2222-2222-2222-222222222222"

	assert.NotEmpty(t, first)
	assert.Equal(t, first, podmanID(info))
}

func TestPodmanHostID_SeparatesRootlessUsersOnOneMachine(t *testing.T) {
	// two rootless users share a hostname but never a store, which is the whole
	// reason the graph root is in the hash
	alice := podmanID(podmanInfo("node-1", "/home/alice/.local/share/containers/storage"))
	bob := podmanID(podmanInfo("node-1", "/home/bob/.local/share/containers/storage"))

	assert.NotEqual(t, alice, bob)
}

func TestPodmanHostID_SeparatesHosts(t *testing.T) {
	one := podmanID(podmanInfo("node-1", "/var/lib/containers/storage"))
	two := podmanID(podmanInfo("node-2", "/var/lib/containers/storage"))

	assert.NotEqual(t, one, two)
}

// Hashing two empty strings would hand every host in the fleet the same id, and
// the duplicate check in RetriableClientManager would then drop all but one of
// them. A churning id is bad; hosts silently vanishing is worse.
func TestPodmanHostID_EmptyWhenNothingToHash(t *testing.T) {
	assert.Empty(t, podmanID(podmanInfo("", "")))
	assert.NotEmpty(t, podmanID(podmanInfo("node-1", "")))
	assert.NotEmpty(t, podmanID(podmanInfo("", "/var/lib/containers/storage")))
}

// The candidate declines for itself rather than making newClient ask what
// runtime it is looking at.
func TestPodmanHostID_DeclinesForOtherRuntimes(t *testing.T) {
	info := podmanInfo("node-1", "/var/lib/containers/storage")

	assert.Empty(t, podmanHostID(info, "docker"))
	assert.Empty(t, podmanHostID(info, ""))
	assert.NotEmpty(t, podmanHostID(info, "podman"))
}

func TestNewClient_PodmanDerivesStableID(t *testing.T) {
	info := podmanInfo("node-1", "/var/lib/containers/storage")
	cli := &stubCLI{info: info, platform: "Podman Engine"}

	c := newClient(cli, container.Host{}, "")

	assert.Equal(t, "podman", c.host.Runtime)
	assert.NotEqual(t, info.ID, c.host.ID)
	assert.Equal(t, podmanID(info), c.host.ID)
}

// Docker's ID comes from /var/lib/docker/engine-id and survives a daemon
// restart, so it stays exactly as it is. Deriving one there would migrate every
// existing install's host ids for no gain.
func TestNewClient_DockerKeepsEngineID(t *testing.T) {
	info := system.Info{ID: "a-stable-engine-id", Name: "node-1", DockerRootDir: "/var/lib/docker"}
	cli := &stubCLI{info: info, platform: "Docker Engine - Community"}

	c := newClient(cli, container.Host{}, "")

	assert.Equal(t, "docker", c.host.Runtime)
	assert.Equal(t, "a-stable-engine-id", c.host.ID)
}

func TestNewClient_OverrideWins(t *testing.T) {
	cli := &stubCLI{info: podmanInfo("node-1", "/var/lib/containers/storage"), platform: "Podman Engine"}

	c := newClient(cli, container.Host{}, "my-host")

	assert.Equal(t, "my-host", c.host.ID)
}

// An unreachable engine used to leave the host with an empty id, which resolves
// to nothing and routes nowhere. ParseConnection already derived one from the
// remote URL, so fall back to it rather than throwing it away.
func TestNewClient_FallsBackToTheCallersID(t *testing.T) {
	cli := &stubCLI{infoErr: errors.New("connection refused")}

	c := newClient(cli, container.Host{ID: "tcp:10.0.0.5:2375"}, "")

	assert.Equal(t, "tcp:10.0.0.5:2375", c.host.ID)
}

func TestNewClient_SwarmNodeIDBeatsDerivedID(t *testing.T) {
	info := system.Info{ID: "engine-id", Name: "node-1"}
	info.Swarm.NodeID = "swarm-node-id"
	cli := &stubCLI{info: info, platform: "Docker Engine"}

	c := newClient(cli, container.Host{}, "")

	assert.Equal(t, "swarm-node-id", c.host.ID)
	assert.True(t, c.host.Swarm)
}

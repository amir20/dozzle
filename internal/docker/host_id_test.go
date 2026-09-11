package docker

import (
	"context"
	"errors"
	"testing"

	"github.com/amir20/dozzle/internal/container"
	"github.com/moby/moby/api/types/system"
	"github.com/moby/moby/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubCLI answers only the two calls NewClient makes while describing the
// engine. Everything else is left to the embedded nil interface and would
// panic, which is the point: these tests should not reach any further.
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

// spyResolver records what the client said about the engine. What the resolver
// does with it is container's business and is tested there; all this package
// owes is an accurate description and using the answer it gets back.
type spyResolver struct {
	seen container.EngineIdentity
	id   string
}

func (s *spyResolver) Resolve(engine container.EngineIdentity) string {
	s.seen = engine
	return s.id
}

func TestNewClient_DescribesTheEngine(t *testing.T) {
	cli := &stubCLI{
		info: system.Info{
			ID:            "engine-id",
			Name:          "node-1",
			DockerRootDir: "/var/lib/containers/storage",
		},
		platform: "Podman Engine",
	}
	spy := &spyResolver{id: "resolved"}

	c := NewClient(cli, container.Host{ID: "tcp:10.0.0.5:2375"}, spy)

	require.Equal(t, container.EngineIdentity{
		Runtime:     "podman",
		EngineID:    "engine-id",
		Hostname:    "node-1",
		StorageRoot: "/var/lib/containers/storage",
		Fallback:    "tcp:10.0.0.5:2375",
	}, spy.seen)
	assert.Equal(t, "resolved", c.host.ID)
}

func TestNewClient_ReportsSwarmNodeID(t *testing.T) {
	info := system.Info{ID: "engine-id", Name: "node-1"}
	info.Swarm.NodeID = "swarm-node-id"
	spy := &spyResolver{id: "resolved"}

	c := NewClient(&stubCLI{info: info, platform: "Docker Engine"}, container.Host{}, spy)

	assert.Equal(t, "swarm-node-id", spy.seen.SwarmNodeID)
	assert.Equal(t, "docker", spy.seen.Runtime)
	assert.True(t, c.host.Swarm)
}

// An engine we cannot reach still has to produce a host, so the resolver is
// asked anyway and gets an identity carrying only what the caller knew.
func TestNewClient_DescribesAnUnreachableEngine(t *testing.T) {
	spy := &spyResolver{id: "tcp:10.0.0.5:2375"}

	c := NewClient(&stubCLI{infoErr: errors.New("connection refused")}, container.Host{ID: "tcp:10.0.0.5:2375"}, spy)

	assert.Equal(t, container.EngineIdentity{Runtime: "docker", Fallback: "tcp:10.0.0.5:2375"}, spy.seen)
	assert.Equal(t, "tcp:10.0.0.5:2375", c.host.ID)
}

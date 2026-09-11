package container

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func podman(hostname, storageRoot string) EngineIdentity {
	return EngineIdentity{
		Runtime: "podman",
		// what Podman actually puts here: a fresh uuid on every single call
		EngineID:    "11111111-1111-1111-1111-111111111111",
		Hostname:    hostname,
		StorageRoot: storageRoot,
	}
}

func derive(engine EngineIdentity) string {
	return DerivedHostID{}.Resolve(engine)
}

func TestDerivedHostID_PodmanIsStableAcrossRestarts(t *testing.T) {
	engine := podman("node-1", "/var/lib/containers/storage")

	first := derive(engine)

	// the same host, one restart later: Podman hands out a different id and
	// nothing else about the machine has moved
	engine.EngineID = "22222222-2222-2222-2222-222222222222"

	assert.NotEmpty(t, first)
	assert.NotEqual(t, engine.EngineID, first)
	assert.Equal(t, first, derive(engine))
}

func TestDerivedHostID_SeparatesRootlessUsersOnOneMachine(t *testing.T) {
	// two rootless users share a hostname but never a store, which is the whole
	// reason the storage root is in the hash
	alice := derive(podman("node-1", "/home/alice/.local/share/containers/storage"))
	bob := derive(podman("node-1", "/home/bob/.local/share/containers/storage"))

	assert.NotEqual(t, alice, bob)
}

func TestDerivedHostID_SeparatesHosts(t *testing.T) {
	one := derive(podman("node-1", "/var/lib/containers/storage"))
	two := derive(podman("node-2", "/var/lib/containers/storage"))

	assert.NotEqual(t, one, two)
}

// Hashing two empty strings would hand every host in the fleet the same id, and
// the duplicate check in RetriableClientManager would then drop all but one of
// them. A churning id is bad; hosts silently vanishing is worse. So fall through
// to the random engine id rather than inventing a shared one.
func TestDerivedHostID_PodmanWithNothingToHashKeepsEngineID(t *testing.T) {
	engine := podman("", "")

	assert.Equal(t, engine.EngineID, derive(engine))
	assert.NotEqual(t, engine.EngineID, derive(podman("node-1", "")))
	assert.NotEqual(t, engine.EngineID, derive(podman("", "/var/lib/containers/storage")))
}

// Docker's id comes from /var/lib/docker/engine-id and survives a daemon
// restart, so it is used as is. Deriving one there would migrate every existing
// install's host ids for no gain.
func TestDerivedHostID_DockerKeepsEngineID(t *testing.T) {
	engine := EngineIdentity{
		Runtime:     "docker",
		EngineID:    "a-stable-engine-id",
		Hostname:    "node-1",
		StorageRoot: "/var/lib/docker",
	}

	assert.Equal(t, "a-stable-engine-id", derive(engine))
}

func TestDerivedHostID_KubernetesUsesMachineID(t *testing.T) {
	assert.Equal(t, "machine-id", derive(EngineIdentity{Runtime: "k8s", EngineID: "machine-id"}))
}

// Every other node of the swarm already refers to this one by its node id.
func TestDerivedHostID_SwarmNodeIDWins(t *testing.T) {
	engine := EngineIdentity{
		Runtime:     "docker",
		EngineID:    "engine-id",
		SwarmNodeID: "swarm-node-id",
	}

	assert.Equal(t, "swarm-node-id", derive(engine))
}

// An unreachable engine reports nothing at all. ParseConnection already derived
// an id from the remote URL, so use it rather than leaving the host with an
// empty id that routes nowhere.
func TestDerivedHostID_FallsBackToWhatTheCallerKnew(t *testing.T) {
	assert.Equal(t, "tcp:10.0.0.5:2375", derive(EngineIdentity{Fallback: "tcp:10.0.0.5:2375"}))
}

func TestDerivedHostID_EmptyWhenNothingIsKnown(t *testing.T) {
	assert.Empty(t, derive(EngineIdentity{}))
}

func TestStaticHostID_WinsOverEverything(t *testing.T) {
	engine := podman("node-1", "/var/lib/containers/storage")
	engine.SwarmNodeID = "swarm-node-id"

	assert.Equal(t, "my-host", StaticHostID("my-host").Resolve(engine))
}

func TestNewHostIDResolver(t *testing.T) {
	assert.Equal(t, DerivedHostID{}, NewHostIDResolver(""))
	assert.Equal(t, StaticHostID("my-host"), NewHostIDResolver("my-host"))
}

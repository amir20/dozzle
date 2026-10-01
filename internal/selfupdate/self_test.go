package selfupdate

import (
	"context"
	"testing"

	dcontainer "github.com/moby/moby/api/types/container"
	"github.com/stretchr/testify/assert"
)

const (
	sidecarID = "5555555555550000000000000000000000000000000000000000000000000000"
	appID     = "6666666666660000000000000000000000000000000000000000000000000000"
)

func sharedNamespace(f *fakeDocker) {
	f.containers[sidecarID] = dcontainer.InspectResponse{ID: sidecarID, Name: "/sidecar", HostConfig: &dcontainer.HostConfig{NetworkMode: "bridge"}}
	f.containers[appID] = dcontainer.InspectResponse{ID: appID, Name: "/app", HostConfig: &dcontainer.HostConfig{NetworkMode: "container:" + sidecarID}}
	self := f.containers[selfID]
	self.HostConfig = &dcontainer.HostConfig{NetworkMode: "container:" + sidecarID}
	f.containers[selfID] = self
}

func TestResolveSelfNoSharedNamespace(t *testing.T) {
	f := newFake()
	assert.Equal(t, selfID, resolveSelf(context.Background(), f, selfID, t.TempDir()))
}

// Behind a sidecar, mountinfo names the sidecar. The marker finds Dozzle among
// everything joined to it. #5289
func TestResolveSelfBehindSidecar(t *testing.T) {
	f := newFake()
	sharedNamespace(f)
	f.hasMarker = map[string]bool{selfID: true}
	assert.Equal(t, selfID, resolveSelf(context.Background(), f, sidecarID, t.TempDir()))
}

func TestResolveSelfSidecarIsDozzle(t *testing.T) {
	f := newFake()
	sharedNamespace(f)
	f.hasMarker = map[string]bool{sidecarID: true}
	assert.Equal(t, sidecarID, resolveSelf(context.Background(), f, sidecarID, t.TempDir()))
}

func TestResolveSelfUnknown(t *testing.T) {
	f := newFake()
	sharedNamespace(f)
	assert.Empty(t, resolveSelf(context.Background(), f, sidecarID, t.TempDir()), "no candidate has the marker")
	assert.Empty(t, resolveSelf(context.Background(), f, sidecarID, "/nonexistent"), "the marker cannot be written")
}

func TestJoinsNetworkOf(t *testing.T) {
	const id = "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
	assert.True(t, JoinsNetworkOf("container:"+id, id, "sidecar"), "compose writes the full id")
	assert.True(t, JoinsNetworkOf("container:"+id[:12], id, "sidecar"))
	assert.True(t, JoinsNetworkOf("container:sidecar", id, "sidecar"))
	assert.False(t, JoinsNetworkOf("container:abc", id, "sidecar"), "too short to be an id")
	assert.False(t, JoinsNetworkOf("container:other", id, "sidecar"))
	assert.False(t, JoinsNetworkOf("bridge", id, "sidecar"))
	assert.False(t, JoinsNetworkOf("container:", id, ""))
}

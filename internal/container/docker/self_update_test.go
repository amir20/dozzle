package docker

import (
	"testing"

	docker_types "github.com/moby/moby/api/types/container"
	"github.com/stretchr/testify/assert"
)

func TestIsSelf(t *testing.T) {
	const full = "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
	prev := selfContainerID
	t.Cleanup(func() { selfContainerID = prev })

	selfContainerID = func() string { return full }
	assert.True(t, isSelf(full[:12]), "Dozzle's short id")
	assert.True(t, isSelf(full))
	assert.False(t, isSelf("abc"), "too short to be an id")
	assert.False(t, isSelf("0123456789ab"))

	selfContainerID = func() string { return "" }
	assert.False(t, isSelf(full[:12]), "not in a container")
}

func TestMayBeSelf(t *testing.T) {
	const full = "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
	prevID, prevHost := selfContainerID, hostname
	t.Cleanup(func() { selfContainerID, hostname = prevID, prevHost })
	hostname = func() (string, error) { return "somehost", nil }

	inspect := func(id, image string, labels map[string]string) docker_types.InspectResponse {
		return docker_types.InspectResponse{ID: id, Config: &docker_types.Config{Image: image, Labels: labels}}
	}

	selfContainerID = func() string { return "" }
	assert.True(t, mayBeSelf(inspect("0123456789ab", "amir20/dozzle:latest", nil)), "Dozzle image with no known id")
	assert.True(t, mayBeSelf(inspect("0123456789ab", "sha256:"+full, map[string]string{"dev.dozzle.self-update.image": "amir20/dozzle:latest"})), "rolled back Dozzle")
	assert.False(t, mayBeSelf(inspect("0123456789ab", "nginx", nil)))
	hostname = func() (string, error) { return full[:12], nil }
	assert.True(t, mayBeSelf(inspect(full, "my-registry/logs:latest", nil)), "renamed image, hostname is its short id")

	selfContainerID = func() string { return full }
	assert.False(t, mayBeSelf(inspect("0123456789ab", "amir20/dozzle:latest", nil)), "a known id decides through isSelf")
}

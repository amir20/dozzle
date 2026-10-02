package docker

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// A local client pointed at DOCKER_HOST=tcp:// or ssh:// is still typed "local",
// but this machine's /proc says nothing about that engine.
func TestIsLocalDaemon(t *testing.T) {
	assert.True(t, isLocalDaemon("unix:///var/run/docker.sock"))
	assert.True(t, isLocalDaemon("npipe:////./pipe/docker_engine"))
	assert.False(t, isLocalDaemon("tcp://10.0.0.5:2375"))
	assert.False(t, isLocalDaemon("ssh://user@other-box"))
	assert.False(t, isLocalDaemon(""))
}

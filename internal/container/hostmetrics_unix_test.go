//go:build !windows

package container

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// In a container /data stands in for an unmounted data directory; natively it is
// some unrelated path and must not be read.
func TestDiskPaths(t *testing.T) {
	assert.Equal(t, []string{"/var/lib/docker", "/data"}, diskPaths("/var/lib/docker", true))
	assert.Equal(t, []string{"/var/lib/docker"}, diskPaths("/var/lib/docker", false))
	assert.Equal(t, []string{"/data"}, diskPaths("", true))
	assert.Empty(t, diskPaths("", false))
}

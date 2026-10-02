//go:build !windows

package container

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// In a container /data stands in for an unmounted data directory; natively it is
// some unrelated path and must not be read.
func TestDiskPaths(t *testing.T) {
	assert.Equal(t, []string{"/var/lib/docker", "/data"}, diskPaths("/var/lib/docker", true))
	assert.Equal(t, []string{"/var/lib/docker"}, diskPaths("/var/lib/docker", false))
	assert.Equal(t, []string{"/data"}, diskPaths("", true))
	assert.Empty(t, diskPaths("", false))
}

// Each folder under the root is one drive, named after the folder; loose files
// are not drives, and a missing root means none were mounted.
func TestReadDisks(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(root, "media"), 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(root, "backup"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "notes.txt"), nil, 0o644))
	require.NoError(t, os.Symlink(t.TempDir(), filepath.Join(root, "linked")))

	disks := readDisks(root)
	require.Len(t, disks, 3)
	assert.Equal(t, "backup", disks[0].Name)
	assert.Equal(t, "linked", disks[1].Name)
	assert.Equal(t, "media", disks[2].Name)
	assert.NotZero(t, disks[0].Total)

	assert.Nil(t, readDisks(filepath.Join(root, "missing")))
}

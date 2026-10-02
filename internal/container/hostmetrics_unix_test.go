//go:build !windows

package container

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

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

// A stale network mount blocks stat and statfs in the kernel. The probe must
// return within its timeout, keep the last good reading, and never stack a
// second read behind a stuck one.
func TestProbeDriveSurvivesAHungMount(t *testing.T) {
	origStat, origTimeout := statDrive, driveProbeTimeout
	t.Cleanup(func() { statDrive, driveProbeTimeout = origStat, origTimeout })
	driveProbeTimeout = 50 * time.Millisecond

	var calls atomic.Int32
	hang := make(chan struct{})
	var hung atomic.Bool
	statDrive = func(string) (uint64, uint64, error) {
		calls.Add(1)
		if hung.Load() {
			<-hang
		}
		return 1000, 400, nil
	}
	path := "/host/disks/nas-" + t.Name()

	total, free, ok := probeDrive(path)
	require.True(t, ok)
	assert.Equal(t, uint64(1000), total)
	assert.Equal(t, uint64(400), free)

	// The share goes stale: the read hangs, the caller gets the last good value.
	hung.Store(true)
	start := time.Now()
	total, _, ok = probeDrive(path)
	assert.Less(t, time.Since(start), time.Second)
	assert.True(t, ok)
	assert.Equal(t, uint64(1000), total)

	// While that read is stuck, later ticks do not start another one.
	probeDrive(path)
	probeDrive(path)
	assert.Equal(t, int32(2), calls.Load())

	// Once it comes back, reads resume.
	hung.Store(false)
	close(hang)
	assert.Eventually(t, func() bool {
		probeDrive(path)
		return calls.Load() > 2
	}, time.Second, 10*time.Millisecond)
}

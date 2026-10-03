//go:build !windows

package container

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// hostRoot is where a containerized Dozzle finds the host: its /proc at
// /host/proc, and the extra drives operators mount to watch at /host/disks, one
// folder per drive. DOZZLE_DEV_HOST_ROOT moves it for local development only: a
// native binary on macOS has no /proc, Docker's data directory is inside Docker
// Desktop's VM, and /host cannot be created on a read-only system volume, so
// without it no host read-out ever appears under `make dev`. Not documented on
// purpose.
var hostRoot = func() string {
	if dir := os.Getenv("DOZZLE_DEV_HOST_ROOT"); dir != "" {
		return dir
	}
	return "/host"
}()

var hostDisksRoot = filepath.Join(hostRoot, "disks")

var errShort = errors.New("unexpected proc format")

// inContainer reports whether we are running inside a container, where /proc
// describes the container rather than the host. Detection is heuristic (the
// usual marker files and the init cgroup), which is all the kernel exposes.
func inContainer() bool {
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}
	if _, err := os.Stat("/run/.containerenv"); err == nil { // podman
		return true
	}
	if b, err := os.ReadFile("/proc/1/cgroup"); err == nil {
		s := string(b)
		for _, marker := range []string{"docker", "kubepods", "containerd", "lxc", "buildkit"} {
			if strings.Contains(s, marker) {
				return true
			}
		}
	}
	return false
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// hostProcRoot returns the directory holding the *host* /proc. Operators running
// Dozzle in a container mount it at /host/proc; a native install has it at /proc.
// In a container without the mount we cannot tell the host's values from the
// container's, so we report nothing rather than something wrong.
func hostProcRoot() (string, bool) {
	if proc := filepath.Join(hostRoot, "proc"); fileExists(filepath.Join(proc, "loadavg")) {
		return proc, true
	}
	if !inContainer() {
		if _, err := os.Stat("/proc/loadavg"); err == nil {
			return "/proc", true
		}
	}
	return "", false
}

// diskPaths lists where disk usage is read from, in order. statfs only needs some
// path on the filesystem that holds the engine's data, not the data itself, so in
// a container /data stands in when the data directory is not mounted: a named
// volume lives under the data directory, and a bind mount is usually on the same
// disk. A native install has no such stand-in, since its /data is unrelated.
func diskPaths(dockerRootDir string, containerized bool) []string {
	var paths []string
	if dockerRootDir != "" {
		paths = append(paths, dockerRootDir)
	}
	if containerized {
		paths = append(paths, "/data")
	}
	return paths
}

// ReadHostMetrics reads host-level metrics. Disk is read from the first of
// diskPaths that statfs accepts, the same way volume_monitor.go measures volume
// sources. Load and uptime come from the host /proc, so ok is false when it is
// not mounted; the caller must not present container values as host values.
// Disk is filled independently of that mount.
func ReadHostMetrics(dockerRootDir string) (HostMetrics, bool) {
	var m HostMetrics
	for _, path := range diskPaths(dockerRootDir, inContainer()) {
		if total, free, err := statfs(path); err == nil {
			m.DiskTotal, m.DiskFree = total, free
			break
		}
	}

	m.Disks = readDisks(hostDisksRoot)

	proc, ok := hostProcRoot()
	if !ok {
		return m, false
	}
	if l1, l5, l15, err := readLoadAvg(proc + "/loadavg"); err == nil {
		m.Load1, m.Load5, m.Load15 = l1, l5, l15
	}
	if up, err := readUptime(proc + "/uptime"); err == nil {
		m.Uptime = up
	}
	return m, true
}

func readLoadAvg(path string) (float64, float64, float64, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0, 0, 0, err
	}
	f := strings.Fields(string(b))
	if len(f) < 3 {
		return 0, 0, 0, errShort
	}
	l1, _ := strconv.ParseFloat(f[0], 64)
	l5, _ := strconv.ParseFloat(f[1], 64)
	l15, _ := strconv.ParseFloat(f[2], 64)
	return l1, l5, l15, nil
}

func readUptime(path string) (uint64, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	f := strings.Fields(string(b))
	if len(f) < 1 {
		return 0, errShort
	}
	secs, err := strconv.ParseFloat(f[0], 64)
	if err != nil {
		return 0, err
	}
	return uint64(secs), nil
}

// readDisks reads each folder under root as one drive, named after the folder and
// sorted by name so the tooltip order is stable. statfs only needs the mount
// point, so an empty folder on the drive is enough. A folder statfs rejects is
// skipped rather than shown as an empty drive.
func readDisks(root string) []Disk {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	var disks []Disk
	for _, entry := range entries {
		total, free, ok := probeDrive(filepath.Join(root, entry.Name()))
		if !ok {
			continue
		}
		disks = append(disks, Disk{Name: entry.Name(), Total: total, Free: free})
	}
	sort.Slice(disks, func(i, j int) bool { return disks[i].Name < disks[j].Name })
	return disks
}

// driveProbeTimeout bounds how long one tick waits on a drive. A local disk
// answers in microseconds; anything slower is a network mount in trouble.
var driveProbeTimeout = time.Second

// statDrive measures one drive. Stat, not entry.IsDir(), so a symlink to a mount
// point counts: that is how a native install points /host/disks at its drives.
var statDrive = func(path string) (uint64, uint64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, 0, err
	}
	if !info.IsDir() {
		return 0, 0, errNotDir
	}
	return statfs(path)
}

var errNotDir = errors.New("not a directory")

// driveProbe remembers one drive's last good reading and whether a read is
// still out. A stale NFS or SMB hard mount blocks stat and statfs in the kernel
// with no way to cancel them, so the read runs in its own goroutine: the caller
// waits at most driveProbeTimeout and falls back to the last good value, and no
// second read starts while the first is stuck. One dead share then costs a
// single parked goroutine, not a hung Host() for everything that calls it.
type driveProbe struct {
	mu       sync.Mutex
	inflight bool
	ok       bool
	total    uint64
	free     uint64
}

var driveProbes sync.Map // path -> *driveProbe

func probeDrive(path string) (total, free uint64, ok bool) {
	value, _ := driveProbes.LoadOrStore(path, &driveProbe{})
	p := value.(*driveProbe)

	p.mu.Lock()
	if p.inflight {
		defer p.mu.Unlock()
		return p.total, p.free, p.ok
	}
	p.inflight = true
	p.mu.Unlock()

	done := make(chan struct{})
	go func() {
		defer close(done)
		total, free, err := statDrive(path)
		p.mu.Lock()
		defer p.mu.Unlock()
		p.inflight = false
		// A failed or empty read drops the drive; it was unmounted or never was one.
		p.ok = err == nil && total > 0
		p.total, p.free = total, free
	}()

	select {
	case <-done:
	case <-time.After(driveProbeTimeout):
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	return p.total, p.free, p.ok
}

//go:build !windows

package container

import (
	"errors"
	"os"
	"strconv"
	"strings"
)

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

// hostProcRoot returns the directory holding the *host* /proc. Operators running
// Dozzle in a container mount it at /host/proc; a native install has it at /proc.
// In a container without the mount we cannot tell the host's values from the
// container's, so we report nothing rather than something wrong.
func hostProcRoot() (string, bool) {
	if _, err := os.Stat("/host/proc/loadavg"); err == nil {
		return "/host/proc", true
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

//go:build !windows

package container

import (
	"bufio"
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

// hostRootPath returns the directory holding the host root filesystem: /host/root
// when the container mounts it, / on a native install.
func hostRootPath() (string, bool) {
	if _, err := os.Stat("/host/root"); err == nil {
		return "/host/root", true
	}
	if !inContainer() {
		return "/", true
	}
	return "", false
}

// ReadHostMetrics reads host-level metrics. ok is false when the host /proc is
// not mounted, so callers never present container values as host values. Only
// meaningful for the local host.
func ReadHostMetrics() (HostMetrics, bool) {
	proc, ok := hostProcRoot()
	if !ok {
		return HostMetrics{}, false
	}

	var m HostMetrics
	if l1, l5, l15, err := readLoadAvg(proc + "/loadavg"); err == nil {
		m.Load1, m.Load5, m.Load15 = l1, l5, l15
	}
	if up, err := readUptime(proc + "/uptime"); err == nil {
		m.Uptime = up
	}
	if used, err := readMemUsed(proc + "/meminfo"); err == nil {
		m.MemUsed = used
	}
	// /proc/net/dev reflects the reader's network namespace, so mounting the
	// host proc is not enough. /proc/<pid>/net/dev is bound to that process's
	// namespace, so pid 1 on the mounted host proc gives the host's interfaces.
	if rx, tx, err := readNetDev(proc + "/1/net/dev"); err == nil {
		m.NetRxTotal, m.NetTxTotal = rx, tx
	}
	if root, ok := hostRootPath(); ok {
		if total, free, err := statfs(root); err == nil {
			m.DiskTotal, m.DiskFree = total, free
		}
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

func readMemUsed(path string) (uint64, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	var total, available uint64
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		switch {
		case strings.HasPrefix(line, "MemTotal:"):
			total = parseKBLine(line)
		case strings.HasPrefix(line, "MemAvailable:"):
			available = parseKBLine(line)
		}
	}
	if err := sc.Err(); err != nil {
		return 0, err
	}
	if total == 0 {
		return 0, errShort
	}
	if available > total {
		available = total
	}
	return total - available, nil
}

// parseKBLine extracts the kB value from a /proc/meminfo line like
// "MemTotal:       16384000 kB".
func parseKBLine(line string) uint64 {
	f := strings.Fields(line)
	if len(f) < 2 {
		return 0
	}
	kb, err := strconv.ParseUint(f[1], 10, 64)
	if err != nil {
		return 0
	}
	return kb * 1024
}

// readNetDev sums rx/tx bytes across all non-loopback interfaces in /proc/net/dev.
func readNetDev(path string) (uint64, uint64, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()

	var rx, tx uint64
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		i := strings.Index(line, ":")
		if i < 0 {
			continue
		}
		iface := strings.TrimSpace(line[:i])
		if iface == "lo" {
			continue
		}
		fields := strings.Fields(line[i+1:])
		if len(fields) < 9 {
			continue
		}
		r, _ := strconv.ParseUint(fields[0], 10, 64)
		t, _ := strconv.ParseUint(fields[8], 10, 64)
		rx += r
		tx += t
	}
	if err := sc.Err(); err != nil {
		return 0, 0, err
	}
	return rx, tx, nil
}

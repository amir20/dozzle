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

// hostProcRoot returns the directory holding the host /proc. When the container
// mounts the host proc at /host/proc we use it; otherwise we fall back to /proc
// (which yields the container's own values).
func hostProcRoot() string {
	for _, p := range []string{"/host/proc", "/proc"} {
		if _, err := os.Stat(p + "/loadavg"); err == nil {
			return p
		}
	}
	return "/proc"
}

// hostRootPath returns the directory holding the host root filesystem, used for
// the disk metric.
func hostRootPath() string {
	for _, p := range []string{"/host/root", "/host", "/"} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return "/"
}

// ReadHostMetrics reads host-level metrics. ok is false when nothing could be
// read. Only meaningful for the local host.
func ReadHostMetrics() (HostMetrics, bool) {
	var m HostMetrics
	ok := false
	proc := hostProcRoot()

	if l1, l5, l15, err := readLoadAvg(proc + "/loadavg"); err == nil {
		m.Load1, m.Load5, m.Load15 = l1, l5, l15
		ok = true
	}
	if up, err := readUptime(proc + "/uptime"); err == nil {
		m.Uptime = up
		ok = true
	}
	if used, err := readMemUsed(proc + "/meminfo"); err == nil {
		m.MemUsed = used
		ok = true
	}
	if rx, tx, err := readNetDev(proc + "/net/dev"); err == nil {
		m.NetRxTotal, m.NetTxTotal = rx, tx
		ok = true
	}
	if total, free, err := statfs(hostRootPath()); err == nil {
		m.DiskTotal, m.DiskFree = total, free
		ok = true
	}
	return m, ok
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

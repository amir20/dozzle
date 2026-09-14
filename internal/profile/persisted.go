package profile

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

var persisted = sync.OnceValue(func() bool {
	// Outside a container ./data is a directory on the host's own disk, so it
	// already outlives the process.
	if _, err := os.Stat("/.dockerenv"); err != nil {
		return true
	}
	f, err := os.Open("/proc/self/mountinfo")
	if err != nil {
		// Unknown is treated as persisted so nobody is told to fix a setup that works.
		return true
	}
	defer f.Close()
	return isMounted(f, dataPath)
})

// Persisted reports whether the data directory survives the container being
// recreated, which is only true when something is mounted at or above it.
func Persisted() bool {
	return persisted()
}

// isMounted reads a mountinfo table and reports whether path is a mount point or
// sits inside one. The root mount is the container's own writable layer, so it
// does not count.
func isMounted(r io.Reader, path string) bool {
	path = filepath.Clean(path)
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		// Fields: id parent major:minor root mountpoint options ...
		fields := strings.Fields(scanner.Text())
		if len(fields) < 5 {
			continue
		}
		mountPoint := unescapeMountInfo(fields[4])
		if mountPoint == "/" {
			continue
		}
		if path == mountPoint || strings.HasPrefix(path, mountPoint+"/") {
			return true
		}
	}
	return false
}

// mountinfo escapes space, tab, newline and backslash as three-digit octal.
func unescapeMountInfo(s string) string {
	if !strings.Contains(s, `\`) {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+3 < len(s) {
			if n, err := strconv.ParseUint(s[i+1:i+4], 8, 8); err == nil {
				b.WriteByte(byte(n))
				i += 3
				continue
			}
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

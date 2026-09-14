package profile

import (
	"bufio"
	"io"
	"os"
	"regexp"
	"strings"
	"sync"
)

// Docker bind-mounts /etc/hostname, /etc/hosts and /etc/resolv.conf from the
// container's own directory, so its id shows up in the root field of mountinfo.
var containerIDInRoot = regexp.MustCompile(`/containers/([0-9a-f]{64})/`)

var selfContainerID = sync.OnceValue(func() string {
	f, err := os.Open("/proc/self/mountinfo")
	if err != nil {
		return ""
	}
	defer f.Close()
	return parseSelfContainerID(f)
})

// SelfContainerID is the id of the container this process runs in, or "" when
// it cannot be found (not in a container, or not under Docker).
func SelfContainerID() string {
	return selfContainerID()
}

func parseSelfContainerID(r io.Reader) string {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		// Fields: id parent major:minor root mountpoint options ...
		fields := strings.Fields(scanner.Text())
		if len(fields) < 5 {
			continue
		}
		if m := containerIDInRoot.FindStringSubmatch(fields[3]); m != nil {
			return m[1]
		}
	}
	return ""
}

package histogram

import (
	"strings"
	"time"
)

// lineSeconds returns the Unix second of the RFC 3339 timestamp a log line
// starts with, as Docker and the kubelet write it.
func lineSeconds(line string) (int64, bool) {
	before, _, ok := strings.Cut(line, " ")
	if !ok {
		return 0, false
	}
	t, err := time.Parse(time.RFC3339Nano, before)
	if err != nil {
		return 0, false
	}
	return t.Unix(), true
}

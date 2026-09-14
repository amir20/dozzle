package profile

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const unmounted = `591 540 0:52 / / rw,relatime master:245 - overlay overlay rw,lowerdir=/var/lib/docker/overlay2/l/A
592 591 0:55 / /proc rw,nosuid,nodev,noexec,relatime - proc proc rw
597 591 254:1 /docker/containers/abc/resolv.conf /etc/resolv.conf rw,relatime - ext4 /dev/vda1 rw
`

func TestIsMounted(t *testing.T) {
	tests := []struct {
		name  string
		table string
		path  string
		want  bool
	}{
		{"only the container layer", unmounted, "/data", false},
		{"volume at data", unmounted + "598 591 254:1 /volumes/dozzle/_data /data rw,relatime - ext4 /dev/vda1 rw\n", "/data", true},
		{"bind mount above data", unmounted + "598 591 254:1 /srv /app rw,relatime - ext4 /dev/vda1 rw\n", "/app/data", true},
		{"sibling with a shared prefix", unmounted + "598 591 254:1 /srv /dat rw,relatime - ext4 /dev/vda1 rw\n", "/data", false},
		{"escaped space in mount point", unmounted + `598 591 254:1 /srv /my\040data rw - ext4 /dev/vda1 rw` + "\n", "/my data", true},
		{"malformed line", "garbage\n", "/data", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isMounted(strings.NewReader(tt.table), tt.path))
		})
	}
}

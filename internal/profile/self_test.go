package profile

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const sampleID = "4f1c9b2e8d7a6b5c4d3e2f1a0b9c8d7e6f5a4b3c2d1e0f9a8b7c6d5e4f3a2b1c"

func TestParseSelfContainerID(t *testing.T) {
	mountinfo := strings.Join([]string{
		"1024 980 0:52 / / rw,relatime master:1 - overlay overlay rw,lowerdir=/var/lib/docker/overlay2/l/ABC",
		"1025 1024 0:55 / /proc rw,nosuid,nodev,noexec,relatime - proc proc rw",
		"1040 1024 254:1 /docker/volumes/dozzle_data/_data /data rw,relatime - ext4 /dev/vda1 rw",
		"1041 1024 254:1 /docker/containers/" + sampleID + "/resolv.conf /etc/resolv.conf rw,relatime - ext4 /dev/vda1 rw",
		"1042 1024 254:1 /docker/containers/" + sampleID + "/hostname /etc/hostname rw,relatime - ext4 /dev/vda1 rw",
	}, "\n")
	assert.Equal(t, sampleID, parseSelfContainerID(strings.NewReader(mountinfo)))
}

func TestParseSelfContainerID_NotFound(t *testing.T) {
	mountinfo := strings.Join([]string{
		"22 1 8:1 / / rw,relatime shared:1 - ext4 /dev/sda1 rw",
		"23 22 0:21 / /proc rw,nosuid - proc proc rw",
		// Too short to be an id.
		"24 22 8:1 /containers/abc123/hostname /etc/hostname rw - ext4 /dev/sda1 rw",
		"garbage",
	}, "\n")
	assert.Equal(t, "", parseSelfContainerID(strings.NewReader(mountinfo)))
}

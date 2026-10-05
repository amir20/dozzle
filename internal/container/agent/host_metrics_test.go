package agent

import (
	"testing"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/container/agent/pb"
	"github.com/stretchr/testify/assert"
)

// What the agent reads off its machine has to arrive intact on the server side.
func TestHostMetricsRoundTrip(t *testing.T) {
	host := container.Host{
		MetricsAvailable: true,
		Load1:            1.5,
		Load5:            1.2,
		Load15:           0.9,
		Uptime:           3600,
		DiskTotal:        1000,
		DiskFree:         400,
		Disks:            []container.Disk{{Name: "media", Total: 2000, Free: 100}},
		Reclaimable:      &container.Reclaimable{Images: 1, ImagesSize: 2, Volumes: 3, VolumesSize: 4, Containers: 5, ContainersSize: 6, BuildCacheSize: 7},
	}

	var out pb.Host
	setHostMetricsProto(&out, host)
	m, ok := hostMetricsFromProto(&out)

	assert.True(t, ok)
	assert.Equal(t, container.HostMetrics{
		Load1: 1.5, Load5: 1.2, Load15: 0.9, Uptime: 3600, DiskTotal: 1000, DiskFree: 400,
		Disks:       []container.Disk{{Name: "media", Total: 2000, Free: 100}},
		Reclaimable: host.Reclaimable,
	}, m)
}

// An agent built before these fields sends none of them, which must read as a
// host with nothing to show rather than an idle one.
func TestHostMetricsFromOldAgent(t *testing.T) {
	m, ok := hostMetricsFromProto(&pb.Host{Id: "abc", CpuCores: 4})
	assert.False(t, ok)
	assert.Zero(t, m.DiskTotal)
	assert.Empty(t, m.Disks)
	assert.Nil(t, m.Reclaimable, "unknown, not zero")
}

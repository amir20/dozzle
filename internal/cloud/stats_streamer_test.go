package cloud

import (
	"context"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/container"
	container_support "github.com/amir20/dozzle/internal/support/container"
	pb "github.com/amir20/dozzle/proto/cloud"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeStatsHostService is a StatsStreamHostService returning a scripted
// container/host snapshot. SubscribeStats is unused by the tests below, which
// drive observe/flush directly rather than waiting on the 30s ticker.
type fakeStatsHostService struct {
	containers []container.Container
	hosts      []container.Host
}

func (f *fakeStatsHostService) ListAllContainers(_ container.ContainerLabels) ([]container.Container, []error) {
	return f.containers, nil
}

func (f *fakeStatsHostService) FindContainer(_ string, _ string, _ container.ContainerLabels) (*container_support.ContainerService, error) {
	return nil, nil
}

func (f *fakeStatsHostService) Hosts() []container.Host { return f.hosts }

func (f *fakeStatsHostService) SubscribeStats(_ context.Context, _ chan<- StatSample) {}

// newTestStreamer builds a streamer over one host with the given containers and
// a capture func standing in for the gRPC send.
func newTestStreamer(t *testing.T, containers []container.Container, ncpu int) (*statsStreamer, *[]*pb.StatsBatch) {
	t.Helper()
	hs := &fakeStatsHostService{
		containers: containers,
		hosts:      []container.Host{{ID: "host1", Name: "host1", NCPU: ncpu}},
	}
	var sent []*pb.StatsBatch
	ss := newStatsStreamer(hs, nil, func(resp *pb.ToolResponse) error {
		sent = append(sent, resp.GetStatsBatch())
		return nil
	})
	ss.refreshMeta()
	return ss, &sent
}

func testContainer(id, name string) container.Container {
	return container.Container{ID: id, Name: name, Host: "host1", Labels: map[string]string{}}
}

func TestStatsStreamer_AveragesGaugesAndKeepsLastCounters(t *testing.T) {
	ss, sent := newTestStreamer(t, []container.Container{testContainer("c1", "api")}, 1)

	for i, cpu := range []float64{10, 20, 60} {
		ss.observe(StatSample{HostID: "host1", Stat: container.ContainerStat{
			ID:             "c1",
			CPUPercent:     cpu,
			MemoryPercent:  cpu / 2,
			MemoryUsage:    float64(100 * (i + 1)),
			NetworkRxTotal: uint64(1000 * (i + 1)),
			NetworkTxTotal: uint64(2000 * (i + 1)),
			DiskReadTotal:  uint64(10 * (i + 1)),
			DiskWriteTotal: uint64(20 * (i + 1)),
		}})
	}

	now := time.Unix(1700000000, 0)
	require.NoError(t, ss.flush(now))

	require.Len(t, *sent, 1)
	entries := (*sent)[0].GetEntries()
	require.Len(t, entries, 1)
	e := entries[0]

	assert.Equal(t, "host1", e.GetHostId())
	assert.Equal(t, "api", e.GetContainerName())
	assert.Equal(t, now.UnixNano(), e.GetTimestampNs())

	assert.InDelta(t, 30.0, e.GetCpuPercent(), 0.001, "mean of 10/20/60")
	assert.InDelta(t, 60.0, e.GetCpuPercentMax(), 0.001)
	assert.InDelta(t, 15.0, e.GetMemoryPercent(), 0.001)
	assert.InDelta(t, 30.0, e.GetMemoryPercentMax(), 0.001)
	assert.InDelta(t, 200.0, e.GetMemoryUsageBytes(), 0.001, "mean of 100/200/300")

	// Counters are cumulative — the newest value, not a sum or a delta.
	assert.Equal(t, uint64(3000), e.GetNetworkRxTotal())
	assert.Equal(t, uint64(6000), e.GetNetworkTxTotal())
	assert.Equal(t, uint64(30), e.GetDiskReadTotal())
	assert.Equal(t, uint64(60), e.GetDiskWriteTotal())

	assert.Equal(t, uint32(3), e.GetSamples())
	assert.InDelta(t, 1.0, e.GetCpuCores(), 0.001)
}

func TestStatsStreamer_NormalisesCPUByCores(t *testing.T) {
	t.Run("falls back to host NCPU", func(t *testing.T) {
		ss, sent := newTestStreamer(t, []container.Container{testContainer("c1", "api")}, 4)
		ss.observe(StatSample{HostID: "host1", Stat: container.ContainerStat{ID: "c1", CPUPercent: 200}})
		require.NoError(t, ss.flush(time.Unix(1, 0)))

		e := (*sent)[0].GetEntries()[0]
		assert.InDelta(t, 50.0, e.GetCpuPercent(), 0.001, "200%% of one core across 4 cores is 50%% overall")
		assert.InDelta(t, 50.0, e.GetCpuPercentMax(), 0.001)
		assert.InDelta(t, 4.0, e.GetCpuCores(), 0.001)
	})

	t.Run("prefers the container CPU limit", func(t *testing.T) {
		c := testContainer("c1", "api")
		c.CPULimit = 2
		ss, sent := newTestStreamer(t, []container.Container{c}, 8)
		ss.observe(StatSample{HostID: "host1", Stat: container.ContainerStat{ID: "c1", CPUPercent: 100}})
		require.NoError(t, ss.flush(time.Unix(1, 0)))

		e := (*sent)[0].GetEntries()[0]
		assert.InDelta(t, 50.0, e.GetCpuPercent(), 0.001)
		assert.InDelta(t, 2.0, e.GetCpuCores(), 0.001)
	})

	t.Run("defaults to one core when nothing is known", func(t *testing.T) {
		ss, sent := newTestStreamer(t, []container.Container{testContainer("c1", "api")}, 0)
		ss.observe(StatSample{HostID: "host1", Stat: container.ContainerStat{ID: "c1", CPUPercent: 37}})
		require.NoError(t, ss.flush(time.Unix(1, 0)))

		assert.InDelta(t, 37.0, (*sent)[0].GetEntries()[0].GetCpuPercent(), 0.001)
	})
}

func TestStatsStreamer_ResetsBetweenWindows(t *testing.T) {
	ss, sent := newTestStreamer(t, []container.Container{testContainer("c1", "api")}, 1)

	ss.observe(StatSample{HostID: "host1", Stat: container.ContainerStat{ID: "c1", CPUPercent: 80}})
	require.NoError(t, ss.flush(time.Unix(1, 0)))
	assert.InDelta(t, 80.0, (*sent)[0].GetEntries()[0].GetCpuPercent(), 0.001)

	// Second window sees a single low sample — if state leaked the mean would
	// still be dragged up by the previous window's 80.
	ss.observe(StatSample{HostID: "host1", Stat: container.ContainerStat{ID: "c1", CPUPercent: 10}})
	require.NoError(t, ss.flush(time.Unix(2, 0)))
	require.Len(t, *sent, 2)
	assert.InDelta(t, 10.0, (*sent)[1].GetEntries()[0].GetCpuPercent(), 0.001)
}

func TestStatsStreamer_EmptyWindowSendsNothing(t *testing.T) {
	ss, sent := newTestStreamer(t, []container.Container{testContainer("c1", "api")}, 1)

	require.NoError(t, ss.flush(time.Unix(1, 0)))
	assert.Empty(t, *sent, "a window with no samples must not push an empty batch")

	// A container that stops simply stops producing samples, so it drops out.
	ss.observe(StatSample{HostID: "host1", Stat: container.ContainerStat{ID: "c1", CPUPercent: 5}})
	require.NoError(t, ss.flush(time.Unix(2, 0)))
	require.Len(t, *sent, 1)
	require.NoError(t, ss.flush(time.Unix(3, 0)))
	assert.Len(t, *sent, 1, "no new batch after the container went quiet")
}

func TestStatsStreamer_SkipsDisabledContainers(t *testing.T) {
	disabled := testContainer("c1", "secret")
	disabled.Labels[cloudMinLevelLabel] = "disabled"
	ss, sent := newTestStreamer(t, []container.Container{disabled, testContainer("c2", "api")}, 1)

	ss.observe(StatSample{HostID: "host1", Stat: container.ContainerStat{ID: "c1", CPUPercent: 90}})
	ss.observe(StatSample{HostID: "host1", Stat: container.ContainerStat{ID: "c2", CPUPercent: 10}})
	require.NoError(t, ss.flush(time.Unix(1, 0)))

	require.Len(t, *sent, 1)
	entries := (*sent)[0].GetEntries()
	require.Len(t, entries, 1)
	assert.Equal(t, "api", entries[0].GetContainerName())
}

func TestStatsStreamer_HoldsThenDropsUnresolvedContainers(t *testing.T) {
	// No containers in the snapshot at all — nothing can resolve to a name.
	ss, sent := newTestStreamer(t, nil, 1)

	ss.observe(StatSample{HostID: "host1", Stat: container.ContainerStat{ID: "ghost", CPUPercent: 50}})
	require.NoError(t, ss.flush(time.Unix(1, 0)))
	assert.Empty(t, *sent)
	assert.Len(t, ss.acc, 1, "held for one grace window in case metadata catches up")

	require.NoError(t, ss.flush(time.Unix(2, 0)))
	assert.Empty(t, *sent)
	assert.Empty(t, ss.acc, "abandoned after the grace window rather than leaking")
}

func TestStatsStreamer_EmitsOnceMetadataCatchesUp(t *testing.T) {
	hs := &fakeStatsHostService{hosts: []container.Host{{ID: "host1", NCPU: 1}}}
	var sent []*pb.StatsBatch
	ss := newStatsStreamer(hs, nil, func(resp *pb.ToolResponse) error {
		sent = append(sent, resp.GetStatsBatch())
		return nil
	})
	ss.refreshMeta()

	// Container started mid-window: samples arrive before it shows up in a listing.
	ss.observe(StatSample{HostID: "host1", Stat: container.ContainerStat{ID: "c1", CPUPercent: 40}})
	require.NoError(t, ss.flush(time.Unix(1, 0)))
	assert.Empty(t, sent)

	hs.containers = []container.Container{testContainer("c1", "late-starter")}
	ss.observe(StatSample{HostID: "host1", Stat: container.ContainerStat{ID: "c1", CPUPercent: 60}})
	require.NoError(t, ss.flush(time.Unix(2, 0)))

	require.Len(t, sent, 1)
	e := sent[0].GetEntries()[0]
	assert.Equal(t, "late-starter", e.GetContainerName())
	assert.InDelta(t, 50.0, e.GetCpuPercent(), 0.001, "both windows' samples are folded in")
	assert.Equal(t, uint32(2), e.GetSamples())
}

func TestStatsStreamer_SeparatesSameNameOnDifferentHosts(t *testing.T) {
	hs := &fakeStatsHostService{
		containers: []container.Container{
			{ID: "c1", Name: "api", Host: "host1", Labels: map[string]string{}},
			{ID: "c2", Name: "api", Host: "host2", Labels: map[string]string{}},
		},
		hosts: []container.Host{{ID: "host1", NCPU: 1}, {ID: "host2", NCPU: 1}},
	}
	var sent []*pb.StatsBatch
	ss := newStatsStreamer(hs, nil, func(resp *pb.ToolResponse) error {
		sent = append(sent, resp.GetStatsBatch())
		return nil
	})
	ss.refreshMeta()

	ss.observe(StatSample{HostID: "host1", Stat: container.ContainerStat{ID: "c1", CPUPercent: 10}})
	ss.observe(StatSample{HostID: "host2", Stat: container.ContainerStat{ID: "c2", CPUPercent: 90}})
	require.NoError(t, ss.flush(time.Unix(1, 0)))

	entries := sent[0].GetEntries()
	require.Len(t, entries, 2)
	// Sorted by (host_id, container_name).
	assert.Equal(t, "host1", entries[0].GetHostId())
	assert.InDelta(t, 10.0, entries[0].GetCpuPercent(), 0.001)
	assert.Equal(t, "host2", entries[1].GetHostId())
	assert.InDelta(t, 90.0, entries[1].GetCpuPercent(), 0.001)
}

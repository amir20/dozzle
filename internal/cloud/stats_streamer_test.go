package cloud

import (
	"context"
	"sync"
	"sync/atomic"
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
	// subscribed, when non-nil, hands the test the channel run() is draining.
	subscribed chan chan<- StatSample
}

func (f *fakeStatsHostService) ListAllContainers(_ container.ContainerLabels) ([]container.Container, []error) {
	return f.containers, nil
}

func (f *fakeStatsHostService) FindContainer(_ string, _ string, _ container.ContainerLabels) (*container_support.ContainerService, error) {
	return nil, nil
}

func (f *fakeStatsHostService) Hosts() []container.Host { return f.hosts }

func (f *fakeStatsHostService) SubscribeStats(_ context.Context, samples chan<- StatSample) {
	if f.subscribed != nil {
		f.subscribed <- samples
	}
}

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
	ss.refreshMeta() // stands in for the async refresh run() kicks off each tick
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

// A container redeployed inside one window keeps its name and gets a new ID, so
// the dying and the starting container hold two accumulators that resolve to the
// same series. They have to collapse into one entry — two points on the same
// (host, name) series at the same timestamp is not something the wire format can
// express, since StatsBatchEntry carries no container ID.
func TestStatsStreamer_MergesRedeployedContainerWithinAWindow(t *testing.T) {
	ss, sent := newTestStreamer(t, []container.Container{
		testContainer("old", "api"),
		testContainer("new", "api"),
	}, 1)

	// Old container: 3 samples, high counters from a long uptime.
	for _, cpu := range []float64{10, 20, 30} {
		ss.observe(StatSample{HostID: "host1", Stat: container.ContainerStat{
			ID: "old", CPUPercent: cpu, MemoryPercent: 40, MemoryUsage: 100,
			NetworkRxTotal: 9000, DiskReadTotal: 900,
		}})
	}
	// Replacement: 1 sample, counters back near zero.
	ss.observe(StatSample{HostID: "host1", Stat: container.ContainerStat{
		ID: "new", CPUPercent: 80, MemoryPercent: 80, MemoryUsage: 300,
		NetworkRxTotal: 5, DiskReadTotal: 1,
	}})

	require.NoError(t, ss.flush(time.Unix(1, 0)))

	entries := (*sent)[0].GetEntries()
	require.Len(t, entries, 1, "one series, not two points at the same timestamp")
	e := entries[0]

	assert.Equal(t, "api", e.GetContainerName())
	assert.Equal(t, uint32(4), e.GetSamples())
	// Means weighted by sample count, not a flat average of the two entries.
	assert.InDelta(t, 35.0, e.GetCpuPercent(), 0.001, "(10+20+30+80)/4")
	assert.InDelta(t, 50.0, e.GetMemoryPercent(), 0.001, "(40*3+80)/4")
	assert.InDelta(t, 150.0, e.GetMemoryUsageBytes(), 0.001, "(100*3+300)/4")
	assert.InDelta(t, 80.0, e.GetCpuPercentMax(), 0.001)
	assert.InDelta(t, 80.0, e.GetMemoryPercentMax(), 0.001)
	// Counters take the larger: the series was already sitting at the old
	// container's total, and summing would report a value neither ever had.
	assert.Equal(t, uint64(9000), e.GetNetworkRxTotal())
	assert.Equal(t, uint64(900), e.GetDiskReadTotal())
}

// slowStatsHostService lets the first N ListAllContainers calls through, then
// blocks — standing in for a multi-host setup where one peer goes unreachable
// mid-run (each host costs a 5s timeout, sequentially).
type slowStatsHostService struct {
	fakeStatsHostService
	blockAfter int32
	calls      atomic.Int32
	entered    chan struct{}
	release    chan struct{}
	once       sync.Once
}

func (f *slowStatsHostService) ListAllContainers(l container.ContainerLabels) ([]container.Container, []error) {
	if f.calls.Add(1) > f.blockAfter {
		f.once.Do(func() { close(f.entered) })
		<-f.release
	}
	return f.fakeStatsHostService.ListAllContainers(l)
}

// A slow host must not stall the sample-draining loop.
//
// flush() used to call ListAllContainers inline, on the same goroutine that
// drains samples. A hung host would block that drain; once the sample buffer
// filled, backpressure reached the stats collector's blocking per-subscriber
// dispatch and starved the LIVE UI stats broadcast on that host. Metrics
// shipping must never be able to degrade the UI.
//
// The window is driven short here so flush() actually runs — that is where the
// original bug lived.
func TestStatsStreamer_SlowHostDoesNotStallSampleDrain(t *testing.T) {
	hs := &slowStatsHostService{
		containers: []container.Container{testContainer("c1", "api")},
		hosts:      []container.Host{{ID: "host1", NCPU: 1}},
		subscribed: make(chan chan<- StatSample, 1),
		blockAfter: 1, // let the startup refresh through, wedge the flush-time one
		entered:    make(chan struct{}),
		release:    make(chan struct{}),
	}
	ss := newStatsStreamer(hs, nil, func(*pb.ToolResponse) error { return nil })
	ss.window = 20 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() { defer close(done); ss.run(ctx) }()

	samples := <-hs.subscribed

	// Wait for a refresh to wedge, which only happens once the ticker has fired.
	select {
	case <-hs.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("host never blocked; test did not reach the interesting state")
	}

	// With a refresh wedged, the loop must still drain. Push well past the
	// buffer: if the drain were blocked behind the host call this never finishes.
	drained := make(chan struct{})
	go func() {
		defer close(drained)
		for range statsSampleChanBuf * 3 {
			select {
			case samples <- StatSample{HostID: "host1", Stat: container.ContainerStat{ID: "c1", CPUPercent: 1}}:
			case <-ctx.Done():
				return
			}
		}
	}()

	select {
	case <-drained:
	case <-time.After(5 * time.Second):
		t.Fatal("sample drain stalled behind a slow host — backpressure would reach the stats collector and starve the live UI stats broadcast")
	}

	close(hs.release)
	cancel()
	<-done
}

// The refreshed snapshot must actually be installed by the run loop — if it
// weren't, names would never resolve and every window would be dropped. Asserted
// through run()'s real ticker on observable output, rather than by reading
// ss.meta, which the run goroutine owns.
func TestStatsStreamer_AsyncRefreshInstallsSnapshot(t *testing.T) {
	hs := &fakeStatsHostService{
		containers: []container.Container{testContainer("c1", "api")},
		hosts:      []container.Host{{ID: "host1", NCPU: 1}},
		subscribed: make(chan chan<- StatSample, 1),
	}
	batches := make(chan *pb.StatsBatch, 4)
	ss := newStatsStreamer(hs, nil, func(r *pb.ToolResponse) error {
		batches <- r.GetStatsBatch()
		return nil
	})
	ss.window = 50 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() { defer close(done); ss.run(ctx) }()

	samples := <-hs.subscribed
	samples <- StatSample{HostID: "host1", Stat: container.ContainerStat{ID: "c1", CPUPercent: 42}}

	select {
	case b := <-batches:
		require.Len(t, b.GetEntries(), 1)
		assert.Equal(t, "api", b.GetEntries()[0].GetContainerName(),
			"name resolved, so the async snapshot was installed")
		assert.InDelta(t, 42.0, b.GetEntries()[0].GetCpuPercent(), 0.001)
	case <-time.After(5 * time.Second):
		t.Fatal("no batch emitted — async refresh never installed its snapshot")
	}

	cancel()
	<-done
}

// A refresh slower than the window must not queue up more refreshes.
func TestStatsStreamer_OnlyOneRefreshInFlight(t *testing.T) {
	hs := &slowStatsHostService{
		hosts:      []container.Host{{ID: "host1", NCPU: 1}},
		blockAfter: 0, // block immediately
		entered:    make(chan struct{}),
		release:    make(chan struct{}),
	}
	ss := newStatsStreamer(hs, nil, func(*pb.ToolResponse) error { return nil })
	ctx := t.Context()

	ss.startMetaRefresh(ctx)
	<-hs.entered
	// Every subsequent attempt while one is in flight is a no-op.
	for range 5 {
		ss.startMetaRefresh(ctx)
	}
	assert.True(t, ss.refreshing, "should still be marked in-flight")

	close(hs.release)
	select {
	case m := <-ss.metaCh:
		assert.NotNil(t, m, "exactly one snapshot delivered")
	case <-time.After(2 * time.Second):
		t.Fatal("refresh never completed")
	}
	// Nothing else queued behind it.
	select {
	case <-ss.metaCh:
		t.Fatal("a second refresh was queued while one was in flight")
	case <-time.After(100 * time.Millisecond):
	}
}

// A failed send must not end the streamer. It used to `return`, so a single
// transient error meant the instance reported no metrics at all until it
// reconnected — the failure mode that made a live Cloud instance look idle for
// hours.
func TestStatsStreamer_SurvivesAFailedSend(t *testing.T) {
	hs := &fakeStatsHostService{
		containers: []container.Container{testContainer("c1", "api")},
		hosts:      []container.Host{{ID: "host1", NCPU: 1}},
		subscribed: make(chan chan<- StatSample, 1),
	}
	var failNext atomic.Bool
	failNext.Store(true)
	batches := make(chan *pb.StatsBatch, 4)
	ss := newStatsStreamer(hs, nil, func(r *pb.ToolResponse) error {
		if failNext.Swap(false) {
			return assert.AnError
		}
		batches <- r.GetStatsBatch()
		return nil
	})
	ss.window = 50 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() { defer close(done); ss.run(ctx) }()

	samples := <-hs.subscribed
	// Two windows' worth: the first send fails, the second must still happen.
	samples <- StatSample{HostID: "host1", Stat: container.ContainerStat{ID: "c1", CPUPercent: 10}}
	time.Sleep(80 * time.Millisecond)
	samples <- StatSample{HostID: "host1", Stat: container.ContainerStat{ID: "c1", CPUPercent: 20}}

	select {
	case b := <-batches:
		require.Len(t, b.GetEntries(), 1)
		assert.Equal(t, "api", b.GetEntries()[0].GetContainerName())
	case <-time.After(5 * time.Second):
		t.Fatal("streamer stopped after one failed send")
	}

	cancel()
	<-done
}

// ...but a stream that fails every single time is wedged, and spinning on it
// forever helps nobody. It gives up after statsMaxSendFailures.
func TestStatsStreamer_GivesUpOnAPermanentlyFailingSend(t *testing.T) {
	hs := &fakeStatsHostService{
		containers: []container.Container{testContainer("c1", "api")},
		hosts:      []container.Host{{ID: "host1", NCPU: 1}},
		subscribed: make(chan chan<- StatSample, 1),
	}
	var sends atomic.Int32
	ss := newStatsStreamer(hs, nil, func(*pb.ToolResponse) error {
		sends.Add(1)
		return assert.AnError
	})
	ss.window = 20 * time.Millisecond

	ctx := t.Context()
	done := make(chan struct{})
	go func() { defer close(done); ss.run(ctx) }()

	samples := <-hs.subscribed
	// Keep feeding so every window has something to send; run() exits once the
	// failures reach the cap.
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case samples <- StatSample{HostID: "host1", Stat: container.ContainerStat{ID: "c1", CPUPercent: 1}}:
				time.Sleep(5 * time.Millisecond)
			}
		}
	}()

	select {
	case <-done:
		assert.Equal(t, int32(statsMaxSendFailures), sends.Load(),
			"should stop exactly at the failure cap")
	case <-time.After(5 * time.Second):
		t.Fatal("streamer never gave up on a permanently failing send")
	}
}

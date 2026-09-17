package web

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/config"
	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/imagecheck"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type autoUpdateRecorder struct {
	mu     sync.Mutex
	checks int
	starts int
	status imagecheck.Status
}

func stubAutoUpdate(t *testing.T, status imagecheck.Status) *autoUpdateRecorder {
	t.Helper()
	rec := &autoUpdateRecorder{status: status}
	oldCheck := selfUpdateCheck
	selfUpdateCheck = func(_ context.Context, image string, _ []string) imagecheck.Result {
		rec.mu.Lock()
		defer rec.mu.Unlock()
		rec.checks++
		return imagecheck.Result{Image: image, Status: rec.status}
	}
	stubSelfUpdateStart(t, func(_ context.Context, id string, _ func(container.UpdateProgress)) (bool, error) {
		rec.mu.Lock()
		defer rec.mu.Unlock()
		rec.starts++
		return true, nil
	})
	t.Cleanup(func() { selfUpdateCheck = oldCheck })
	return rec
}

func (r *autoUpdateRecorder) counts() (int, int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.checks, r.starts
}

func writeSchedule(t *testing.T, mode, at string) {
	t.Helper()
	require.NoError(t, config.Update(setupConfigPath, func(c *config.File) {
		c.AutoUpdate = &mode
		c.AutoUpdateTime = &at
	}))
}

func newTestScheduler(cfg Config) *autoUpdateScheduler {
	return &autoUpdateScheduler{config: &cfg, now: time.Now, after: time.After}
}

var serverActions = Config{Mode: "server", EnableActions: true}

// 2026-09-13 is a Sunday.
func at(day int, hhmm string) time.Time {
	t, err := time.ParseInLocation("2006-01-02 15:04:05", fmt.Sprintf("2026-09-%02d %s:20", day, hhmm), time.Local)
	if err != nil {
		panic(err)
	}
	return t
}

func TestAutoUpdate_SwarmPrimaryOnManagerUpdates(t *testing.T) {
	setupTestEnv(t, true)
	rec := stubAutoUpdate(t, imagecheck.StatusUpdateAvailable)
	writeSchedule(t, "daily", "03:00")
	selfUpdateInspect = func(context.Context, HostService, string) (selfImage, error) {
		return selfImage{Ref: "amir20/dozzle:master", Swarm: true, ServiceID: "svc"}, nil
	}

	newTestScheduler(serverActions).tick(context.Background(), at(14, "03:00"))
	checks, starts := rec.counts()
	assert.Equal(t, 1, checks)
	assert.Equal(t, 1, starts)
}

func TestAutoUpdate_DailyFiresOncePerDay(t *testing.T) {
	setupTestEnv(t, true)
	rec := stubAutoUpdate(t, imagecheck.StatusUpdateAvailable)
	writeSchedule(t, "daily", "03:00")
	s := newTestScheduler(serverActions)
	ctx := context.Background()

	s.tick(ctx, at(14, "02:59"))
	checks, starts := rec.counts()
	assert.Equal(t, 0, checks)
	assert.Equal(t, 0, starts)

	s.tick(ctx, at(14, "03:00"))
	s.tick(ctx, at(14, "03:00"))
	checks, starts = rec.counts()
	assert.Equal(t, 1, checks)
	assert.Equal(t, 1, starts)

	s.tick(ctx, at(15, "03:00"))
	checks, starts = rec.counts()
	assert.Equal(t, 2, checks)
	assert.Equal(t, 2, starts)
}

func TestAutoUpdate_WeeklyOnlySunday(t *testing.T) {
	setupTestEnv(t, true)
	rec := stubAutoUpdate(t, imagecheck.StatusUpdateAvailable)
	writeSchedule(t, "weekly", "04:30")
	s := newTestScheduler(serverActions)

	s.tick(context.Background(), at(14, "04:30")) // Monday
	_, starts := rec.counts()
	assert.Equal(t, 0, starts)

	s.tick(context.Background(), at(13, "04:30")) // Sunday
	_, starts = rec.counts()
	assert.Equal(t, 1, starts)
}

// A rolled-back update leaves the tag on the broken image, so the same remote
// digest keeps being offered. It must not be retried every schedule.
func TestAutoUpdate_DoesNotRetryRolledBackImage(t *testing.T) {
	setupTestEnv(t, true)
	rec := stubAutoUpdate(t, imagecheck.StatusUpdateAvailable)
	digest := "sha256:broken"
	oldCheck := selfUpdateCheck
	selfUpdateCheck = func(ctx context.Context, image string, digests []string) imagecheck.Result {
		r := oldCheck(ctx, image, digests)
		r.RemoteDigest = digest
		return r
	}
	t.Cleanup(func() { selfUpdateCheck = oldCheck })
	writeSchedule(t, "daily", "03:00")
	s := newTestScheduler(serverActions)
	ctx := context.Background()

	s.tick(ctx, at(14, "03:00"))
	s.tick(ctx, at(15, "03:00"))
	_, starts := rec.counts()
	assert.Equal(t, 1, starts, "the same digest is not attempted twice")

	digest = "sha256:fixed"
	s.tick(ctx, at(16, "03:00"))
	_, starts = rec.counts()
	assert.Equal(t, 2, starts, "a newly pushed image is tried")

	rec.mu.Lock()
	rec.status = imagecheck.StatusUpToDate
	rec.mu.Unlock()
	s.tick(ctx, at(17, "03:00"))
	rec.mu.Lock()
	rec.status = imagecheck.StatusUpdateAvailable
	rec.mu.Unlock()
	s.tick(ctx, at(18, "03:00"))
	_, starts = rec.counts()
	assert.Equal(t, 3, starts, "an up-to-date check clears the record")
}

func TestAutoUpdate_NoUpdateAvailableSkipsStart(t *testing.T) {
	setupTestEnv(t, true)
	rec := stubAutoUpdate(t, imagecheck.StatusUpToDate)
	writeSchedule(t, "daily", "03:00")

	newTestScheduler(serverActions).tick(context.Background(), at(14, "03:00"))
	checks, starts := rec.counts()
	assert.Equal(t, 1, checks)
	assert.Equal(t, 0, starts)
}

func TestAutoUpdate_OffAndUnsupported(t *testing.T) {
	setupTestEnv(t, true)
	rec := stubAutoUpdate(t, imagecheck.StatusUpdateAvailable)
	ctx := context.Background()

	// Nothing in the file means off.
	newTestScheduler(serverActions).tick(ctx, at(14, "03:00"))

	writeSchedule(t, "daily", "03:00")
	newTestScheduler(Config{Mode: "server"}).tick(ctx, at(14, "03:00"))
	newTestScheduler(Config{Mode: "swarm", EnableActions: true}).tick(ctx, at(14, "03:00"))

	selfUpdateInspect = func(context.Context, HostService, string) (selfImage, error) {
		return selfImage{Ref: "amir20/dozzle:v8.12.0"}, nil
	}
	newTestScheduler(serverActions).tick(ctx, at(14, "03:00"))

	// A swarm task on a worker cannot update the service.
	selfUpdateSwarmManager = func(context.Context, string) bool { return false }
	selfUpdateInspect = func(context.Context, HostService, string) (selfImage, error) {
		return selfImage{Ref: "amir20/dozzle:latest", Swarm: true, ServiceID: "svc"}, nil
	}
	newTestScheduler(serverActions).tick(ctx, at(14, "03:00"))

	// Nor does a replica other than the first, even on a manager.
	selfUpdateSwarmManager = func(context.Context, string) bool { return true }
	selfUpdateInspect = func(context.Context, HostService, string) (selfImage, error) {
		return selfImage{Ref: "amir20/dozzle:latest", Swarm: true, ServiceID: "svc", SecondaryReplica: true}, nil
	}
	newTestScheduler(serverActions).tick(ctx, at(14, "03:00"))

	setupSelfID = func() string { return "" }
	newTestScheduler(serverActions).tick(ctx, at(14, "03:00"))

	checks, starts := rec.counts()
	assert.Equal(t, 0, checks)
	assert.Equal(t, 0, starts)
}

func TestAutoUpdate_FileChangeAppliesLiveAndFlagsWin(t *testing.T) {
	setupTestEnv(t, true)
	rec := stubAutoUpdate(t, imagecheck.StatusUpdateAvailable)
	ctx := context.Background()
	s := newTestScheduler(serverActions)

	writeSchedule(t, "daily", "05:00")
	s.tick(ctx, at(14, "03:00"))
	writeSchedule(t, "daily", "03:00")
	s.tick(ctx, at(14, "03:00"))
	_, starts := rec.counts()
	assert.Equal(t, 1, starts)

	off := "off"
	locked := serverActions
	locked.Setup = SetupConfig{LockedAutoUpdate: true, AutoUpdateMode: &off}
	newTestScheduler(locked).tick(ctx, at(15, "03:00"))
	_, starts = rec.counts()
	assert.Equal(t, 1, starts)
}

func TestAutoUpdate_RunWakesEveryMinute(t *testing.T) {
	setupTestEnv(t, true)
	rec := stubAutoUpdate(t, imagecheck.StatusUpdateAvailable)
	writeSchedule(t, "daily", "03:00")

	var mu sync.Mutex
	clock := at(14, "02:58")
	var waits []time.Duration
	s := newTestScheduler(serverActions)
	s.now = func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		return clock
	}
	s.after = func(d time.Duration) <-chan time.Time {
		mu.Lock()
		defer mu.Unlock()
		waits = append(waits, d)
		clock = clock.Add(d)
		ch := make(chan time.Time, 1)
		ch <- clock
		return ch
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.run(ctx)
	}()
	require.Eventually(t, func() bool {
		_, starts := rec.counts()
		return starts == 1
	}, 2*time.Second, time.Millisecond)
	cancel()
	<-done

	mu.Lock()
	defer mu.Unlock()
	// 02:58:20 wakes at 02:59:01, then 03:00:01.
	assert.Equal(t, 41*time.Second, waits[0])
	assert.Equal(t, time.Minute, waits[1])
}

func TestAutoUpdate_Helpers(t *testing.T) {
	assert.True(t, pinnedReference("amir20/dozzle:v8.12.0"))
	assert.True(t, pinnedReference("amir20/dozzle:8.12.0"))
	assert.True(t, pinnedReference("amir20/dozzle@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"))
	assert.False(t, pinnedReference("amir20/dozzle:latest"))
	assert.False(t, pinnedReference("amir20/dozzle"))
	assert.False(t, pinnedReference("amir20/dozzle:v8"))
	assert.False(t, pinnedReference("localhost:5000/dozzle:beta"))

	sunday := at(13, "03:00")
	assert.True(t, autoUpdateSettings{Mode: "weekly", Time: "03:00"}.due(sunday))
	assert.False(t, autoUpdateSettings{Mode: "weekly", Time: "03:00"}.due(sunday.AddDate(0, 0, 1)))
	assert.True(t, autoUpdateSettings{Mode: "daily", Time: "03:00"}.due(sunday.AddDate(0, 0, 1)))
	assert.False(t, autoUpdateSettings{Mode: "off", Time: "03:00"}.due(sunday))
	assert.False(t, autoUpdateSettings{Mode: "daily", Time: "03:01"}.due(sunday))

	setupTestEnv(t, true)
	require.NoError(t, config.Update(setupConfigPath, func(c *config.File) {
		bad, badTime := "hourly", "9:00"
		c.AutoUpdate, c.AutoUpdateTime = &bad, &badTime
	}))
	settings, err := effectiveAutoUpdate(SetupConfig{})
	require.NoError(t, err)
	assert.Equal(t, autoUpdateSettings{Mode: "off", Time: "03:00"}, settings)
}

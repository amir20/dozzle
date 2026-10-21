package web

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/config"
	"github.com/amir20/dozzle/internal/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// pruneHosts stands in for MultiHostService: a few hosts, each answering a prune
// with what errs says, or freeing 100 bytes.
type pruneHosts struct {
	HostService
	hosts []container.Host
	errs  map[string]error

	mu     sync.Mutex
	pruned []string
	opts   []container.PruneOptions
}

func (p *pruneHosts) Hosts() []container.Host { return p.hosts }

func (p *pruneHosts) PruneHost(_ context.Context, host string, opts container.PruneOptions) (container.PruneResult, *container.Reclaimable, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.pruned = append(p.pruned, host)
	p.opts = append(p.opts, opts)
	if err := p.errs[host]; err != nil {
		return container.PruneResult{}, nil, err
	}
	return container.PruneResult{SpaceReclaimed: 100}, nil, nil
}

func writePruneSchedule(t *testing.T, mode, at string) {
	t.Helper()
	require.NoError(t, config.Update(setupConfigPath, func(c *config.File) {
		c.AutoPrune = &mode
		c.AutoPruneTime = &at
	}))
}

func resetAutoPruneLast(t *testing.T) {
	t.Cleanup(func() {
		autoPruneLast.Lock()
		autoPruneLast.run = nil
		autoPruneLast.Unlock()
	})
}

func TestAutoPrune_CleansEveryAvailableHostOnce(t *testing.T) {
	setupTestEnv(t, true)
	resetAutoPruneLast(t)
	writePruneSchedule(t, "daily", "04:00")
	hosts := &pruneHosts{
		hosts: []container.Host{
			{ID: "a", Name: "a", Available: true},
			{ID: "down", Name: "down"},
			{ID: "old", Name: "old", Available: true},
			{ID: "broken", Name: "broken", Available: true},
			{ID: "b", Name: "b", Available: true},
		},
		errs: map[string]error{
			"old":    errors.ErrUnsupported,
			"broken": errors.New("daemon said no"),
		},
	}
	s := &autoPruneScheduler{hostService: hosts, now: time.Now, after: time.After}

	s.tick(t.Context(), at(14, "03:59"))
	assert.Empty(t, hosts.pruned, "not due yet")

	s.tick(t.Context(), at(14, "04:00"))
	assert.Equal(t, []string{"a", "old", "broken", "b"}, hosts.pruned, "an unavailable host is skipped, a failing one does not stop the rest")
	for _, opts := range hosts.opts {
		assert.Equal(t, container.PruneOptions{Images: true, Containers: true, BuildCache: true}, opts, "never volumes")
	}
	run := lastAutoPrune()
	require.NotNil(t, run)
	assert.Equal(t, uint64(200), run.SpaceReclaimed)
	assert.Equal(t, 1, run.Failed, "a host too old to prune is not a failure")
	assert.Equal(t, at(14, "04:00"), run.At)

	s.tick(t.Context(), at(14, "04:00"))
	assert.Len(t, hosts.pruned, 4, "once a day")
}

func TestAutoPrune_OffAndWeekly(t *testing.T) {
	setupTestEnv(t, true)
	resetAutoPruneLast(t)
	hosts := &pruneHosts{hosts: []container.Host{{ID: "a", Available: true}}}
	s := &autoPruneScheduler{hostService: hosts, now: time.Now, after: time.After}

	s.tick(t.Context(), at(13, "04:00"))
	assert.Empty(t, hosts.pruned, "off by default")

	writePruneSchedule(t, "weekly", "04:00")
	s.tick(t.Context(), at(14, "04:00"))
	assert.Empty(t, hosts.pruned, "weekly skips a Monday")
	s.tick(t.Context(), at(20, "04:00"))
	assert.Equal(t, []string{"a"}, hosts.pruned, "and runs on Sunday")
}

func TestSetup_AutoPrunePatch(t *testing.T) {
	setupTestEnv(t, true)
	resetAutoPruneLast(t)
	h := setupNoneHandler(time.Now(), SetupConfig{})

	state := getSetupState(t, h)
	assert.Equal(t, "off", state.AutoPrune.Mode)
	assert.Equal(t, "04:00", state.AutoPrune.Time)
	assert.Nil(t, state.AutoPrune.LastRun)

	assert.Equal(t, http.StatusBadRequest, doSetup(h, "PATCH", "/api/setup/config", `{"autoPrune":{"mode":"hourly","time":"04:00"}}`).Code)
	assert.Equal(t, http.StatusBadRequest, doSetup(h, "PATCH", "/api/setup/config", `{"autoPrune":{"mode":"daily","time":"4:00"}}`).Code)
	require.Equal(t, http.StatusNoContent, doSetup(h, "PATCH", "/api/setup/config", `{"autoPrune":{"mode":"weekly","time":"05:00"}}`).Code)

	file, err := config.Load(setupConfigPath)
	require.NoError(t, err)
	assert.Equal(t, "weekly", *file.AutoPrune)
	assert.Equal(t, "05:00", *file.AutoPruneTime)
	assert.Nil(t, file.AutoUpdate, "the update schedule is left alone")

	autoPruneLast.Lock()
	autoPruneLast.run = &autoPruneRun{SpaceReclaimed: 42}
	autoPruneLast.Unlock()
	state = getSetupState(t, h)
	assert.Equal(t, "weekly", state.AutoPrune.Mode)
	assert.Equal(t, "05:00", state.AutoPrune.Time)
	assert.NotEmpty(t, state.AutoPrune.Zone)
	require.NotNil(t, state.AutoPrune.LastRun)
	assert.Equal(t, uint64(42), state.AutoPrune.LastRun.SpaceReclaimed)
}

func TestAutoPrune_FlagWinsOverFile(t *testing.T) {
	setupTestEnv(t, true)
	resetAutoPruneLast(t)
	writePruneSchedule(t, "off", "04:00")
	hosts := &pruneHosts{hosts: []container.Host{{ID: "a", Available: true}}}
	daily := "daily"
	s := &autoPruneScheduler{hostService: hosts, setup: SetupConfig{AutoPruneMode: &daily}, now: time.Now, after: time.After}

	s.tick(t.Context(), at(14, "04:00"))
	assert.Equal(t, []string{"a"}, hosts.pruned, "the flag turns it on and the file keeps the time")
}

func TestSetup_AutoPruneLocked(t *testing.T) {
	setupTestEnv(t, true)
	resetAutoPruneLast(t)
	writePruneSchedule(t, "weekly", "05:00")
	lockedTime := "02:30"
	h := setupNoneHandler(time.Now(), SetupConfig{AutoPruneTime: &lockedTime})

	state := getSetupState(t, h)
	assert.Equal(t, "weekly", state.AutoPrune.Mode)
	assert.Equal(t, "02:30", state.AutoPrune.Time)
	assert.False(t, state.Locked.AutoPruneMode)
	assert.True(t, state.Locked.AutoPruneTime)

	assert.Equal(t, http.StatusConflict, doSetup(h, "PATCH", "/api/setup/config", `{"autoPrune":{"mode":"daily","time":"04:00"}}`).Code)
}

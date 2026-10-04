package web

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recordingClientService plays back a fixed update for each container and
// records the order the updates ran in.
type recordingClientService struct {
	container.ClientService
	mu       sync.Mutex
	order    []string
	fail     map[string]bool
	rollBack map[string]bool
	opts     []container.UpdateOptions
}

func (s *recordingClientService) UpdateContainer(ctx context.Context, c container.Container, opts container.UpdateOptions, progressCh chan<- container.UpdateProgress) (bool, error) {
	defer close(progressCh)
	s.mu.Lock()
	s.order = append(s.order, c.ID)
	s.opts = append(s.opts, opts)
	s.mu.Unlock()

	progressCh <- container.UpdateProgress{Status: "pulling", Layer: "a", Current: 5, Total: 10}
	progressCh <- container.UpdateProgress{Status: "pulling", Layer: "b", Current: 10, Total: 10}
	if s.fail[c.ID] {
		progressCh <- container.UpdateProgress{Status: "error", Error: "pull failed"}
		return false, errors.New("pull failed")
	}
	if s.rollBack[c.ID] {
		progressCh <- container.UpdateProgress{Status: "recreating"}
		progressCh <- container.UpdateProgress{Status: "verifying"}
		progressCh <- container.UpdateProgress{Status: "rolled-back", Error: "replacement is unhealthy"}
		return false, errors.New("update rolled back: replacement is unhealthy")
	}
	progressCh <- container.UpdateProgress{Status: "recreating"}
	progressCh <- container.UpdateProgress{Status: "done"}
	return true, nil
}

func newTestUpdater() *bulkUpdater {
	return &bulkUpdater{watchers: make(map[chan struct{}]struct{})}
}

func waitDone(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("bulk update did not finish")
	}
}

func TestBulkUpdate_SelfRunsLast(t *testing.T) {
	selfID := "aaaaaaaaaaaa0000000000000000000000000000000000000000000000000000"
	prev := setupSelfID
	setupSelfID = func() string { return selfID }
	t.Cleanup(func() { setupSelfID = prev })

	client := &recordingClientService{}
	services := []*container.ContainerService{
		container.NewContainerService(client, container.Container{ID: "aaaaaaaaaaaa", Host: "local", Name: "dozzle"}),
		container.NewContainerService(client, container.Container{ID: "bbbbbbbbbbbb", Host: "local", Name: "web"}),
		container.NewContainerService(client, container.Container{ID: "cccccccccccc", Host: "local", Name: "db"}),
	}

	u := newTestUpdater()
	done, err := u.Start(services, "manual", "", "", nil)
	require.NoError(t, err)
	waitDone(t, done)

	assert.Equal(t, []string{"bbbbbbbbbbbb", "cccccccccccc", "aaaaaaaaaaaa"}, client.order)

	job, running := u.snapshot(nil)
	assert.False(t, running)
	require.NotNil(t, job.FinishedAt)
	for _, item := range job.Items {
		assert.Equal(t, "done", item.Status, item.Name)
		assert.Equal(t, int64(15), item.Current)
		assert.Equal(t, int64(20), item.Total)
	}
	assert.True(t, job.Items[0].Self)
}

func TestBulkUpdate_FailureDoesNotStopTheRest(t *testing.T) {
	client := &recordingClientService{fail: map[string]bool{"bbbbbbbbbbbb": true}}
	services := []*container.ContainerService{
		container.NewContainerService(client, container.Container{ID: "bbbbbbbbbbbb", Host: "local", Name: "web"}),
		container.NewContainerService(client, container.Container{ID: "cccccccccccc", Host: "local", Name: "db"}),
	}

	u := newTestUpdater()
	done, err := u.Start(services, "manual", "", "", nil)
	require.NoError(t, err)
	waitDone(t, done)

	job, _ := u.snapshot(nil)
	assert.Equal(t, "error", job.Items[0].Status)
	assert.Equal(t, "pull failed", job.Items[0].Error)
	assert.Equal(t, "done", job.Items[1].Status)
}

// A rolled back container ends the job as rolled back, not failed, with the
// reason the new one was refused.
func TestBulkUpdate_RolledBackStaysRolledBack(t *testing.T) {
	client := &recordingClientService{rollBack: map[string]bool{"bbbbbbbbbbbb": true}}
	services := []*container.ContainerService{
		container.NewContainerService(client, container.Container{ID: "bbbbbbbbbbbb", Host: "local", Name: "web"}),
		container.NewContainerService(client, container.Container{ID: "cccccccccccc", Host: "local", Name: "db"}),
	}

	u := newTestUpdater()
	done, err := u.Start(services, "schedule", "", "", nil)
	require.NoError(t, err)
	waitDone(t, done)

	job, _ := u.snapshot(nil)
	assert.Equal(t, "rolled-back", job.Items[0].Status)
	assert.Equal(t, "replacement is unhealthy", job.Items[0].Error)
	assert.Equal(t, "done", job.Items[1].Status)
	require.NotEmpty(t, job.RunID)
	want := container.UpdateOptions{Source: container.UpdateSourceSchedule, RunID: job.RunID}
	assert.Equal(t, []container.UpdateOptions{want, want}, client.opts)
}

// Every container in a run carries the same run id, and each run gets its own,
// so one scheduled night reads as one group.
func TestBulkUpdate_StampsSourceAndRun(t *testing.T) {
	client := &recordingClientService{}
	start := func(trigger string) bulkUpdateJob {
		u := newTestUpdater()
		done, err := u.Start([]*container.ContainerService{
			container.NewContainerService(client, container.Container{ID: "bbbbbbbbbbbb", Host: "local", Name: "web"}),
		}, trigger, "", "", nil)
		require.NoError(t, err)
		waitDone(t, done)
		job, _ := u.snapshot(nil)
		return job
	}

	manual := start("manual")
	scheduled := start("schedule")
	assert.NotEqual(t, manual.RunID, scheduled.RunID)
	assert.Equal(t, []container.UpdateOptions{
		{Source: container.UpdateSourceDozzle, RunID: manual.RunID},
		{Source: container.UpdateSourceSchedule, RunID: scheduled.RunID},
	}, client.opts)
}

func TestBulkUpdate_SwarmServiceUpdatedOnce(t *testing.T) {
	client := &recordingClientService{}
	labels := map[string]string{"com.docker.swarm.service.id": "svc1"}
	services := []*container.ContainerService{
		container.NewContainerService(client, container.Container{ID: "bbbbbbbbbbbb", Host: "node1", Labels: labels}),
		container.NewContainerService(client, container.Container{ID: "cccccccccccc", Host: "node2", Labels: labels}),
	}

	u := newTestUpdater()
	done, err := u.Start(services, "manual", "", "", nil)
	require.NoError(t, err)
	waitDone(t, done)

	assert.Len(t, client.order, 1)
}

func TestBulkUpdate_RejectsOverlap(t *testing.T) {
	block := make(chan struct{})
	client := &blockingClientService{release: block}
	u := newTestUpdater()
	done, err := u.Start([]*container.ContainerService{
		container.NewContainerService(client, container.Container{ID: "bbbbbbbbbbbb", Host: "local"}),
	}, "manual", "", "", nil)
	require.NoError(t, err)

	_, err = u.Start([]*container.ContainerService{
		container.NewContainerService(client, container.Container{ID: "cccccccccccc", Host: "local"}),
	}, "schedule", "", "", nil)
	assert.ErrorIs(t, err, errBulkUpdateBusy)

	close(block)
	waitDone(t, done)
}

func TestBulkUpdate_SnapshotFiltersHidden(t *testing.T) {
	client := &recordingClientService{}
	u := newTestUpdater()
	done, err := u.Start([]*container.ContainerService{
		container.NewContainerService(client, container.Container{ID: "bbbbbbbbbbbb", Host: "local", Name: "web"}),
		container.NewContainerService(client, container.Container{ID: "cccccccccccc", Host: "local", Name: "db"}),
	}, "manual", "", "", nil)
	require.NoError(t, err)
	waitDone(t, done)

	job, _ := u.snapshot(func(_ *bulkUpdateJob, item *bulkUpdateItem) bool { return item.Name == "db" })
	require.Len(t, job.Items, 1)
	assert.Equal(t, "cccccccccccc", job.Items[0].ID)
}

// Dozzle as a swarm service: the drawer lists a replica per node, and the one
// kept after dedupe is rarely this node's. It must still go last.
func TestBulkUpdate_SwarmSelfServiceRunsLast(t *testing.T) {
	selfID := "aaaaaaaaaaaa0000000000000000000000000000000000000000000000000000"
	prev := setupSelfID
	setupSelfID = func() string { return selfID }
	t.Cleanup(func() { setupSelfID = prev })

	client := &recordingClientService{}
	dozzle := map[string]string{swarmServiceLabel: "dozzle-svc"}
	services := []*container.ContainerService{
		container.NewContainerService(client, container.Container{ID: "dddddddddddd", Host: "node2", Name: "dozzle.2", Labels: dozzle}),
		container.NewContainerService(client, container.Container{ID: "aaaaaaaaaaaa", Host: "node1", Name: "dozzle.1", Labels: dozzle}),
		container.NewContainerService(client, container.Container{ID: "bbbbbbbbbbbb", Host: "node2", Name: "web"}),
	}

	u := newTestUpdater()
	done, err := u.Start(services, "manual", "dozzle-svc", "", nil)
	require.NoError(t, err)
	waitDone(t, done)

	assert.Equal(t, []string{"bbbbbbbbbbbb", "dddddddddddd"}, client.order)
	job, _ := u.snapshot(nil)
	require.Len(t, job.Items, 2)
	assert.True(t, job.Items[0].Self)
}

func TestIsSelfContainer(t *testing.T) {
	prev := setupSelfID
	setupSelfID = func() string { return "aaaaaaaaaaaa0000" }
	t.Cleanup(func() { setupSelfID = prev })

	containers := []container.Container{
		{ID: "aaaaaaaaaaaa", Labels: map[string]string{swarmServiceLabel: "svc"}},
		{ID: "bbbbbbbbbbbb", Labels: map[string]string{swarmServiceLabel: "other"}},
	}
	service := selfSwarmService(containers)
	assert.Equal(t, "svc", service)
	assert.True(t, isSelfContainer(container.Container{ID: "cccccccccccc", Labels: map[string]string{swarmServiceLabel: "svc"}}, service))
	assert.False(t, isSelfContainer(containers[1], service))
	assert.False(t, isSelfContainer(container.Container{ID: "cccccccccccc"}, ""))
}

func TestBulkUpdate_IdleWaitsForRunningJob(t *testing.T) {
	block := make(chan struct{})
	u := newTestUpdater()
	select {
	case <-u.idle():
	default:
		t.Fatal("idle should be closed with no job")
	}

	done, err := u.Start([]*container.ContainerService{
		container.NewContainerService(&blockingClientService{release: block}, container.Container{ID: "bbbbbbbbbbbb", Host: "local"}),
	}, "manual", "", "", nil)
	require.NoError(t, err)

	idle := u.idle()
	select {
	case <-idle:
		t.Fatal("idle closed while a job runs")
	default:
	}
	close(block)
	waitDone(t, done)
	<-idle
}

type blockingClientService struct {
	container.ClientService
	release chan struct{}
}

func (s *blockingClientService) UpdateContainer(ctx context.Context, _ container.Container, _ container.UpdateOptions, progressCh chan<- container.UpdateProgress) (bool, error) {
	defer close(progressCh)
	<-s.release
	return false, nil
}

func TestAutoUpdateEnabled(t *testing.T) {
	for value, want := range map[string]bool{
		"true": true, "TRUE": true, " yes ": true, "1": true, "on": true,
		"false": false, "": false, "no": false, "maybe": false,
	} {
		assert.Equal(t, want, autoUpdateEnabled(map[string]string{AutoUpdateLabel: value}), value)
	}
	assert.False(t, autoUpdateEnabled(nil))
}

func TestStartBulkUpdate_RefusesNonJSONBodies(t *testing.T) {
	h := &handler{config: &Config{Authorization: Authorization{Provider: NONE}}}
	for _, ct := range []string{"text/plain", "application/x-www-form-urlencoded", ""} {
		req := httptest.NewRequest(http.MethodPost, "/api/updates", strings.NewReader(`{"containers":[{"host":"h","id":"c"}]}`))
		if ct != "" {
			req.Header.Set("Content-Type", ct)
		}
		rr := httptest.NewRecorder()
		h.startBulkUpdate(rr, req)
		assert.Equal(t, http.StatusUnsupportedMediaType, rr.Code, ct)
	}
}

func TestUpdatable(t *testing.T) {
	swarm := map[string]string{swarmServiceLabel: "svc"}
	assert.True(t, container.Updatable(container.Container{State: "running"}))
	assert.True(t, container.Updatable(container.Container{State: "exited"}), "a stopped standalone container can still start on a new image")
	assert.False(t, container.Updatable(container.Container{State: "deleted"}))
	assert.True(t, container.Updatable(container.Container{State: "running", Labels: swarm}))
	assert.False(t, container.Updatable(container.Container{State: "exited", Labels: swarm}), "an exited swarm task was already replaced")
}

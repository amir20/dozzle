package container

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// updatingClient reports progress the way a docker host or an agent does,
// closing the channel it is given.
type updatingClient struct {
	ClientService
	progress []UpdateProgress
}

func (c *updatingClient) UpdateContainer(_ context.Context, _ Container, ch chan<- UpdateProgress) (bool, error) {
	defer close(ch)
	for _, p := range c.progress {
		ch <- p
	}
	return true, nil
}

func (c *updatingClient) RollbackContainer(_ context.Context, _ Container, _ RollbackOptions, ch chan<- UpdateProgress) error {
	defer close(ch)
	for _, p := range c.progress {
		ch <- p
	}
	return nil
}

func withUpdates(t *testing.T) *UpdateRecords {
	t.Helper()
	prev := Updates
	Updates = NewUpdateRecords()
	t.Cleanup(func() { Updates = prev })
	return Updates
}

func TestContainerServiceRecordsUpdates(t *testing.T) {
	records := withUpdates(t)
	sub := make(chan UpdateRecord, 4)
	records.Subscribe(t.Context(), sub)

	result := &UpdateResult{OldID: "old000000000", NewID: "new000000000", FromImageID: "sha256:a", ToImageID: "sha256:b", FromDigest: "nginx@sha256:a", ToDigest: "nginx@sha256:b"}
	client := &updatingClient{progress: []UpdateProgress{{Status: UpdatePulling}, {Status: UpdateDone, Result: result}}}
	cs := NewContainerService(client, Container{ID: "old000000000", Name: "web", Host: "nas", Image: "nginx:latest"})

	ch := make(chan UpdateProgress, 10)
	updated, err := cs.Update(context.Background(), UpdateSourceSchedule, ch)
	require.NoError(t, err)
	assert.True(t, updated)
	var statuses []string
	for p := range ch {
		statuses = append(statuses, p.Status)
	}
	assert.Equal(t, []string{UpdatePulling, UpdateDone}, statuses, "every progress reaches the caller, and the channel is closed")

	got, ok := records.Latest("nas", "new000000000")
	require.True(t, ok)
	assert.Equal(t, UpdateSourceSchedule, got.Source)
	assert.Equal(t, "web", got.Name)
	assert.Equal(t, "nginx:latest", got.ImageRef)
	assert.Equal(t, "nginx@sha256:b", got.ToDigest)
	select {
	case e := <-sub:
		assert.Equal(t, got, e)
	case <-time.After(time.Second):
		t.Fatal("subscriber not told")
	}

	// A rollback is recorded as one.
	client.progress = []UpdateProgress{{Status: UpdateDone, Result: &UpdateResult{OldID: "new000000000", NewID: "back00000000"}}}
	ch = make(chan UpdateProgress, 10)
	require.NoError(t, cs.Rollback(context.Background(), RollbackOptions{}, ch))
	got, ok = records.Latest("nas", "back00000000")
	require.True(t, ok)
	assert.Equal(t, UpdateSourceRollback, got.Source)
}

// An update that changed nothing (up to date, an error before the swap, a
// swarm service, Dozzle's own helper) carries no result and is not recorded.
func TestContainerServiceSkipsUpdatesWithoutResult(t *testing.T) {
	records := withUpdates(t)
	client := &updatingClient{progress: []UpdateProgress{{Status: UpdateUpToDate}}}
	cs := NewContainerService(client, Container{ID: "c00000000000", Name: "web", Host: "nas"})
	ch := make(chan UpdateProgress, 10)
	_, err := cs.Update(context.Background(), UpdateSourceDozzle, ch)
	require.NoError(t, err)
	assert.Empty(t, records.Recent())
}

// A caller that stops reading must not block the update it started.
func TestContainerServiceUpdateOutlivesItsReader(t *testing.T) {
	withUpdates(t)
	client := &updatingClient{progress: []UpdateProgress{{Status: UpdatePulling}, {Status: UpdatePulling}, {Status: UpdateDone, Result: &UpdateResult{NewID: "n"}}}}
	cs := NewContainerService(client, Container{ID: "c", Host: "nas"})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	done := make(chan struct{})
	go func() {
		_, _ = cs.Update(ctx, UpdateSourceDozzle, make(chan UpdateProgress))
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("update blocked on a reader that went away")
	}
}

func TestUpdateRecordsKeepsTheLastPerHost(t *testing.T) {
	records := NewUpdateRecords()
	base := time.Date(2026, 10, 1, 3, 0, 0, 0, time.UTC)
	for i := range updateRecordsPerHost + 5 {
		records.Add(UpdateRecord{Host: "a", NewID: "x", At: base.Add(time.Duration(i) * time.Minute)})
	}
	records.Add(UpdateRecord{Host: "b", NewID: "y", At: base})
	all := records.Recent()
	assert.Len(t, all, updateRecordsPerHost+1)
	assert.Equal(t, "b", all[0].Host, "oldest first across hosts")
	latest, ok := records.Latest("a", "x")
	require.True(t, ok)
	assert.Equal(t, base.Add(time.Duration(updateRecordsPerHost+4)*time.Minute), latest.At)
}

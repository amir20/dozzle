package container

import (
	"fmt"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	imageX = "sha256:xxxx"
	imageY = "sha256:yyyy"
	imageW = "sha256:wwww"
)

var (
	t0 = time.Date(2026, 10, 3, 3, 0, 0, 0, time.UTC)
	t1 = t0.Add(time.Hour)
)

func runningOn(id, name, imageID string, startedAt time.Time, labels map[string]string) Container {
	return Container{
		ID:          id,
		Name:        name,
		EngineName:  name,
		Host:        "host1",
		Image:       "nginx:latest",
		ImageID:     imageID,
		ImageDigest: "nginx@" + imageID,
		State:       "running",
		StartedAt:   startedAt,
		Labels:      labels,
		FullyLoaded: true,
		Stats:       utils.NewRingBuffer[ContainerStat](300),
	}
}

// Watchtower stops and removes the old container before it creates and starts
// the new one, so by the start the old one is gone.
func TestUpdateTracker_watchtower(t *testing.T) {
	tr := newUpdateTracker()
	labels := map[string]string{"com.centurylinklabs.watchtower.enable": "true"}
	tr.seen(runningOn("aaa", "web", imageX, t0, labels))
	tr.gone("aaa")

	event, ok := tr.started(runningOn("bbb", "web", imageY, t1, labels))
	require.True(t, ok)
	assert.Equal(t, ContainerUpdateEvent{
		Host:         "host1",
		Name:         "web",
		EngineName:   "web",
		OldID:        "aaa",
		NewID:        "bbb",
		FromRef:      "nginx:latest",
		ToRef:        "nginx:latest",
		FromDigest:   "nginx@" + imageX,
		ToDigest:     "nginx@" + imageY,
		FromImageID:  imageX,
		ToImageID:    imageY,
		OldStartedAt: t0,
		At:           t1,
		Source:       UpdateSourceWatchtower,
	}, event)
	assert.Equal(t, []ContainerUpdateEvent{event}, tr.recent())
}

func TestUpdateTracker_restartIsNotAnUpdate(t *testing.T) {
	tr := newUpdateTracker()
	tr.seen(runningOn("aaa", "web", imageX, t0, nil))
	_, ok := tr.started(runningOn("aaa", "web", imageX, t1, nil))
	assert.False(t, ok)

	// a recreate on the same image is not an update either
	_, ok = tr.started(runningOn("bbb", "web", imageX, t1, nil))
	assert.False(t, ok)
	assert.Empty(t, tr.recent())
}

func TestUpdateTracker_firstStartIsNotAnUpdate(t *testing.T) {
	tr := newUpdateTracker()
	_, ok := tr.started(runningOn("aaa", "web", imageX, t0, nil))
	assert.False(t, ok)
	_, ok = tr.started(runningOn("bbb", "web", imageY, t1, nil))
	assert.True(t, ok, "the first start is what the second is compared against")
}

func TestUpdateTracker_createdContainerDoesNotSeedName(t *testing.T) {
	tr := newUpdateTracker()
	created := runningOn("bbb", "web", imageY, time.Time{}, nil)
	created.State = "created"
	tr.seen(created)
	assert.Empty(t, tr.byName, "a container that never ran says nothing about its name")
}

// A list entry has no digest or start time. It must not wipe what an inspect
// of the same container already found.
func TestUpdateTracker_listEntryKeepsInspectData(t *testing.T) {
	tr := newUpdateTracker()
	tr.seen(runningOn("aaa", "web", imageX, t0, nil))
	tr.seen(Container{ID: "aaa", Name: "web", EngineName: "web", ImageID: imageX, State: "running"})

	event, ok := tr.started(runningOn("bbb", "web", imageY, t1, nil))
	require.True(t, ok)
	assert.Equal(t, "nginx@"+imageX, event.FromDigest)
	assert.Equal(t, t0, event.OldStartedAt)
}

// A list taken before an update lands after it: the old container must not
// overwrite the record of the one that replaced it.
func TestUpdateTracker_staleListDoesNotOverwrite(t *testing.T) {
	tr := newUpdateTracker()
	tr.seen(runningOn("aaa", "web", imageX, t0, nil))
	_, ok := tr.started(runningOn("bbb", "web", imageY, t1, nil))
	require.True(t, ok)
	tr.seen(runningOn("aaa", "web", imageX, t0, nil))

	_, ok = tr.started(runningOn("bbb", "web", imageY, t1.Add(time.Minute), nil))
	assert.False(t, ok, "a restart of the new container, not a second update")
}

func TestUpdateTracker_dozzleLabels(t *testing.T) {
	tr := newUpdateTracker()
	tr.seen(runningOn("aaa", "web", imageX, t0, nil))
	event, ok := tr.started(runningOn("bbb", "web", imageY, t1, map[string]string{
		PreviousImageLabel: imageX,
		UpdateSourceLabel:  UpdateSourceSchedule,
		UpdateRunLabel:     "run-1",
	}))
	require.True(t, ok)
	assert.Equal(t, UpdateSourceSchedule, event.Source)
	assert.Equal(t, "run-1", event.RunID)
}

func TestUpdateTracker_dozzleLabelWithoutSourceIsDozzle(t *testing.T) {
	tr := newUpdateTracker()
	tr.seen(runningOn("aaa", "web", imageX, t0, nil))
	event, ok := tr.started(runningOn("bbb", "web", imageY, t1, map[string]string{PreviousImageLabel: imageX}))
	require.True(t, ok)
	assert.Equal(t, UpdateSourceDozzle, event.Source)
	assert.Empty(t, event.RunID)
}

// Watchtower copies every label, so a container Dozzle once updated still says
// so after Watchtower recreates it. The previous image no longer matches.
func TestUpdateTracker_copiedDozzleLabelsAreNotDozzle(t *testing.T) {
	tr := newUpdateTracker()
	labels := map[string]string{
		PreviousImageLabel:                      imageW,
		UpdateSourceLabel:                       UpdateSourceSchedule,
		UpdateRunLabel:                          "run-0",
		"com.centurylinklabs.watchtower.enable": "true",
	}
	tr.seen(runningOn("aaa", "web", imageX, t0, labels))
	event, ok := tr.started(runningOn("bbb", "web", imageY, t1, labels))
	require.True(t, ok)
	assert.Equal(t, UpdateSourceWatchtower, event.Source)
	assert.Empty(t, event.RunID)
}

func TestUpdateTracker_watchtowerDisabledIsExternal(t *testing.T) {
	tr := newUpdateTracker()
	labels := map[string]string{"com.centurylinklabs.watchtower.enable": "false"}
	tr.seen(runningOn("aaa", "web", imageX, t0, labels))
	event, ok := tr.started(runningOn("bbb", "web", imageY, t1, labels))
	require.True(t, ok)
	assert.Equal(t, UpdateSourceExternal, event.Source)
}

// A swap whose replacement started and then failed puts the old container
// back. Its start is the swap undoing itself, which the swap reports.
func TestUpdateTracker_restoredOldContainerIsNotAnUpdate(t *testing.T) {
	tr := newUpdateTracker()
	tr.seen(runningOn("aaa", "web", imageX, t0, nil))
	_, ok := tr.started(runningOn("bbb", "web", imageY, t1, map[string]string{PreviousImageLabel: imageX}))
	require.True(t, ok)
	tr.gone("bbb")

	_, ok = tr.started(runningOn("aaa", "web", imageX, t1.Add(time.Minute), nil))
	assert.False(t, ok)

	tr.rolledBack(ContainerUpdateEvent{Name: "web", EngineName: "web", OldID: "aaa", NewID: "aaa", FromImageID: imageX, ToImageID: imageY, Source: UpdateSourceDozzle})
	events := tr.recent()
	require.Len(t, events, 2)
	assert.False(t, events[0].RolledBack)
	assert.True(t, events[1].RolledBack)
	assert.False(t, events[1].At.IsZero())

	// the next real update compares against the restored container
	event, ok := tr.started(runningOn("ccc", "web", imageW, t1.Add(time.Hour), nil))
	require.True(t, ok)
	assert.Equal(t, "aaa", event.OldID)
	assert.Equal(t, imageX, event.FromImageID)
}

// A --rm container is gone after its stop, so the rollback recreates it on its
// old image, under a new id.
func TestUpdateTracker_recreatedRmContainerIsNotAnUpdate(t *testing.T) {
	tr := newUpdateTracker()
	tr.seen(runningOn("aaa", "web", imageX, t0, nil))
	_, ok := tr.started(runningOn("bbb", "web", imageY, t1, nil))
	require.True(t, ok)

	_, ok = tr.started(runningOn("ccc", "web", imageX, t1.Add(time.Minute), map[string]string{restoredRefLabel: "nginx:latest"}))
	assert.False(t, ok)
}

// A rollback also runs the previous image by id, as a recreated --rm container
// does, but it is an update of its own, from the rollback source.
func TestUpdateTracker_rollbackIsAnUpdate(t *testing.T) {
	tr := newUpdateTracker()
	tr.seen(runningOn("aaa", "web", imageX, t0, nil))
	_, ok := tr.started(runningOn("bbb", "web", imageY, t1, map[string]string{PreviousImageLabel: imageX, UpdateSourceLabel: UpdateSourceSchedule, UpdateRunLabel: "run-1"}))
	require.True(t, ok)

	event, ok := tr.started(runningOn("ccc", "web", imageX, t1.Add(time.Hour), map[string]string{
		restoredRefLabel:   "nginx:latest",
		PreviousImageLabel: imageY,
		UpdateSourceLabel:  UpdateSourceRollback,
	}))
	require.True(t, ok)
	assert.Equal(t, UpdateSourceRollback, event.Source)
	assert.Empty(t, event.RunID)
	assert.Equal(t, imageY, event.FromImageID)
	assert.Equal(t, imageX, event.ToImageID)
}

// Going back to the old image by hand is an update like any other.
func TestUpdateTracker_manualDowngradeIsAnUpdate(t *testing.T) {
	tr := newUpdateTracker()
	tr.seen(runningOn("aaa", "web", imageX, t0, nil))
	_, ok := tr.started(runningOn("bbb", "web", imageY, t1, nil))
	require.True(t, ok)

	event, ok := tr.started(runningOn("ccc", "web", imageX, t1.Add(time.Minute), nil))
	require.True(t, ok)
	assert.Equal(t, imageY, event.FromImageID)
	assert.Equal(t, imageX, event.ToImageID)
}

func TestUpdateTracker_keepsLastEvents(t *testing.T) {
	tr := newUpdateTracker()
	tr.seen(runningOn("c0", "web", "sha256:0", t0, nil))
	for i := 1; i <= maxUpdateEvents+5; i++ {
		_, ok := tr.started(runningOn(fmt.Sprintf("c%d", i), "web", fmt.Sprintf("sha256:%d", i), t0.Add(time.Duration(i)*time.Minute), nil))
		require.True(t, ok)
	}
	events := tr.recent()
	require.Len(t, events, maxUpdateEvents)
	assert.Equal(t, "sha256:6", events[0].ToImageID, "the oldest are dropped first")
	assert.Equal(t, fmt.Sprintf("sha256:%d", maxUpdateEvents+5), events[len(events)-1].ToImageID)
}

func TestUpdateTracker_prunesGoneRecords(t *testing.T) {
	tr := newUpdateTracker()
	now := t0
	tr.now = func() time.Time { return now }
	for i := 0; i <= pruneRecordsAbove; i++ {
		id := fmt.Sprintf("c%d", i)
		tr.seen(runningOn(id, id, imageX, t0, nil))
		tr.gone(id)
	}
	now = now.Add(goneRecordTTL + time.Minute)
	tr.started(runningOn("live", "live", imageX, now, nil))
	assert.Len(t, tr.byName, 1)
}

// dev.dozzle.name and other labels can give two containers one display name.
// They are still two names to the engine, so restarting either is never an
// update from the other's image.
func TestUpdateTracker_sharedDisplayName(t *testing.T) {
	tr := newUpdateTracker()
	labels := map[string]string{"dev.dozzle.name": "db"}
	pg15 := runningOn("aaa", "db", imageX, t0, labels)
	pg15.EngineName, pg15.Image = "db-15", "postgres:15"
	pg16 := runningOn("bbb", "db", imageY, t0, labels)
	pg16.EngineName, pg16.Image = "db-16", "postgres:16"
	tr.seen(pg15)
	tr.seen(pg16)

	for range 2 {
		pg16.StartedAt = pg16.StartedAt.Add(time.Minute)
		_, ok := tr.started(pg16)
		assert.False(t, ok, "restarting postgres:16 is not an update from postgres:15")
		pg15.StartedAt = pg15.StartedAt.Add(time.Minute)
		_, ok = tr.started(pg15)
		assert.False(t, ok, "restarting postgres:15 is not an update from postgres:16")
	}
	assert.Empty(t, tr.recent())
	assert.Len(t, tr.byName, 2)
}

// Containers without a name have no engine name, and are never tracked under
// a shared "no name".
func TestUpdateTracker_noEngineName(t *testing.T) {
	tr := newUpdateTracker()
	first := runningOn("aaa", "no name", imageX, t0, nil)
	first.EngineName = ""
	second := runningOn("bbb", "no name", imageY, t1, nil)
	second.EngineName = ""
	tr.seen(first)
	_, ok := tr.started(second)
	assert.False(t, ok)
	assert.Empty(t, tr.byName)
}

func TestUpdateTracker_kubernetesDigest(t *testing.T) {
	tr := newUpdateTracker()
	tr.seen(Container{ID: "p1", Name: "web", EngineName: "web", ImageDigest: "nginx@sha256:a", State: "running"})
	event, ok := tr.started(Container{ID: "p2", Name: "web", EngineName: "web", ImageDigest: "nginx@sha256:b", State: "running"})
	require.True(t, ok, "without an image id the digest identifies the image")
	assert.Equal(t, "nginx@sha256:b", event.ToImageID)
}

func TestUpdateTracker_nilRecordsNothing(t *testing.T) {
	var tr *updateTracker
	tr.seen(runningOn("aaa", "web", imageX, t0, nil))
	_, ok := tr.started(runningOn("bbb", "web", imageY, t1, nil))
	assert.False(t, ok)
	tr.gone("aaa")
	tr.rolledBack(ContainerUpdateEvent{})
	tr.subscribe(t.Context(), make(chan ContainerUpdateEvent))
	assert.Nil(t, tr.recent())
}

// storeWithUpdates runs a store over the given list and hands back the event
// feed, the store's broadcast events and its update events.
func storeWithUpdates(t *testing.T, client *mockedClient, listed ...Container) (chan<- ContainerEvent, *Store, <-chan ContainerEvent, <-chan ContainerUpdateEvent) {
	t.Helper()
	client.On("ListContainers", mock.Anything, mock.Anything).Return(listed, nil)
	client.On("Host").Return(Host{ID: "host1"})
	feed := feedEvents(client)

	store := NewStore(t.Context(), client, newCaptureStatsCollector(), ContainerLabels{})
	events := make(chan ContainerEvent, 64)
	store.SubscribeEvents(t.Context(), events)
	updates := make(chan ContainerUpdateEvent, 8)
	store.SubscribeUpdates(t.Context(), updates)
	_, err := store.ListContainers(t.Context(), ContainerLabels{})
	require.NoError(t, err)
	return feed, store, events, updates
}

func waitForUpdate(t *testing.T, updates <-chan ContainerUpdateEvent) ContainerUpdateEvent {
	t.Helper()
	select {
	case e := <-updates:
		return e
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for an update event")
		return ContainerUpdateEvent{}
	}
}

func TestStore_watchtowerUpdate(t *testing.T) {
	labels := map[string]string{"com.centurylinklabs.watchtower.enable": "true"}
	old := runningOn("aaa", "web", imageX, t0, labels)
	replacement := runningOn("bbb", "web", imageY, t1, labels)
	client := new(mockedClient)
	client.On("FindContainer", mock.Anything, "bbb").Return(replacement, nil)
	feed, store, events, updates := storeWithUpdates(t, client, old)

	feed <- ContainerEvent{Name: "die", ActorID: "aaa"}
	feed <- ContainerEvent{Name: "destroy", ActorID: "aaa"}
	feed <- ContainerEvent{Name: "create", ActorID: "bbb"}
	feed <- ContainerEvent{Name: "start", ActorID: "bbb"}
	waitForEvent(t, events, "start")

	event := waitForUpdate(t, updates)
	assert.Equal(t, "aaa", event.OldID)
	assert.Equal(t, "bbb", event.NewID)
	assert.Equal(t, imageX, event.FromImageID)
	assert.Equal(t, imageY, event.ToImageID)
	assert.Equal(t, UpdateSourceWatchtower, event.Source)
	assert.Equal(t, []ContainerUpdateEvent{event}, store.RecentUpdates())
}

// The reviewer's repro: two containers labelled dev.dozzle.name: db, one on
// postgres:15 and one on postgres:16. docker restart of either is a restart.
func TestStore_sharedDisplayNameRestarts(t *testing.T) {
	labels := map[string]string{"dev.dozzle.name": "db"}
	pg15 := runningOn("aaa", "db", imageX, t0, labels)
	pg15.EngineName, pg15.Image = "db-15", "postgres:15"
	pg16 := runningOn("bbb", "db", imageY, t0, labels)
	pg16.EngineName, pg16.Image = "db-16", "postgres:16"
	restarted15, restarted16 := pg15, pg16
	restarted15.StartedAt, restarted16.StartedAt = t1, t1
	client := new(mockedClient)
	client.On("FindContainer", mock.Anything, "aaa").Return(restarted15, nil)
	client.On("FindContainer", mock.Anything, "bbb").Return(restarted16, nil)
	feed, store, events, updates := storeWithUpdates(t, client, pg15, pg16)

	for _, id := range []string{"bbb", "aaa", "bbb", "aaa"} {
		feed <- ContainerEvent{Name: "die", ActorID: id}
		feed <- ContainerEvent{Name: "start", ActorID: id}
		waitForEvent(t, events, "start")
	}

	assert.Empty(t, store.RecentUpdates())
	select {
	case e := <-updates:
		t.Fatalf("a restart was recorded as an update: %+v", e)
	default:
	}
}

// compose creates the new container under a temporary name, removes the old
// one and renames the new one before starting it.
func TestStore_composeRecreate(t *testing.T) {
	old := runningOn("aaa", "proj-web-1", imageX, t0, map[string]string{"com.docker.compose.service": "web"})
	created := runningOn("bbb", "bbbbbbbbbbbb_proj-web-1", imageY, time.Time{}, nil)
	created.State = "created"
	started := runningOn("bbb", "proj-web-1", imageY, t1, map[string]string{"com.docker.compose.service": "web"})
	client := new(mockedClient)
	client.On("FindContainer", mock.Anything, "bbb").Return(created, nil).Once()
	client.On("FindContainer", mock.Anything, "bbb").Return(started, nil)
	feed, store, events, updates := storeWithUpdates(t, client, old)

	feed <- ContainerEvent{Name: "create", ActorID: "bbb"}
	feed <- ContainerEvent{Name: "die", ActorID: "aaa"}
	feed <- ContainerEvent{Name: "destroy", ActorID: "aaa"}
	feed <- ContainerEvent{Name: "rename", ActorID: "bbb", ActorAttributes: map[string]string{"name": "proj-web-1"}}
	feed <- ContainerEvent{Name: "start", ActorID: "bbb"}
	waitForEvent(t, events, "start")

	event := waitForUpdate(t, updates)
	assert.Equal(t, "proj-web-1", event.Name)
	assert.Equal(t, "aaa", event.OldID)
	assert.Equal(t, "bbb", event.NewID)
	assert.Equal(t, UpdateSourceExternal, event.Source)
	assert.Len(t, store.RecentUpdates(), 1, "the temporary name is not an update")
}

// Dozzle's swap renames the old container out of the way, creates the new one
// under the name, stops the old one, starts the new one and removes the old.
func TestStore_dozzleSwap(t *testing.T) {
	old := runningOn("aaa", "app", imageX, t0, nil)
	labels := map[string]string{PreviousImageLabel: imageX, UpdateSourceLabel: UpdateSourceSchedule, UpdateRunLabel: "run-1"}
	created := runningOn("bbb", "app", imageY, time.Time{}, labels)
	created.State = "created"
	client := new(mockedClient)
	client.On("FindContainer", mock.Anything, "bbb").Return(created, nil).Once()
	client.On("FindContainer", mock.Anything, "bbb").Return(runningOn("bbb", "app", imageY, t1, labels), nil)
	feed, store, events, updates := storeWithUpdates(t, client, old)

	feed <- ContainerEvent{Name: "rename", ActorID: "aaa", ActorAttributes: map[string]string{"name": "app-dozzle-old-aaa"}}
	feed <- ContainerEvent{Name: "create", ActorID: "bbb"}
	feed <- ContainerEvent{Name: "die", ActorID: "aaa"}
	feed <- ContainerEvent{Name: "start", ActorID: "bbb"}
	feed <- ContainerEvent{Name: "destroy", ActorID: "aaa"}
	waitForEvent(t, events, "destroy")

	event := waitForUpdate(t, updates)
	assert.Equal(t, "app", event.Name)
	assert.Equal(t, "aaa", event.OldID)
	assert.Equal(t, "bbb", event.NewID)
	assert.Equal(t, UpdateSourceSchedule, event.Source)
	assert.Equal(t, "run-1", event.RunID)
	assert.Equal(t, t0, event.OldStartedAt)
	assert.False(t, event.RolledBack)
	assert.Len(t, store.RecentUpdates(), 1)
}

// The replacement started, then failed: the swap removes it, renames the old
// container back and starts it. That is one update and its rollback, never a
// second update back to the old image.
func TestStore_dozzleSwapRolledBack(t *testing.T) {
	old := runningOn("aaa", "app", imageX, t0, nil)
	labels := map[string]string{PreviousImageLabel: imageX}
	client := new(mockedClient)
	client.On("FindContainer", mock.Anything, "bbb").Return(runningOn("bbb", "app", imageY, t1, labels), nil)
	client.On("FindContainer", mock.Anything, "aaa").Return(runningOn("aaa", "app", imageX, t1.Add(time.Minute), nil), nil)
	feed, store, events, updates := storeWithUpdates(t, client, old)

	feed <- ContainerEvent{Name: "rename", ActorID: "aaa", ActorAttributes: map[string]string{"name": "app-dozzle-old-aaa"}}
	feed <- ContainerEvent{Name: "die", ActorID: "aaa"}
	feed <- ContainerEvent{Name: "start", ActorID: "bbb"}
	waitForEvent(t, events, "start")
	forward := waitForUpdate(t, updates)
	assert.False(t, forward.RolledBack)

	feed <- ContainerEvent{Name: "die", ActorID: "bbb"}
	feed <- ContainerEvent{Name: "destroy", ActorID: "bbb"}
	feed <- ContainerEvent{Name: "rename", ActorID: "aaa", ActorAttributes: map[string]string{"name": "app"}}
	feed <- ContainerEvent{Name: "start", ActorID: "aaa"}
	waitForEvent(t, events, "start")

	store.RecordRolledBack(ContainerUpdateEvent{Host: "host1", Name: "app", EngineName: "app", OldID: "aaa", NewID: "aaa", FromImageID: imageX, ToImageID: imageY, Source: UpdateSourceDozzle})
	rolledBack := waitForUpdate(t, updates)
	assert.True(t, rolledBack.RolledBack)

	recent := store.RecentUpdates()
	require.Len(t, recent, 2)
	assert.Equal(t, forward, recent[0])
	assert.Equal(t, rolledBack, recent[1])
}

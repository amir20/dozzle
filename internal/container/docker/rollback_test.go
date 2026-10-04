package docker

import (
	"context"
	"maps"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/container/swap"
	docker_types "github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/image"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// updatedApp is a container Dozzle updated from olderImage to oldImageID.
func updatedApp(extra map[string]string) docker_types.InspectResponse {
	labels := map[string]string{
		container.PreviousImageLabel: olderImage,
		container.PreviousRefLabel:   "nginx@sha256:older",
		container.UpdateSourceLabel:  container.UpdateSourceSchedule,
		container.UpdateRunLabel:     "run-1",
	}
	maps.Copy(labels, extra)
	return appInspect(labels)
}

func newRollbackClient(t *testing.T, old docker_types.InspectResponse) *updateClient {
	cli := newUpdateClient(t, old)
	cli.images[olderImage] = image.InspectResponse{ID: olderImage, RepoDigests: []string{"nginx@sha256:older"}}
	return cli
}

func runRollback(svc *Service, opts container.RollbackOptions) updateRun {
	ch := make(chan container.UpdateProgress, 100)
	err := svc.RollbackContainer(context.Background(), container.Container{ID: appID, Name: "app", Host: "host1"}, opts, ch)
	run := updateRun{err: err}
	for p := range ch {
		if len(run.statuses) == 0 || run.statuses[len(run.statuses)-1] != p.Status {
			run.statuses = append(run.statuses, p.Status)
		}
		run.last = p
	}
	return run
}

func TestRollbackContainerSwapsToPreviousImage(t *testing.T) {
	cli := newRollbackClient(t, updatedApp(nil))
	run := runRollback(&Service{client: cli}, container.RollbackOptions{})

	require.NoError(t, run.err)
	assert.Equal(t, []string{"recreating", "verifying", "done"}, run.statuses)
	assert.Empty(t, cli.pulled, "the previous image is still here, nothing is pulled")
	assert.Equal(t, []string{
		"rename " + appID + " app-dozzle-old-abc000000000",
		"create app",
		"stop " + appID,
		"start new1",
		"remove " + appID,
	}, cli.engine.Calls, "the current container goes only after the previous image stayed up")

	spec := cli.engine.Created[0].Config
	assert.Equal(t, olderImage, spec.Image, "the previous image runs by id")
	assert.Equal(t, "nginx:latest", swap.ImageRef(spec), "and the container still follows its tag")
	assert.Equal(t, oldImageID, spec.Labels[container.PreviousImageLabel], "what it rolled back from")
	assert.Equal(t, "nginx@sha256:bbb", spec.Labels[container.PreviousRefLabel])
	assert.Equal(t, container.UpdateSourceRollback, spec.Labels[container.UpdateSourceLabel])
	assert.NotContains(t, spec.Labels, container.UpdateRunLabel, "a rollback is not part of the run")
}

func TestRollbackContainerCleanupRemovesImageRolledBackFrom(t *testing.T) {
	cli := newRollbackClient(t, updatedApp(nil))
	run := runRollback(&Service{client: cli}, container.RollbackOptions{})
	require.NoError(t, run.err)
	assert.Equal(t, []string{oldImageID}, cli.removed, "an untagged image rolled back from goes")

	// A refusal (another container uses it) is only logged.
	cli = newRollbackClient(t, updatedApp(nil))
	cli.removeErr = assert.AnError
	run = runRollback(&Service{client: cli}, container.RollbackOptions{})
	require.NoError(t, run.err)
	assert.Equal(t, "done", run.last.Status)
}

func TestRollbackContainerCleanupKeepsATaggedImage(t *testing.T) {
	// The usual case: the tag still names the image rolled back from, and
	// removing it by id would untag it too.
	cli := newRollbackClient(t, updatedApp(nil))
	img := cli.images[oldImageID]
	img.RepoTags = []string{"nginx:latest"}
	cli.images[oldImageID] = img
	run := runRollback(&Service{client: cli}, container.RollbackOptions{})
	require.NoError(t, run.err)
	assert.Empty(t, cli.removed)
}

func TestRollbackContainerPullsPrunedImageByDigest(t *testing.T) {
	cli := newRollbackClient(t, updatedApp(nil))
	delete(cli.images, olderImage)
	cli.ids = map[string]string{"nginx@sha256:older": olderImage}
	run := runRollback(&Service{client: cli}, container.RollbackOptions{})

	require.NoError(t, run.err)
	assert.Equal(t, []string{"nginx@sha256:older"}, cli.pulled, "pulled by digest, never by tag")
	assert.Equal(t, []string{"pulling", "recreating", "verifying", "done"}, run.statuses)
	assert.Equal(t, olderImage, cli.engine.Created[0].Config.Image)
}

func TestRollbackContainerNeverPullsATag(t *testing.T) {
	for name, ref := range map[string]string{"no ref": "", "a tag": "nginx:1.4.1"} {
		t.Run(name, func(t *testing.T) {
			cli := newRollbackClient(t, updatedApp(map[string]string{container.PreviousRefLabel: ref}))
			delete(cli.images, olderImage)
			run := runRollback(&Service{client: cli}, container.RollbackOptions{})

			require.ErrorContains(t, run.err, "no registry digest to pull it by")
			assert.Equal(t, "error", run.last.Status)
			assert.Empty(t, cli.pulled)
			assert.Empty(t, cli.engine.Calls, "the container is untouched")
		})
	}
}

func TestRollbackContainerChecksExpectedDigest(t *testing.T) {
	cli := newRollbackClient(t, updatedApp(nil))
	run := runRollback(&Service{client: cli}, container.RollbackOptions{ExpectedFromDigest: "nginx@sha256:moved"})
	require.ErrorIs(t, run.err, container.ErrDigestMismatch)
	assert.Contains(t, run.last.Error, "nginx@sha256:bbb", "says what it runs instead")
	assert.Empty(t, cli.engine.Calls, "the container is untouched")

	for _, expected := range []string{"nginx@sha256:bbb", "sha256:bbb", "ghcr.io/other/nginx@sha256:aaa", oldImageID} {
		cli = newRollbackClient(t, updatedApp(nil))
		run = runRollback(&Service{client: cli}, container.RollbackOptions{ExpectedFromDigest: expected})
		require.NoError(t, run.err, expected)
	}
}

func TestRollbackContainerWithoutTarget(t *testing.T) {
	cli := newRollbackClient(t, appInspect(nil))
	run := runRollback(&Service{client: cli}, container.RollbackOptions{})
	require.ErrorIs(t, run.err, container.ErrNoRollbackTarget)
	assert.Equal(t, "error", run.last.Status)

	// A container a rollback created points back at the newer image: rolling
	// it "back" again would undo the rollback.
	cli = newRollbackClient(t, appInspect(map[string]string{
		container.PreviousImageLabel: olderImage,
		container.UpdateSourceLabel:  container.UpdateSourceRollback,
	}))
	run = runRollback(&Service{client: cli}, container.RollbackOptions{})
	require.ErrorIs(t, run.err, container.ErrNoRollbackTarget)
	assert.Empty(t, cli.engine.Calls)
}

func TestRollbackContainerOnlyToItsPreviousImage(t *testing.T) {
	cli := newRollbackClient(t, updatedApp(nil))
	run := runRollback(&Service{client: cli}, container.RollbackOptions{ToImageID: newImageID})
	require.ErrorIs(t, run.err, container.ErrNoRollbackTarget)
	assert.Empty(t, cli.engine.Calls)

	cli = newRollbackClient(t, updatedApp(nil))
	run = runRollback(&Service{client: cli}, container.RollbackOptions{ToImageID: olderImage})
	require.NoError(t, run.err)
}

func TestRollbackContainerUnsupportedForSwarm(t *testing.T) {
	cli := newRollbackClient(t, updatedApp(map[string]string{"com.docker.swarm.service.name": "web"}))
	run := runRollback(&Service{client: cli}, container.RollbackOptions{})
	require.ErrorIs(t, run.err, container.ErrRollbackUnsupported)
	assert.Contains(t, run.last.Error, "swarm")
	assert.Empty(t, cli.engine.Calls)
}

func TestRollbackContainerUndoneWhenPreviousImageFails(t *testing.T) {
	cli := newRollbackClient(t, updatedApp(nil))
	cli.engine.StartErrFor = "new1"
	store := container.NewStore(t.Context(), idleClient{}, idleStats{}, container.ContainerLabels{})
	svc := &Service{client: cli, store: store}
	run := runRollback(svc, container.RollbackOptions{})

	require.ErrorContains(t, run.err, "still runs its current image")
	assert.Equal(t, []string{"recreating", "rolled-back"}, run.statuses)
	assert.Equal(t, "start "+appID, cli.engine.Calls[len(cli.engine.Calls)-1], "the current container runs again")
	assert.Empty(t, cli.removed, "nothing is cleaned up")

	events := svc.RecentUpdates()
	require.Len(t, events, 1)
	assert.True(t, events[0].RolledBack)
	assert.Equal(t, container.UpdateSourceRollback, events[0].Source)
	assert.Equal(t, oldImageID, events[0].FromImageID)
	assert.Equal(t, olderImage, events[0].ToImageID)
}

// startingClient is an engine whose one event is a start, sent once the test
// opens the gate: the store has to have listed the container it replaces first.
type startingClient struct {
	idleClient
	listed  []container.Container
	started container.Container
	gate    chan struct{}
}

func (c startingClient) ListContainers(context.Context, container.ContainerLabels) ([]container.Container, error) {
	return c.listed, nil
}

func (c startingClient) FindContainer(_ context.Context, id string) (container.Container, error) {
	if id == c.started.ID {
		return c.started, nil
	}
	return container.Container{}, container.ErrContainerNotFound
}

func (c startingClient) ContainerEvents(ctx context.Context, ch chan<- container.ContainerEvent) error {
	select {
	case <-c.gate:
		ch <- container.ContainerEvent{Name: "start", ActorID: c.started.ID, Host: "host1"}
	case <-ctx.Done():
	}
	<-ctx.Done()
	return ctx.Err()
}

// Watchtower copies every label onto the container it recreates, so after it
// updates a container Dozzle once updated, the label names an older image than
// the one just replaced. The update events know which one that was.
func TestRollbackContainerPrefersUpdateEvents(t *testing.T) {
	const watchtowerFrom = "sha256:4444444444444444444444444444444444444444444444444444444444444444"
	old := updatedApp(map[string]string{"com.centurylinklabs.watchtower.enable": "true"})
	cli := newRollbackClient(t, old)
	cli.images[watchtowerFrom] = image.InspectResponse{ID: watchtowerFrom}

	engine := startingClient{
		listed: []container.Container{{ID: "fff000000000", Name: "app", EngineName: "app", Host: "host1", State: "running", ImageID: watchtowerFrom}},
		started: container.Container{
			ID: appID[:12], Name: "app", EngineName: "app", Host: "host1", State: "running", ImageID: oldImageID,
			Labels: old.Config.Labels, StartedAt: time.Now(), FullyLoaded: true,
		},
		gate: make(chan struct{}),
	}
	store := container.NewStore(t.Context(), engine, idleStats{}, container.ContainerLabels{})
	_, err := store.ListContainers(t.Context(), container.ContainerLabels{})
	require.NoError(t, err)
	close(engine.gate)
	require.Eventually(t, func() bool { return len(store.RecentUpdates()) == 1 }, 2*time.Second, 5*time.Millisecond)
	require.Equal(t, container.UpdateSourceWatchtower, store.RecentUpdates()[0].Source)

	run := runRollback(&Service{client: cli, store: store}, container.RollbackOptions{})
	require.NoError(t, run.err)
	assert.Equal(t, watchtowerFrom, cli.engine.Created[0].Config.Image, "the image Watchtower replaced, not the stale label's")

	// The stale label is still a target the caller may name.
	cli = newRollbackClient(t, old)
	run = runRollback(&Service{client: cli, store: store}, container.RollbackOptions{ToImageID: olderImage})
	require.NoError(t, run.err)
	assert.Equal(t, olderImage, cli.engine.Created[0].Config.Image)
}

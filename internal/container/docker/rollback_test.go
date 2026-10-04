package docker

import (
	"context"
	"maps"
	"testing"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/container/docker/swap"
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
	assert.Equal(t, "nginx@sha256:bbb", spec.Labels[container.RolledBackFromLabel], "what it rolled back from, for the schedule")
	assert.NotContains(t, spec.Labels, container.PreviousImageLabel, "a rolled back container has no target of its own")
	assert.NotContains(t, spec.Labels, container.PreviousRefLabel)

	require.NotNil(t, run.last.Result, "the record of the rollback")
	assert.Equal(t, container.UpdateResult{
		OldID:       appID[:12],
		NewID:       "new1",
		FromImageID: oldImageID,
		ToImageID:   olderImage,
		FromDigest:  "nginx@sha256:bbb",
		ToDigest:    "nginx@sha256:older",
	}, *run.last.Result)
}

func TestRollbackContainerCleanupRemovesImageRolledBackFrom(t *testing.T) {
	cli := newRollbackClient(t, updatedApp(nil))
	run := runRollback(&Service{client: cli}, container.RollbackOptions{})
	require.NoError(t, run.err)
	assert.Equal(t, []string{oldImageID}, cli.engine.RemovedImages, "an untagged image rolled back from goes")

	// A refusal (another container uses it) is only logged.
	cli = newRollbackClient(t, updatedApp(nil))
	cli.engine.ImageRemoveErr = assert.AnError
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
	assert.Empty(t, cli.engine.RemovedImages)
}

// Nothing is ever pulled: the tag names the image being rolled back from, and
// a previous image that is gone is reported as gone.
func TestRollbackContainerFailsWhenPreviousImageIsGone(t *testing.T) {
	cli := newRollbackClient(t, updatedApp(nil))
	delete(cli.images, olderImage)
	run := runRollback(&Service{client: cli}, container.RollbackOptions{})

	require.ErrorIs(t, run.err, container.ErrNoRollbackTarget)
	assert.Contains(t, run.last.Error, "no longer on this host")
	assert.Equal(t, "error", run.last.Status)
	assert.Empty(t, cli.engine.Calls, "the container is untouched")
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

	// A container a rollback created must not be rolled "back" again, which
	// would undo the rollback.
	cli = newRollbackClient(t, updatedApp(map[string]string{container.RolledBackFromLabel: "nginx@sha256:bbb"}))
	run = runRollback(&Service{client: cli}, container.RollbackOptions{})
	require.ErrorIs(t, run.err, container.ErrNoRollbackTarget)
	assert.Empty(t, cli.engine.Calls)
}

func TestRollbackContainerUnsupportedForSwarm(t *testing.T) {
	cli := newRollbackClient(t, updatedApp(map[string]string{"com.docker.swarm.service.name": "web"}))
	run := runRollback(&Service{client: cli}, container.RollbackOptions{})
	require.ErrorIs(t, run.err, container.ErrRollbackUnsupported)
	assert.Contains(t, run.last.Error, "swarm")
	assert.Empty(t, cli.engine.Calls)
}

// Like an update, a rollback never touches a container someone stopped.
func TestRollbackContainerRefusesStoppedContainer(t *testing.T) {
	old := updatedApp(nil)
	old.State = &docker_types.State{Status: "exited"}
	cli := newRollbackClient(t, old)
	run := runRollback(&Service{client: cli}, container.RollbackOptions{})
	require.ErrorIs(t, run.err, container.ErrNotRunning)
	assert.Equal(t, "error", run.last.Status)
	assert.Empty(t, cli.engine.Calls)
}

func TestRollbackContainerUndoneWhenPreviousImageFails(t *testing.T) {
	cli := newRollbackClient(t, updatedApp(nil))
	cli.engine.StartErrFor = "new1"
	run := runRollback(&Service{client: cli}, container.RollbackOptions{})

	require.ErrorContains(t, run.err, "still runs its current image")
	assert.Equal(t, []string{"recreating", "rolled-back"}, run.statuses)
	assert.Equal(t, "start "+appID, cli.engine.Calls[len(cli.engine.Calls)-1], "the current container runs again")
	assert.Empty(t, cli.engine.RemovedImages, "nothing is cleaned up")
	assert.Nil(t, run.last.Result, "nothing changed, so there is nothing to record")
}

// An update after a rollback clears its mark, so the schedule follows the tag
// again.
func TestUpdateContainerClearsRolledBackFrom(t *testing.T) {
	cli := newUpdateClient(t, appInspect(map[string]string{container.RolledBackFromLabel: "nginx@sha256:bbb"}))
	run := runUpdate(cli)
	require.NoError(t, run.err)
	assert.NotContains(t, cli.engine.Created[0].Config.Labels, container.RolledBackFromLabel)
}

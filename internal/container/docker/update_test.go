package docker

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/container/swap"
	"github.com/amir20/dozzle/internal/container/swap/swaptest"
	docker_types "github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/image"
	"github.com/moby/moby/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	appID      = "abc0000000000000000000000000000000000000000000000000000000000000"
	olderImage = "sha256:1111111111111111111111111111111111111111111111111111111111111111"
	oldImageID = "sha256:2222222222222222222222222222222222222222222222222222222222222222"
	newImageID = "sha256:3333333333333333333333333333333333333333333333333333333333333333"
)

// updateClient is an engine for UpdateContainer: the swap runs against the
// embedded fake, and the rest is canned. The nil UpdateClient panics on any
// call a test did not expect.
type updateClient struct {
	UpdateClient
	engine *swaptest.Fake

	pullBody   string
	images     map[string]image.InspectResponse
	removeErr  error
	removed    []string
	dependents []string
	// calls are what rejoinDependents asked for.
	calls []string
}

func (u *updateClient) ContainerInspect(ctx context.Context, id string) (docker_types.InspectResponse, error) {
	result, err := u.engine.ContainerInspect(ctx, id, client.ContainerInspectOptions{})
	return result.Container, err
}

func (u *updateClient) ImagePull(context.Context, string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader(u.pullBody)), nil
}

func (u *updateClient) ImageID(context.Context, string) (string, error) {
	return newImageID, nil
}

func (u *updateClient) ImageInspect(_ context.Context, ref string) (image.InspectResponse, error) {
	img, ok := u.images[ref]
	if !ok {
		return image.InspectResponse{}, errors.New("no such image")
	}
	return img, nil
}

func (u *updateClient) ImageRemove(_ context.Context, id string) error {
	u.removed = append(u.removed, id)
	return u.removeErr
}

func (u *updateClient) NetworkDependents(context.Context, string, string) ([]string, error) {
	return u.dependents, nil
}

func (u *updateClient) SwapAPI() swap.API { return u.engine }

func (u *updateClient) ContainerActions(_ context.Context, action container.ContainerAction, id string) error {
	u.calls = append(u.calls, string(action)+" "+id)
	return nil
}

func (u *updateClient) ContainerRemove(_ context.Context, id string) error {
	u.calls = append(u.calls, "remove "+id)
	return nil
}

func (u *updateClient) ContainerCreate(_ context.Context, inspect docker_types.InspectResponse, name string) (string, error) {
	u.calls = append(u.calls, "create "+name+" "+string(inspect.HostConfig.NetworkMode))
	return "new-" + name, nil
}

func appInspect(labels map[string]string) docker_types.InspectResponse {
	return docker_types.InspectResponse{
		ID:         appID,
		Name:       "/app",
		Image:      oldImageID,
		State:      &docker_types.State{Running: true, Status: "running"},
		Config:     &docker_types.Config{Image: "nginx:latest", Labels: labels},
		HostConfig: &docker_types.HostConfig{NetworkMode: "bridge"},
	}
}

func newUpdateClient(t *testing.T, old docker_types.InspectResponse) *updateClient {
	swaptest.FastTimings(t)
	prevID, prevHost := selfContainerID, hostname
	t.Cleanup(func() { selfContainerID, hostname = prevID, prevHost })
	selfContainerID = func() string { return "" }
	hostname = func() (string, error) { return "somehost", nil }

	return &updateClient{
		engine:   swaptest.New(old),
		pullBody: `{"status":"Downloading","id":"layer1","progressDetail":{"current":1,"total":2}}`,
		images: map[string]image.InspectResponse{
			oldImageID: {ID: oldImageID, RepoDigests: []string{"ghcr.io/other/nginx@sha256:aaa", "nginx@sha256:bbb"}},
			// Left dangling by the update before last.
			olderImage: {ID: olderImage},
		},
	}
}

type updateRun struct {
	statuses []string
	last     container.UpdateProgress
	updated  bool
	err      error
}

func runUpdate(cli *updateClient, opts container.UpdateOptions) updateRun {
	svc := &Service{client: cli}
	ch := make(chan container.UpdateProgress, 100)
	updated, err := svc.UpdateContainer(context.Background(), container.Container{ID: appID, Name: "app"}, opts, ch)
	run := updateRun{updated: updated, err: err}
	for p := range ch {
		if len(run.statuses) == 0 || run.statuses[len(run.statuses)-1] != p.Status {
			run.statuses = append(run.statuses, p.Status)
		}
		run.last = p
	}
	return run
}

func TestUpdateContainerSwapsAndStampsLabels(t *testing.T) {
	cli := newUpdateClient(t, appInspect(map[string]string{container.PreviousRefLabel: "nginx@sha256:stale"}))
	run := runUpdate(cli, container.UpdateOptions{})

	require.NoError(t, run.err)
	assert.True(t, run.updated)
	assert.Equal(t, []string{"pulling", "recreating", "verifying", "done"}, run.statuses)
	assert.Equal(t, []string{
		"rename " + appID + " app-dozzle-old-abc000000000",
		"create app",
		"stop " + appID,
		"start new1",
		"remove " + appID,
	}, cli.engine.Calls, "the old container goes only after the new one stayed up")

	labels := cli.engine.Created[0].Config.Labels
	assert.Equal(t, oldImageID, labels[container.PreviousImageLabel])
	assert.Equal(t, "nginx@sha256:bbb", labels[container.PreviousRefLabel], "the digest of the repository the tag names")
	assert.Empty(t, cli.removed, "cleanup is off")
}

func TestUpdateContainerLocalImageHasNoPreviousRef(t *testing.T) {
	cli := newUpdateClient(t, appInspect(map[string]string{container.PreviousRefLabel: "nginx@sha256:stale"}))
	cli.images[oldImageID] = image.InspectResponse{ID: oldImageID}
	run := runUpdate(cli, container.UpdateOptions{})

	require.NoError(t, run.err)
	labels := cli.engine.Created[0].Config.Labels
	assert.Equal(t, oldImageID, labels[container.PreviousImageLabel])
	assert.NotContains(t, labels, container.PreviousRefLabel, "a built image has no digest, and the old container's is stale")
}

func TestUpdateContainerRolledBackOnStartFailure(t *testing.T) {
	cli := newUpdateClient(t, appInspect(map[string]string{container.PreviousImageLabel: olderImage}))
	cli.engine.StartErrFor = "new1"
	run := runUpdate(cli, container.UpdateOptions{Cleanup: true})

	require.ErrorContains(t, run.err, "rolled back")
	assert.False(t, run.updated)
	assert.Equal(t, []string{"pulling", "recreating", "rolled-back"}, run.statuses)
	assert.Contains(t, run.last.Error, "start replacement")
	assert.Equal(t, []string{
		"rename " + appID + " app-dozzle-old-abc000000000",
		"create app",
		"stop " + appID,
		"start new1",
		"remove new1",
		"rename " + appID + " app",
		"start " + appID,
	}, cli.engine.Calls)
	assert.Empty(t, cli.removed, "no image is cleaned up after a rollback")
}

func TestUpdateContainerRolledBackOnUnhealthy(t *testing.T) {
	cli := newUpdateClient(t, appInspect(map[string]string{container.PreviousImageLabel: olderImage}))
	cli.engine.NewState = &docker_types.State{Running: true, StartedAt: "t0", Health: &docker_types.Health{Status: docker_types.Unhealthy}}
	run := runUpdate(cli, container.UpdateOptions{Cleanup: true})

	require.ErrorContains(t, run.err, "unhealthy")
	assert.Equal(t, []string{"pulling", "recreating", "verifying", "rolled-back"}, run.statuses)
	assert.Contains(t, cli.engine.Calls, "remove new1")
	assert.Equal(t, "start "+appID, cli.engine.Calls[len(cli.engine.Calls)-1], "the old container runs again")
	assert.Empty(t, cli.removed, "no image is cleaned up after a rollback")
}

// A rollback stopped the old container, so anything joined to its network
// namespace lost it and has to rejoin the restored one.
func TestUpdateContainerRollbackRejoinsDependents(t *testing.T) {
	cli := newUpdateClient(t, appInspect(nil))
	cli.engine.StartErrFor = "new1"
	cli.engine.Containers["dep-id"] = docker_types.InspectResponse{
		ID:         "dep-id",
		Name:       "/dep",
		State:      &docker_types.State{Running: true},
		Config:     &docker_types.Config{},
		HostConfig: &docker_types.HostConfig{NetworkMode: "container:" + appID},
	}
	cli.dependents = []string{"dep-id"}
	run := runUpdate(cli, container.UpdateOptions{})

	require.Error(t, run.err)
	assert.Equal(t, "rolled-back", run.last.Status)
	assert.Equal(t, []string{"stop dep-id", "remove dep-id", "create dep container:" + appID, "start new-dep"}, cli.calls)
}

func TestUpdateContainerRejoinsDependentsAfterCommit(t *testing.T) {
	cli := newUpdateClient(t, appInspect(nil))
	cli.engine.NewIDs = []string{"new-app"}
	cli.engine.Containers["dep-id"] = docker_types.InspectResponse{
		ID:         "dep-id",
		Name:       "/dep",
		State:      &docker_types.State{Running: true},
		Config:     &docker_types.Config{},
		HostConfig: &docker_types.HostConfig{NetworkMode: "container:" + appID},
	}
	cli.dependents = []string{"dep-id"}
	run := runUpdate(cli, container.UpdateOptions{})

	require.NoError(t, run.err)
	assert.Equal(t, []string{"stop dep-id", "remove dep-id", "create dep container:new-app", "start new-dep"}, cli.calls)
}

// A scheduled update lists every container, stopped ones included. A one-shot
// job that already exited moves to the new image and stays stopped.
func TestUpdateContainerStoppedOneShotIsDone(t *testing.T) {
	old := appInspect(nil)
	old.State = &docker_types.State{Status: "exited", ExitCode: 0}
	cli := newUpdateClient(t, old)
	cli.engine.NewState = &docker_types.State{Status: "created"}
	run := runUpdate(cli, container.UpdateOptions{})

	require.NoError(t, run.err)
	assert.True(t, run.updated)
	assert.Equal(t, "done", run.last.Status)
	assert.NotContains(t, run.statuses, "verifying")
	for _, call := range cli.engine.Calls {
		assert.NotContains(t, call, "start", "the replacement is left stopped")
	}
}

func TestUpdateContainerCleanupRemovesTheImageBeforeThePrevious(t *testing.T) {
	cli := newUpdateClient(t, appInspect(map[string]string{container.PreviousImageLabel: olderImage}))
	run := runUpdate(cli, container.UpdateOptions{Cleanup: true})

	require.NoError(t, run.err)
	assert.Equal(t, "done", run.last.Status)
	assert.Equal(t, []string{olderImage}, cli.removed, "the image just replaced stays as the rollback target")
}

// Removing by id without force would untag and delete an image whose tags
// share one repository, so cleanup only touches an untagged leftover.
func TestUpdateContainerCleanupKeepsATaggedImage(t *testing.T) {
	cli := newUpdateClient(t, appInspect(map[string]string{container.PreviousImageLabel: olderImage}))
	cli.images[olderImage] = image.InspectResponse{ID: olderImage, RepoTags: []string{"myapp:1.4.0"}}
	run := runUpdate(cli, container.UpdateOptions{Cleanup: true})

	require.NoError(t, run.err)
	assert.Equal(t, "done", run.last.Status)
	assert.Empty(t, cli.removed)
}

func TestUpdateContainerCleanupImageAlreadyGone(t *testing.T) {
	cli := newUpdateClient(t, appInspect(map[string]string{container.PreviousImageLabel: olderImage}))
	delete(cli.images, olderImage)
	run := runUpdate(cli, container.UpdateOptions{Cleanup: true})

	require.NoError(t, run.err)
	assert.Equal(t, "done", run.last.Status)
	assert.Empty(t, cli.removed)
}

func TestUpdateContainerCleanupImageInUse(t *testing.T) {
	cli := newUpdateClient(t, appInspect(map[string]string{container.PreviousImageLabel: olderImage}))
	cli.removeErr = errors.New("conflict: unable to delete 111111111111 (cannot be forced) - image is being used by running container")
	run := runUpdate(cli, container.UpdateOptions{Cleanup: true})

	require.NoError(t, run.err, "a refused removal never fails the update")
	assert.True(t, run.updated)
	assert.Equal(t, "done", run.last.Status)
	assert.Equal(t, []string{olderImage}, cli.removed)
}

func TestUpdateContainerCleanupOptOutLabel(t *testing.T) {
	cli := newUpdateClient(t, appInspect(map[string]string{
		container.PreviousImageLabel: olderImage,
		container.UpdateCleanupLabel: "false",
	}))
	run := runUpdate(cli, container.UpdateOptions{Cleanup: true})

	require.NoError(t, run.err)
	assert.Empty(t, cli.removed)
}

func TestUpdateContainerCleanupFirstUpdateRemovesNothing(t *testing.T) {
	cli := newUpdateClient(t, appInspect(nil))
	run := runUpdate(cli, container.UpdateOptions{Cleanup: true})

	require.NoError(t, run.err)
	assert.Empty(t, cli.removed, "the container never went through an update, so there is no image before the previous one")
}

func TestUpdateContainerCleanupNeverTheImageJustReplaced(t *testing.T) {
	cli := newUpdateClient(t, appInspect(map[string]string{container.PreviousImageLabel: oldImageID}))
	run := runUpdate(cli, container.UpdateOptions{Cleanup: true})

	require.NoError(t, run.err)
	assert.Empty(t, cli.removed)
}

func TestUpdateContainerPullErrorDetail(t *testing.T) {
	cli := newUpdateClient(t, appInspect(nil))
	cli.pullBody = `{"status":"Pulling from library/nginx","id":"latest"}` + "\n" +
		`{"errorDetail":{"message":"manifest unknown"},"error":"manifest unknown"}`
	run := runUpdate(cli, container.UpdateOptions{})

	require.ErrorContains(t, run.err, "manifest unknown")
	assert.False(t, run.updated)
	assert.Equal(t, "error", run.last.Status)
	assert.Contains(t, run.last.Error, "manifest unknown")
	assert.Empty(t, cli.engine.Calls, "nothing is recreated")
}

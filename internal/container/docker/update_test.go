package docker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/container/docker/swap"
	"github.com/amir20/dozzle/internal/container/docker/swap/swaptest"
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

	pullBody string
	// images are shared with engine, which the image cleanup runs against.
	images     map[string]image.InspectResponse
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

	images := map[string]image.InspectResponse{
		oldImageID: {ID: oldImageID, RepoDigests: []string{"ghcr.io/other/nginx@sha256:aaa", "nginx@sha256:bbb"}},
		// Left dangling by the update before last.
		olderImage: {ID: olderImage},
	}
	engine := swaptest.New(old)
	engine.Images = images
	return &updateClient{
		engine:   engine,
		pullBody: `{"status":"Downloading","id":"layer1","progressDetail":{"current":1,"total":2}}`,
		images:   images,
	}
}

type updateRun struct {
	statuses []string
	last     container.UpdateProgress
	updated  bool
	err      error
}

func runUpdate(cli *updateClient) updateRun {
	svc := &Service{client: cli}
	ch := make(chan container.UpdateProgress, 100)
	updated, err := svc.UpdateContainer(context.Background(), container.Container{ID: appID, Name: "app"}, ch)
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
	run := runUpdate(cli)

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
	assert.Empty(t, cli.engine.RemovedImages, "the first update has no image before the previous one")

	require.NotNil(t, run.last.Result, "done carries what the update changed")
	assert.Equal(t, container.UpdateResult{
		OldID:       appID[:12],
		NewID:       "new1",
		FromImageID: oldImageID,
		ToImageID:   newImageID,
		FromDigest:  "nginx@sha256:bbb",
	}, *run.last.Result)
}

func TestUpdateContainerLocalImageHasNoPreviousRef(t *testing.T) {
	cli := newUpdateClient(t, appInspect(map[string]string{container.PreviousRefLabel: "nginx@sha256:stale"}))
	cli.images[oldImageID] = image.InspectResponse{ID: oldImageID}
	run := runUpdate(cli)

	require.NoError(t, run.err)
	labels := cli.engine.Created[0].Config.Labels
	assert.Equal(t, oldImageID, labels[container.PreviousImageLabel])
	assert.NotContains(t, labels, container.PreviousRefLabel, "a built image has no digest, and the old container's is stale")
}

// A rollback stopped the old container, so anything joined to its network
// namespace lost it and has to rejoin the restored one.
func TestUpdateContainerRollbackRejoinsDependents(t *testing.T) {
	cli := newUpdateClient(t, appInspect(map[string]string{container.PreviousImageLabel: olderImage}))
	cli.engine.StartErrFor = "new1"
	cli.engine.Containers["dep-id"] = docker_types.InspectResponse{
		ID:         "dep-id",
		Name:       "/dep",
		State:      &docker_types.State{Running: true},
		Config:     &docker_types.Config{},
		HostConfig: &docker_types.HostConfig{NetworkMode: "container:" + appID},
	}
	cli.dependents = []string{"dep-id"}
	run := runUpdate(cli)

	require.ErrorContains(t, run.err, "rolled back")
	assert.False(t, run.updated)
	assert.Equal(t, []string{"pulling", "recreating", "rolled-back"}, run.statuses)
	assert.Contains(t, run.last.Error, "start replacement")
	assert.Equal(t, []string{"stop dep-id", "remove dep-id", "create dep container:" + appID, "start new-dep"}, cli.calls)
	assert.Empty(t, cli.engine.RemovedImages, "no image is cleaned up after a rollback")
	require.NotNil(t, run.last.Result)
	assert.True(t, run.last.Result.RolledBack)
	assert.Equal(t, appID[:12], run.last.Result.NewID, "the old container runs again")
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
	run := runUpdate(cli)

	require.NoError(t, run.err)
	assert.Equal(t, []string{"stop dep-id", "remove dep-id", "create dep container:new-app", "start new-dep"}, cli.calls)
}

// A stopped container is refused before anything is pulled, whoever asked.
func TestUpdateContainerRefusesStoppedContainer(t *testing.T) {
	old := appInspect(nil)
	old.State = &docker_types.State{Status: "exited", ExitCode: 0}
	cli := newUpdateClient(t, old)
	cli.pullBody = "not json, so a pull would fail the test differently"
	run := runUpdate(cli)

	require.ErrorIs(t, run.err, container.ErrNotRunning)
	assert.False(t, run.updated)
	assert.Equal(t, []string{"error"}, run.statuses)
	assert.Equal(t, container.ErrNotRunning.Error(), run.last.Error)
	assert.Empty(t, cli.engine.Calls)
}

// What CleanupImage removes is tested in package swap; this is that it runs
// once the update committed.
func TestUpdateContainerCleansUpAfterCommit(t *testing.T) {
	cli := newUpdateClient(t, appInspect(map[string]string{container.PreviousImageLabel: olderImage}))
	run := runUpdate(cli)

	require.NoError(t, run.err)
	assert.Equal(t, "done", run.last.Status)
	assert.Equal(t, []string{olderImage}, cli.engine.RemovedImages, "the image just replaced stays as the rollback target")
}

func TestUpdateContainerPullErrorDetail(t *testing.T) {
	cli := newUpdateClient(t, appInspect(nil))
	cli.pullBody = `{"status":"Pulling from library/nginx","id":"latest"}` + "\n" +
		`{"errorDetail":{"message":"manifest unknown"},"error":"manifest unknown"}`
	run := runUpdate(cli)

	require.ErrorContains(t, run.err, "manifest unknown")
	assert.False(t, run.updated)
	assert.Equal(t, "error", run.last.Status)
	assert.Contains(t, run.last.Error, "manifest unknown")
	assert.Empty(t, cli.engine.Calls, "nothing is recreated")
}

// The request ending mid-swap must not lose the update's record: the final
// progress carries the Result, and it is recorded even though nobody reads it.
func TestUpdateRecordedAfterRequestCancelled(t *testing.T) {
	cli := newUpdateClient(t, appInspect(nil))
	svc := &Service{client: cli}
	c := container.Container{ID: appID, Name: "app", Host: fmt.Sprintf("cancelled-%d", time.Now().UnixNano())}
	cs := container.NewContainerService(svc, c)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	progressCh := make(chan container.UpdateProgress)
	go func() {
		for p := range progressCh {
			if p.Status == container.UpdateVerifying {
				// The client goes away and stops reading.
				cancel()
				return
			}
		}
	}()

	updated, err := cs.Update(ctx, container.UpdateSourceDozzle, progressCh)
	require.NoError(t, err)
	assert.True(t, updated)

	record, ok := container.Updates.Latest(c.Host, "new1")
	require.True(t, ok, "the update is recorded")
	assert.Equal(t, container.UpdateSourceDozzle, record.Source)
	assert.Equal(t, appID[:12], record.OldID)
}

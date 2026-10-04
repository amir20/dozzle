package swap

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/container"
	dockerspec "github.com/moby/docker-image-spec/specs-go/v1"
	dcontainer "github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/image"
	"github.com/moby/moby/api/types/mount"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	appID    = "aaaaaaaaaaaa1111111111111111111111111111111111111111111111111111"
	oldImgID = "sha256:0000000000000000000000000000000000000000000000000000000000000001"
)

type notFoundErr struct{}

func (notFoundErr) Error() string { return "not found" }
func (notFoundErr) NotFound()     {}

// fakeDocker records what was asked of the daemon. It keeps just enough state
// for a swap: containers by id, what creates asked for, and failures to inject.
type fakeDocker struct {
	mu    sync.Mutex
	calls []string

	containers map[string]dcontainer.InspectResponse

	createErr   error
	startErrFor string // container id whose start fails
	created     []client.ContainerCreateOptions
	forced      []string // ids removed with Force
	// goneAfter makes an inspect of that id report it still present for that
	// many calls, the way a --rm removal still in progress does.
	goneAfter map[string]int
	// newState is what an inspect of a created container reports.
	newState *dcontainer.State
	// stopErr fails every stop, after the container did stop, unless
	// stopLeftRunning says it did not.
	stopErr         error
	stopLeftRunning bool
	// removeErrFor is a container id whose removal fails and leaves it there.
	removeErrFor string

	images         map[string]image.InspectResponse
	removedImages  []string
	imageRemoveErr error
}

func (f *fakeDocker) ImageInspect(_ context.Context, id string, _ ...client.ImageInspectOption) (client.ImageInspectResult, error) {
	img, ok := f.images[id]
	if !ok {
		return client.ImageInspectResult{}, notFoundErr{}
	}
	return client.ImageInspectResult{InspectResponse: img}, nil
}

func (f *fakeDocker) ImageRemove(_ context.Context, id string, opts client.ImageRemoveOptions) (client.ImageRemoveResult, error) {
	if opts.Force || opts.PruneChildren {
		return client.ImageRemoveResult{}, errors.New("cleanup must never force or prune")
	}
	f.removedImages = append(f.removedImages, id)
	return client.ImageRemoveResult{}, f.imageRemoveErr
}

func (f *fakeDocker) record(format string, args ...any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, fmt.Sprintf(format, args...))
}

func (f *fakeDocker) ContainerInspect(_ context.Context, id string, _ client.ContainerInspectOptions) (client.ContainerInspectResult, error) {
	if n, ok := f.goneAfter[id]; ok {
		if n <= 0 {
			delete(f.goneAfter, id)
			delete(f.containers, id)
			f.record("gone %s", id)
		} else {
			f.goneAfter[id] = n - 1
		}
	}
	c, ok := f.containers[id]
	if !ok {
		return client.ContainerInspectResult{}, notFoundErr{}
	}
	return client.ContainerInspectResult{Container: c}, nil
}

func (f *fakeDocker) ContainerCreate(_ context.Context, opts client.ContainerCreateOptions) (client.ContainerCreateResult, error) {
	f.record("create %s", opts.Name)
	f.created = append(f.created, opts)
	if f.createErr != nil {
		err := f.createErr
		f.createErr = nil
		return client.ContainerCreateResult{}, err
	}
	id := fmt.Sprintf("new%d", len(f.created))
	state := f.newState
	if state == nil {
		state = &dcontainer.State{Running: true, StartedAt: "t0"}
	}
	f.containers[id] = dcontainer.InspectResponse{ID: id, State: state}
	return client.ContainerCreateResult{ID: id}, nil
}

func (f *fakeDocker) ContainerStart(_ context.Context, id string, _ client.ContainerStartOptions) (client.ContainerStartResult, error) {
	f.record("start %s", id)
	if id == f.startErrFor {
		return client.ContainerStartResult{}, errors.New("boom")
	}
	return client.ContainerStartResult{}, nil
}

func (f *fakeDocker) ContainerStop(_ context.Context, id string, _ client.ContainerStopOptions) (client.ContainerStopResult, error) {
	f.record("stop %s", id)
	if _, removing := f.goneAfter[id]; removing {
		return client.ContainerStopResult{}, nil
	}
	if c, ok := f.containers[id]; ok && c.HostConfig != nil && c.HostConfig.AutoRemove && !f.stopLeftRunning {
		delete(f.containers, id)
	} else if ok && c.State != nil && !f.stopLeftRunning {
		state := *c.State
		state.Running = false
		c.State = &state
		f.containers[id] = c
	}
	return client.ContainerStopResult{}, f.stopErr
}

func (f *fakeDocker) ContainerRename(_ context.Context, id string, opts client.ContainerRenameOptions) (client.ContainerRenameResult, error) {
	f.record("rename %s %s", id, opts.NewName)
	return client.ContainerRenameResult{}, nil
}

func (f *fakeDocker) ContainerRemove(_ context.Context, id string, opts client.ContainerRemoveOptions) (client.ContainerRemoveResult, error) {
	f.record("remove %s volumes=%v", id, opts.RemoveVolumes)
	if opts.Force {
		f.forced = append(f.forced, id)
	}
	if id == f.removeErrFor {
		return client.ContainerRemoveResult{}, errors.New("remove failed")
	}
	delete(f.containers, id)
	return client.ContainerRemoveResult{}, nil
}

func appContainer() dcontainer.InspectResponse {
	return dcontainer.InspectResponse{
		ID:    appID,
		Name:  "/dozzle",
		Image: oldImgID,
		State: &dcontainer.State{Running: true, Status: "running"},
		Config: &dcontainer.Config{
			Hostname:   "aaaaaaaaaaaa",
			Image:      "amir20/dozzle:latest",
			Env:        []string{"PATH=/bin", "DOZZLE_LEVEL=debug"},
			Entrypoint: []string{"/dozzle"},
			Labels:     map[string]string{"org.opencontainers.image.version": "v1", "com.docker.compose.service": "dozzle"},
			Volumes:    map[string]struct{}{"/data": {}, "/cache": {}},
		},
		HostConfig: &dcontainer.HostConfig{
			NetworkMode: "app_default",
			Binds:       []string{"/var/run/docker.sock:/var/run/docker.sock", "named:/named"},
			Mounts:      []mount.Mount{{Type: mount.TypeVolume, Target: "/cache"}},
		},
		Mounts: []dcontainer.MountPoint{
			{Type: mount.TypeBind, Source: "/var/run/docker.sock", Destination: "/var/run/docker.sock", RW: true},
			{Type: mount.TypeVolume, Name: "named", Destination: "/named", RW: true},
			{Type: mount.TypeVolume, Name: "anon-data", Destination: "/data", RW: true},
			{Type: mount.TypeVolume, Name: "anon-cache", Destination: "/cache", RW: false},
		},
		NetworkSettings: &dcontainer.NetworkSettings{
			Networks: map[string]*network.EndpointSettings{
				"app_default": {Aliases: []string{"dozzle", "aaaaaaaaaaaa"}, NetworkID: "runtime", EndpointID: "runtime"},
			},
		},
	}
}

func oldImage() image.InspectResponse {
	return image.InspectResponse{ID: oldImgID, Config: &dockerspec.DockerOCIImageConfig{
		Env:        []string{"PATH=/bin"},
		Entrypoint: []string{"/dozzle"},
		Labels:     map[string]string{"org.opencontainers.image.version": "v1"},
	}}
}

func newFake() *fakeDocker {
	return &fakeDocker{containers: map[string]dcontainer.InspectResponse{appID: appContainer()}}
}

func fastTimings(t *testing.T) {
	prev := []time.Duration{StableFor, HealthTimeout, GoneTimeout, PollInterval}
	StableFor, HealthTimeout, GoneTimeout, PollInterval = 5*time.Millisecond, 20*time.Millisecond, 20*time.Millisecond, time.Millisecond
	t.Cleanup(func() { StableFor, HealthTimeout, GoneTimeout, PollInterval = prev[0], prev[1], prev[2], prev[3] })
}

func swapApp(f *fakeDocker, opts Options) (Result, error) {
	return Swap(context.Background(), f, f.containers[appID], opts)
}

func TestReplacementSpec(t *testing.T) {
	old := appContainer()
	img := oldImage()
	spec := ReplacementSpec(old, &img, "dozzle")

	assert.Equal(t, "dozzle", spec.Name)
	assert.Equal(t, "amir20/dozzle:latest", spec.Config.Image)
	assert.Empty(t, spec.Config.Hostname, "hostname derived from the old id is dropped")
	assert.Equal(t, []string{"DOZZLE_LEVEL=debug"}, spec.Config.Env, "image env is dropped, user env kept")
	assert.Nil(t, spec.Config.Entrypoint, "image entrypoint is left to the new image")
	assert.Equal(t, map[string]string{"com.docker.compose.service": "dozzle"}, spec.Config.Labels)

	assert.Equal(t, []string{"/var/run/docker.sock:/var/run/docker.sock", "named:/named"}, spec.HostConfig.Binds)
	assert.Equal(t, []mount.Mount{
		{Type: mount.TypeVolume, Source: "anon-data", Target: "/data"},
		{Type: mount.TypeVolume, Source: "anon-cache", Target: "/cache", ReadOnly: true},
	}, spec.HostConfig.Mounts, "anonymous volumes are reused by name")
	assert.Nil(t, spec.Config.Volumes)

	require.NotNil(t, spec.NetworkingConfig)
	ep := spec.NetworkingConfig.EndpointsConfig["app_default"]
	require.NotNil(t, ep)
	assert.Equal(t, []string{"dozzle"}, ep.Aliases)
	assert.Empty(t, ep.NetworkID)
	assert.Empty(t, ep.EndpointID)

	// The inspect is not mutated.
	assert.Equal(t, "aaaaaaaaaaaa", old.Config.Hostname)
	assert.Len(t, old.Config.Volumes, 2)
	assert.Len(t, old.NetworkSettings.Networks["app_default"].Aliases, 2)
}

func TestReplacementSpecSharedNamespace(t *testing.T) {
	old := appContainer()
	old.HostConfig.NetworkMode = "container:vpn"
	old.HostConfig.PortBindings = nil
	spec := ReplacementSpec(old, nil, "dozzle")
	assert.Nil(t, spec.NetworkingConfig)
	assert.Empty(t, spec.Config.Hostname)
}

func TestSwapSuccess(t *testing.T) {
	fastTimings(t)
	f := newFake()
	verifying := 0
	result, err := swapApp(f, Options{OnVerifying: func() { verifying++ }})
	require.NoError(t, err)
	assert.Equal(t, Result{NewID: "new1", OldStopped: true}, result)
	assert.Equal(t, 1, verifying)
	assert.Equal(t, []string{
		"rename " + appID + " dozzle-dozzle-old-aaaaaaaaaaaa",
		"create dozzle",
		"stop " + appID,
		"start new1",
		"remove " + appID + " volumes=false",
	}, f.calls)
}

// A stopped container may be stopped on purpose, so it is never swapped:
// nothing is renamed, created or removed.
func TestSwapRefusesStoppedContainer(t *testing.T) {
	fastTimings(t)
	f := newFake()
	old := f.containers[appID]
	old.State = &dcontainer.State{Status: "exited", ExitCode: 0}
	f.containers[appID] = old
	result, err := swapApp(f, Options{})
	require.ErrorIs(t, err, container.ErrNotRunning)
	assert.Equal(t, Result{}, result)
	assert.Empty(t, f.calls)
}

func TestSwapStampsLabels(t *testing.T) {
	fastTimings(t)
	f := newFake()
	_, err := swapApp(f, Options{Labels: map[string]string{
		"dev.dozzle.previous-image":  oldImgID,
		"com.docker.compose.service": "", // empty removes
	}})
	require.NoError(t, err)
	labels := f.created[0].Config.Labels
	assert.Equal(t, oldImgID, labels["dev.dozzle.previous-image"])
	assert.NotContains(t, labels, "com.docker.compose.service")
	assert.Equal(t, "v1", labels["org.opencontainers.image.version"], "no old image given, so its labels stay")
	assert.NotContains(t, appContainer().Config.Labels, "dev.dozzle.previous-image", "the inspect is not mutated")
}

// A rollback runs the previous image by id, and the tag it followed moves to
// a label so update checks still see the tag.
func TestSwapRunsGivenImage(t *testing.T) {
	fastTimings(t)
	f := newFake()
	previous := "sha256:0000000000000000000000000000000000000000000000000000000000000002"
	_, err := swapApp(f, Options{Image: previous})
	require.NoError(t, err)
	spec := f.created[0]
	assert.Equal(t, previous, spec.Config.Image)
	assert.Equal(t, "amir20/dozzle:latest", spec.Config.Labels[ImageRefLabel])
	assert.Equal(t, "amir20/dozzle:latest", ImageRef(spec.Config), "the replacement still follows its tag")

	// Swapping it forward again drops the pin.
	again := ReplacementSpec(dcontainer.InspectResponse{ID: "new1", Config: spec.Config, HostConfig: spec.HostConfig}, nil, "dozzle")
	assert.Equal(t, "amir20/dozzle:latest", again.Config.Image)
	assert.NotContains(t, again.Config.Labels, ImageRefLabel)
}

func TestSwapRollbackOnCreateFailure(t *testing.T) {
	fastTimings(t)
	f := newFake()
	f.createErr = errors.New("create failed")
	result, err := swapApp(f, Options{})
	require.ErrorContains(t, err, "create failed")
	assert.Equal(t, Result{RolledBack: true, RestoredID: appID}, result, "never stopped, so dependents kept their namespace")
	assert.Equal(t, []string{
		"rename " + appID + " dozzle-dozzle-old-aaaaaaaaaaaa",
		"create dozzle",
		"rename " + appID + " dozzle",
	}, f.calls, "old container was never stopped, so it is only renamed back")
}

func TestSwapRollbackOnStartFailure(t *testing.T) {
	fastTimings(t)
	f := newFake()
	f.startErrFor = "new1"
	verifying := 0
	result, err := swapApp(f, Options{OnVerifying: func() { verifying++ }})
	require.ErrorContains(t, err, "start replacement")
	assert.Equal(t, Result{RolledBack: true, RestoredID: appID, OldStopped: true}, result)
	assert.Zero(t, verifying, "never started, so never verified")
	assert.Equal(t, []string{
		"rename " + appID + " dozzle-dozzle-old-aaaaaaaaaaaa",
		"create dozzle",
		"stop " + appID,
		"start new1",
		"remove new1 volumes=false",
		"rename " + appID + " dozzle",
		"start " + appID,
	}, f.calls)
	assert.Equal(t, []string{"new1"}, f.forced)
}

func TestSwapRollbackWhenReplacementExits(t *testing.T) {
	fastTimings(t)
	f := newFake()
	f.newState = &dcontainer.State{Status: "exited", ExitCode: 1}
	result, err := swapApp(f, Options{})
	require.ErrorContains(t, err, "exit code 1")
	assert.True(t, result.RolledBack)
	assert.Equal(t, "start "+appID, f.calls[len(f.calls)-1])
}

func TestSwapRollbackWhenUnhealthy(t *testing.T) {
	fastTimings(t)
	f := newFake()
	f.newState = &dcontainer.State{Running: true, StartedAt: "t0", Health: &dcontainer.Health{Status: dcontainer.Unhealthy}}
	result, err := swapApp(f, Options{})
	require.ErrorContains(t, err, "unhealthy")
	assert.True(t, result.RolledBack)
	assert.Contains(t, f.calls, "remove new1 volumes=false")
}

func TestSwapRollbackWhenNeverHealthy(t *testing.T) {
	fastTimings(t)
	f := newFake()
	f.newState = &dcontainer.State{Running: true, StartedAt: "t0", Health: &dcontainer.Health{Status: dcontainer.Starting}}
	_, err := swapApp(f, Options{})
	require.ErrorContains(t, err, "did not become healthy")
}

func TestSwapWaitsForHealthy(t *testing.T) {
	fastTimings(t)
	f := newFake()
	f.newState = &dcontainer.State{Running: true, StartedAt: "t0", Health: &dcontainer.Health{Status: dcontainer.Healthy}}
	result, err := swapApp(f, Options{})
	require.NoError(t, err)
	assert.Equal(t, "new1", result.NewID)
}

func TestSwapAutoRemoveRollbackRecreatesOld(t *testing.T) {
	fastTimings(t)
	f := newFake()
	app := f.containers[appID]
	app.HostConfig.AutoRemove = true
	f.containers[appID] = app
	f.startErrFor = "new1"

	result, err := swapApp(f, Options{Labels: map[string]string{"dev.dozzle.previous-image": oldImgID}})
	require.Error(t, err)
	assert.Equal(t, Result{RolledBack: true, RestoredID: "new2", OldStopped: true}, result)
	assert.Equal(t, []string{
		"rename " + appID + " dozzle-dozzle-old-aaaaaaaaaaaa",
		"create dozzle",
		"stop " + appID,
		"start new1",
		"remove new1 volumes=false",
		"create dozzle",
		"start new2",
	}, f.calls)
	restored := f.created[1]
	assert.Equal(t, oldImgID, restored.Config.Image)
	assert.Equal(t, "amir20/dozzle:latest", ImageRef(restored.Config), "the tag survives so the next update can pull it")
	assert.NotContains(t, restored.Config.Labels, "dev.dozzle.previous-image", "the restored container is the old one, not the update")
	assert.Contains(t, restored.HostConfig.Mounts, mount.Mount{Type: mount.TypeVolume, Source: "anon-data", Target: "/data"})
}

func TestSwapAutoRemoveSuccess(t *testing.T) {
	fastTimings(t)
	f := newFake()
	app := f.containers[appID]
	app.HostConfig.AutoRemove = true
	f.containers[appID] = app

	_, err := swapApp(f, Options{})
	require.NoError(t, err)
	assert.Equal(t, []string{
		"rename " + appID + " dozzle-dozzle-old-aaaaaaaaaaaa",
		"create dozzle",
		"stop " + appID,
		"start new1",
	}, f.calls, "the replacement exists before the stop, and there is nothing left to remove")
}

func TestSwapIncompleteConfig(t *testing.T) {
	f := newFake()
	_, err := Swap(context.Background(), f, dcontainer.InspectResponse{ID: appID}, Options{})
	require.ErrorContains(t, err, "incomplete container config")
	assert.Empty(t, f.calls)
}

// The old --rm container is still removing itself when the forward wait gave
// up. Removing the replacement then would leave its anonymous volumes
// unreferenced for that removal to delete.
func slowRemoval(t *testing.T, inspectsUntilGone int) (*fakeDocker, *swap) {
	fastTimings(t)
	f := newFake()
	old := f.containers[appID]
	old.HostConfig.AutoRemove = true
	f.containers[appID] = old
	f.containers["new1"] = dcontainer.InspectResponse{ID: "new1", State: &dcontainer.State{}}
	f.goneAfter = map[string]int{appID: inspectsUntilGone}
	s := newSwap(f, old, Options{})
	s.renamed, s.stopped, s.newID = true, true, "new1"
	return f, s
}

func TestRollbackWaitsForOldRemovalBeforeRemovingReplacement(t *testing.T) {
	f, s := slowRemoval(t, 3)
	require.NoError(t, s.rollback(context.Background()))
	assert.Equal(t, []string{
		"gone " + appID,
		"remove new1 volumes=false",
		"create dozzle",
		"start new1", // the fake numbers ids by creates, and this swap made none
	}, f.calls)
}

func TestRollbackKeepsReplacementWhileOldStillRemoving(t *testing.T) {
	f, s := slowRemoval(t, 1_000_000)
	require.ErrorContains(t, s.rollback(context.Background()), "keeping replacement")
	assert.NotContains(t, f.calls, "remove new1 volumes=false")
}

// A stop that errors may still have stopped the container, so the rollback
// starts it regardless: starting one that is still running is a no-op.
func TestSwapRollbackStartsOldWhenStopErrored(t *testing.T) {
	fastTimings(t)
	f := newFake()
	f.stopErr = errors.New("stop timed out")
	result, err := swapApp(f, Options{})
	require.ErrorContains(t, err, "stop old container")
	assert.Equal(t, Result{RolledBack: true, RestoredID: appID, OldStopped: true}, result)
	assert.Equal(t, []string{
		"rename " + appID + " dozzle-dozzle-old-aaaaaaaaaaaa",
		"create dozzle",
		"stop " + appID,
		"remove new1 volumes=false",
		"rename " + appID + " dozzle",
		"start " + appID,
	}, f.calls)
}

// A stop that errors with the old container still running is no stop: the
// rollback puts the name back without waiting on a --rm removal that is not
// coming, and reports nothing stopped.
func TestSwapRollbackStopErroredStillRunning(t *testing.T) {
	fastTimings(t)
	f := newFake()
	old := f.containers[appID]
	old.HostConfig.AutoRemove = true
	f.containers[appID] = old
	f.stopErr = errors.New("stop timed out")
	f.stopLeftRunning = true
	result, err := swapApp(f, Options{})
	require.ErrorContains(t, err, "stop old container")
	assert.Equal(t, Result{RolledBack: true, RestoredID: appID}, result)
	assert.Equal(t, []string{
		"rename " + appID + " dozzle-dozzle-old-aaaaaaaaaaaa",
		"create dozzle",
		"stop " + appID,
		"remove new1 volumes=false",
		"rename " + appID + " dozzle",
	}, f.calls)
}

// A replacement that cannot be removed keeps the name, so the rollback gives
// up, but it still starts the old container under its temporary name.
func TestSwapRollbackStartsOldWhenReplacementStays(t *testing.T) {
	fastTimings(t)
	f := newFake()
	f.startErrFor = "new1"
	f.removeErrFor = "new1"
	_, err := swapApp(f, Options{})
	require.ErrorContains(t, err, "rollback failed: remove replacement")
	assert.Equal(t, "start "+appID, f.calls[len(f.calls)-1])
	assert.NotContains(t, f.calls, "rename "+appID+" dozzle", "the name is still taken")
}

func TestRollbackKeepingReplacementStartsOld(t *testing.T) {
	f, s := slowRemoval(t, 1_000_000)
	require.Error(t, s.rollback(context.Background()))
	assert.Equal(t, "start "+appID, f.calls[len(f.calls)-1], "tried, though a --rm container still removing itself may refuse")
}

func TestCleanupImage(t *testing.T) {
	const older = "sha256:0000000000000000000000000000000000000000000000000000000000000009"
	tests := []struct {
		name      string
		previous  string // the outgoing container's previous-image label
		images    map[string]image.InspectResponse
		removeErr error
		removed   []string
	}{
		{name: "removes the image before the previous one", previous: older, images: map[string]image.InspectResponse{older: {ID: older}}, removed: []string{older}},
		// Removing by id without force would untag and delete an image whose
		// tags share one repository.
		{name: "keeps a tagged image", previous: older, images: map[string]image.InspectResponse{older: {ID: older, RepoTags: []string{"myapp:1.4.0"}}}},
		{name: "image already gone", previous: older},
		{name: "a refused removal is only logged", previous: older, images: map[string]image.InspectResponse{older: {ID: older}}, removeErr: errors.New("image is being used by running container"), removed: []string{older}},
		{name: "first update removes nothing", images: map[string]image.InspectResponse{older: {ID: older}}},
		{name: "never the image just replaced", previous: oldImgID, images: map[string]image.InspectResponse{oldImgID: {ID: oldImgID}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFake()
			f.images, f.imageRemoveErr = tt.images, tt.removeErr
			old := appContainer()
			if tt.previous != "" {
				old.Config.Labels[container.PreviousImageLabel] = tt.previous
			}
			CleanupImage(context.Background(), f, old)
			assert.Equal(t, tt.removed, f.removedImages)
		})
	}
}

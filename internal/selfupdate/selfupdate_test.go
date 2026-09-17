package selfupdate

import (
	"context"
	"errors"
	"fmt"
	"io"
	"iter"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/container"
	dockerspec "github.com/moby/docker-image-spec/specs-go/v1"
	dcontainer "github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/image"
	"github.com/moby/moby/api/types/jsonstream"
	"github.com/moby/moby/api/types/mount"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	selfID   = "aaaaaaaaaaaa1111111111111111111111111111111111111111111111111111"
	oldImgID = "sha256:0000000000000000000000000000000000000000000000000000000000000001"
	newImgID = "sha256:0000000000000000000000000000000000000000000000000000000000000002"
)

type notFoundErr struct{}

func (notFoundErr) Error() string { return "not found" }
func (notFoundErr) NotFound()     {}

type conflictErr struct{}

func (conflictErr) Error() string { return "conflict" }
func (conflictErr) Conflict()     {}

type pullResponse struct{ io.ReadCloser }

func (pullResponse) JSONMessages(context.Context) iter.Seq2[jsonstream.Message, error] {
	return func(func(jsonstream.Message, error) bool) {}
}
func (pullResponse) Wait(context.Context) error { return nil }

type fakeDocker struct {
	mu    sync.Mutex
	calls []string

	containers map[string]dcontainer.InspectResponse
	images     map[string]image.InspectResponse
	pullBody   string

	createErr   error
	startErrFor string // container id whose start fails
	created     []client.ContainerCreateOptions
	forced      []string // ids removed with Force
	// goneAfter makes an inspect of that id report it still present for that
	// many calls, the way a --rm removal still in progress does.
	goneAfter map[string]int
	// newState is what an inspect of a created container reports.
	newState *dcontainer.State

	// service is what a service inspect returns; nil makes it fail the way a
	// worker node's engine does.
	service        *swarm.Service
	serviceUpdates []client.ServiceUpdateOptions
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
	if c, ok := f.containers[id]; ok && c.HostConfig != nil && c.HostConfig.AutoRemove {
		delete(f.containers, id)
	}
	return client.ContainerStopResult{}, nil
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
	delete(f.containers, id)
	return client.ContainerRemoveResult{}, nil
}

func (f *fakeDocker) ImageInspect(_ context.Context, ref string, _ ...client.ImageInspectOption) (client.ImageInspectResult, error) {
	img, ok := f.images[ref]
	if !ok {
		return client.ImageInspectResult{}, notFoundErr{}
	}
	return client.ImageInspectResult{InspectResponse: img}, nil
}

func (f *fakeDocker) ImagePull(_ context.Context, ref string, _ client.ImagePullOptions) (client.ImagePullResponse, error) {
	f.record("pull %s", ref)
	return pullResponse{io.NopCloser(strings.NewReader(f.pullBody))}, nil
}

func (f *fakeDocker) ServiceInspect(_ context.Context, id string, _ client.ServiceInspectOptions) (client.ServiceInspectResult, error) {
	if f.service == nil || f.service.ID != id {
		return client.ServiceInspectResult{}, errors.New("This node is not a swarm manager.")
	}
	return client.ServiceInspectResult{Service: *f.service}, nil
}

func (f *fakeDocker) ServiceUpdate(_ context.Context, id string, opts client.ServiceUpdateOptions) (client.ServiceUpdateResult, error) {
	f.record("service update %s", id)
	f.mu.Lock()
	f.serviceUpdates = append(f.serviceUpdates, opts)
	f.mu.Unlock()
	return client.ServiceUpdateResult{}, nil
}

func (f *fakeDocker) Close() error { return nil }

func dozzleContainer() dcontainer.InspectResponse {
	return dcontainer.InspectResponse{
		ID:    selfID,
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
	self := dozzleContainer()
	return &fakeDocker{
		containers: map[string]dcontainer.InspectResponse{selfID: self},
		images: map[string]image.InspectResponse{
			"amir20/dozzle:latest": {ID: newImgID},
			oldImgID:               oldImage(),
		},
		pullBody: `{"status":"Downloading","id":"layer1","progressDetail":{"current":1,"total":2}}`,
	}
}

func fastTimings(t *testing.T) {
	prev := []time.Duration{stableFor, healthTimeout, goneTimeout, pollInterval}
	stableFor, healthTimeout, goneTimeout, pollInterval = 5*time.Millisecond, 20*time.Millisecond, 20*time.Millisecond, time.Millisecond
	t.Cleanup(func() { stableFor, healthTimeout, goneTimeout, pollInterval = prev[0], prev[1], prev[2], prev[3] })
}

func TestReplacementSpec(t *testing.T) {
	old := dozzleContainer()
	img := oldImage()
	spec := replacementSpec(old, &img, "dozzle")

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
	old := dozzleContainer()
	old.HostConfig.NetworkMode = "container:vpn"
	old.HostConfig.PortBindings = nil
	spec := replacementSpec(old, nil, "dozzle")
	assert.Nil(t, spec.NetworkingConfig)
	assert.Empty(t, spec.Config.Hostname)
}

func TestHelperSpec(t *testing.T) {
	self := dozzleContainer()
	self.Mounts[0].Source = "/run/user/docker.sock"
	spec := helperSpec(self, newImgID)

	assert.Equal(t, "dozzle-self-update-aaaaaaaaaaaa", spec.Name)
	assert.Equal(t, newImgID, spec.Config.Image)
	assert.Equal(t, []string{"self-update", "--target", selfID}, spec.Config.Cmd)
	assert.Equal(t, []string{"/dozzle"}, spec.Config.Entrypoint)
	assert.Equal(t, "true", spec.Config.Labels[HelperLabel])
	assert.True(t, spec.HostConfig.AutoRemove)
	assert.Equal(t, dcontainer.NetworkMode("none"), spec.HostConfig.NetworkMode)
	assert.Equal(t, []string{"/run/user/docker.sock:/var/run/docker.sock"}, spec.HostConfig.Binds)
	assert.Equal(t, []string{"DOZZLE_LEVEL=debug"}, spec.Config.Env)
}

func TestHelperSpecSocketFallback(t *testing.T) {
	self := dozzleContainer()
	self.Mounts = nil
	spec := helperSpec(self, newImgID)
	assert.Equal(t, []string{"/var/run/docker.sock:/var/run/docker.sock"}, spec.HostConfig.Binds)
}

func TestHelperSpecTCP(t *testing.T) {
	self := dozzleContainer()
	self.Config.Env = []string{"DOCKER_HOST=tcp://proxy:2375"}
	spec := helperSpec(self, newImgID)
	assert.Equal(t, dcontainer.NetworkMode("app_default"), spec.HostConfig.NetworkMode)
	assert.Contains(t, spec.Config.Env, "DOCKER_HOST=tcp://proxy:2375")
	assert.Empty(t, spec.HostConfig.Binds, "no certs, so none of Dozzle's binds are passed on")

	self.Config.Env = append(self.Config.Env, "DOCKER_CERT_PATH=/certs/client", "DOCKER_TLS_VERIFY=1")
	self.Mounts = append(self.Mounts,
		dcontainer.MountPoint{Type: mount.TypeBind, Source: "/srv/dozzle", Destination: "/data"},
		dcontainer.MountPoint{Type: mount.TypeBind, Source: "/srv/certs", Destination: "/certs"},
	)
	spec = helperSpec(self, newImgID)
	assert.Equal(t, []string{"/srv/certs:/certs:ro"}, spec.HostConfig.Binds)
	assert.Contains(t, spec.Config.Env, "DOCKER_CERT_PATH=/certs/client")
}

func TestStartUpToDate(t *testing.T) {
	f := newFake()
	f.images["amir20/dozzle:latest"] = image.InspectResponse{ID: oldImgID}
	var got []container.UpdateProgress
	updated, err := start(context.Background(), f, selfID, func(p container.UpdateProgress) { got = append(got, p) })
	require.NoError(t, err)
	assert.False(t, updated)
	assert.Equal(t, "pulling", got[0].Status)
	assert.Equal(t, "up-to-date", got[len(got)-1].Status)
	assert.Equal(t, []string{"pull amir20/dozzle:latest"}, f.calls, "no helper is created")
}

func TestStartLaunchesHelper(t *testing.T) {
	f := newFake()
	var got []string
	updated, err := start(context.Background(), f, selfID, func(p container.UpdateProgress) { got = append(got, p.Status) })
	require.NoError(t, err)
	assert.True(t, updated)
	assert.Equal(t, []string{"pulling", "recreating", "done"}, got)
	assert.Equal(t, []string{"pull amir20/dozzle:latest", "create dozzle-self-update-aaaaaaaaaaaa", "start new1"}, f.calls)
	assert.Equal(t, newImgID, f.created[0].Config.Image)
}

func TestStartReplacesStaleHelper(t *testing.T) {
	f := newFake()
	f.createErr = conflictErr{}
	f.containers["dozzle-self-update-aaaaaaaaaaaa"] = dcontainer.InspectResponse{State: &dcontainer.State{Status: "exited"}}
	updated, err := start(context.Background(), f, selfID, func(container.UpdateProgress) {})
	require.NoError(t, err)
	assert.True(t, updated)
	assert.Contains(t, f.calls, "remove dozzle-self-update-aaaaaaaaaaaa volumes=false")
	assert.Empty(t, f.forced, "a helper that started in the meantime must not be force-removed")
}

func TestStartRefusesConcurrentStart(t *testing.T) {
	startMu.Lock()
	defer startMu.Unlock()
	f := newFake()
	var last container.UpdateProgress
	updated, err := start(context.Background(), f, selfID, func(p container.UpdateProgress) { last = p })
	require.ErrorContains(t, err, "already in progress")
	assert.False(t, updated)
	assert.Equal(t, "error", last.Status)
	assert.Empty(t, f.calls, "nothing is pulled or created while another start runs")
}

func TestStartHelperStartFailureIsNotForced(t *testing.T) {
	f := newFake()
	f.startErrFor = "new1"
	_, err := start(context.Background(), f, selfID, func(container.UpdateProgress) {})
	require.ErrorContains(t, err, "start helper failed")
	assert.Contains(t, f.calls, "remove new1 volumes=false")
	assert.Empty(t, f.forced)
}

func TestStartPullError(t *testing.T) {
	f := newFake()
	f.pullBody = `{"errorDetail":{"message":"manifest unknown"}}`
	var last container.UpdateProgress
	_, err := start(context.Background(), f, selfID, func(p container.UpdateProgress) { last = p })
	require.Error(t, err)
	assert.Equal(t, "error", last.Status)
	assert.Contains(t, last.Error, "manifest unknown")
}

func TestRunSuccess(t *testing.T) {
	fastTimings(t)
	f := newFake()
	require.NoError(t, run(context.Background(), f, selfID))
	assert.Equal(t, []string{
		"rename " + selfID + " dozzle-dozzle-old-aaaaaaaaaaaa",
		"create dozzle",
		"stop " + selfID,
		"start new1",
		"remove " + selfID + " volumes=false",
	}, f.calls)
}

func TestRunRollbackOnCreateFailure(t *testing.T) {
	fastTimings(t)
	f := newFake()
	f.createErr = errors.New("create failed")
	require.Error(t, run(context.Background(), f, selfID))
	assert.Equal(t, []string{
		"rename " + selfID + " dozzle-dozzle-old-aaaaaaaaaaaa",
		"create dozzle",
		"rename " + selfID + " dozzle",
	}, f.calls, "old container was never stopped, so it is only renamed back")
}

func TestRunRollbackOnStartFailure(t *testing.T) {
	fastTimings(t)
	f := newFake()
	f.startErrFor = "new1"
	require.Error(t, run(context.Background(), f, selfID))
	assert.Equal(t, []string{
		"rename " + selfID + " dozzle-dozzle-old-aaaaaaaaaaaa",
		"create dozzle",
		"stop " + selfID,
		"start new1",
		"remove new1 volumes=false",
		"rename " + selfID + " dozzle",
		"start " + selfID,
	}, f.calls)
}

func TestRunRollbackWhenReplacementExits(t *testing.T) {
	fastTimings(t)
	f := newFake()
	f.newState = &dcontainer.State{Status: "exited", ExitCode: 1}
	err := run(context.Background(), f, selfID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exit code 1")
	assert.Equal(t, "start "+selfID, f.calls[len(f.calls)-1])
}

func TestRunRollbackWhenUnhealthy(t *testing.T) {
	fastTimings(t)
	f := newFake()
	f.newState = &dcontainer.State{Running: true, StartedAt: "t0", Health: &dcontainer.Health{Status: dcontainer.Unhealthy}}
	err := run(context.Background(), f, selfID)
	require.ErrorContains(t, err, "unhealthy")
	assert.Contains(t, f.calls, "remove new1 volumes=false")
}

func TestRunAutoRemoveRollbackRecreatesOld(t *testing.T) {
	fastTimings(t)
	f := newFake()
	self := f.containers[selfID]
	self.HostConfig.AutoRemove = true
	f.containers[selfID] = self
	f.startErrFor = "new1"

	require.Error(t, run(context.Background(), f, selfID))
	assert.Equal(t, []string{
		"rename " + selfID + " dozzle-dozzle-old-aaaaaaaaaaaa",
		"create dozzle",
		"stop " + selfID,
		"start new1",
		"remove new1 volumes=false",
		"create dozzle",
		"start new2",
	}, f.calls)
	restored := f.created[1]
	assert.Equal(t, oldImgID, restored.Config.Image)
	assert.Equal(t, "amir20/dozzle:latest", ImageRef(restored.Config), "the tag survives so the next update can pull it")
	assert.Contains(t, restored.HostConfig.Mounts, mount.Mount{Type: mount.TypeVolume, Source: "anon-data", Target: "/data"})

	// The restored container still updates: its next replacement is on the tag.
	again := f.containers[selfID]
	again.Config = restored.Config
	next := replacementSpec(again, nil, "dozzle")
	assert.Equal(t, "amir20/dozzle:latest", next.Config.Image)
	assert.NotContains(t, next.Config.Labels, imageRefLabel)
	g := newFake()
	g.containers[selfID] = again
	ok, reason, ref := support(context.Background(), g, selfID)
	assert.True(t, ok, reason)
	assert.Equal(t, "amir20/dozzle:latest", ref)
}

// The old --rm container is still removing itself when the forward wait gave
// up. Removing the replacement then would leave its anonymous volumes
// unreferenced for that removal to delete.
func slowRemoval(t *testing.T, inspectsUntilGone int) (*fakeDocker, *swap) {
	fastTimings(t)
	f := newFake()
	old := f.containers[selfID]
	old.HostConfig.AutoRemove = true
	f.containers[selfID] = old
	f.containers["new1"] = dcontainer.InspectResponse{ID: "new1", State: &dcontainer.State{}}
	f.goneAfter = map[string]int{selfID: inspectsUntilGone}
	return f, &swap{cli: f, old: old, name: "dozzle", tmpName: oldName("dozzle", selfID), renamed: true, stopped: true, newID: "new1"}
}

func TestRollbackWaitsForOldRemovalBeforeRemovingReplacement(t *testing.T) {
	f, s := slowRemoval(t, 3)
	require.NoError(t, s.rollback(context.Background()))
	assert.Equal(t, []string{
		"gone " + selfID,
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

func TestRunAutoRemoveSuccess(t *testing.T) {
	fastTimings(t)
	f := newFake()
	self := f.containers[selfID]
	self.HostConfig.AutoRemove = true
	f.containers[selfID] = self

	require.NoError(t, run(context.Background(), f, selfID))
	assert.Equal(t, []string{
		"rename " + selfID + " dozzle-dozzle-old-aaaaaaaaaaaa",
		"create dozzle",
		"stop " + selfID,
		"start new1",
	}, f.calls, "the replacement exists before the stop, and there is nothing left to remove")
}

func TestRunNothingToDo(t *testing.T) {
	f := newFake()
	f.images["amir20/dozzle:latest"] = image.InspectResponse{ID: oldImgID}
	require.NoError(t, run(context.Background(), f, selfID))
	assert.Empty(t, f.calls)
}

func TestSupport(t *testing.T) {
	f := newFake()
	ok, reason, img := support(context.Background(), f, selfID)
	assert.True(t, ok)
	assert.Empty(t, reason)
	assert.Equal(t, "amir20/dozzle:latest", img)

	_, reason, _ = support(context.Background(), f, "missing")
	assert.Equal(t, ReasonNoContainer, reason)

	for _, ref := range []string{"amir20/dozzle:v8.12.0", "amir20/dozzle:8.12.0", "amir20/dozzle@sha256:abc", oldImgID} {
		self := f.containers[selfID]
		self.Config.Image = ref
		f.containers[selfID] = self
		ok, reason, _ = support(context.Background(), f, selfID)
		assert.False(t, ok, ref)
		assert.Equal(t, ReasonPinnedTag, reason, ref)
	}

	self := f.containers[selfID]
	self.Config.Image = "ghcr.io/amir20/dozzle:beta"
	self.HostConfig.AutoRemove = true
	f.containers[selfID] = self
	ok, _, _ = support(context.Background(), f, selfID)
	assert.True(t, ok, "moving tags and --rm containers are supported")
}

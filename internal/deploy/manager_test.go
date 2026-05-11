package deploy

import (
	"bytes"
	"context"
	"errors"
	"io"
	"iter"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/moby/moby/api/types/jsonstream"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/api/types/volume"
	"github.com/moby/moby/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testCompose = `
services:
  web:
    image: nginx:latest
`

const testComposeUpdated = `
services:
  web:
    image: nginx:1.27
`

const testComposeV3 = `
services:
  web:
    image: nginx:1.28
`

// fakeDockerClient is a no-op DockerClient for tests. Hooks let tests inject
// delays or errors into specific calls.
type fakeDockerClient struct {
	mu sync.Mutex

	onContainerList func(ctx context.Context) error
	// networksOnList / volumesOnList are returned by NetworkList / VolumeList
	// respectively. Tests set these to simulate existing resources.
	networksOnList []network.Summary
	volumesOnList  []volume.Volume
	// pullBody is returned by ImagePull so tests can exercise pull-stream
	// error handling. Empty body is a successful no-op.
	pullBody []byte

	pullCalls          atomic.Int64
	startCalls         atomic.Int64
	networkRemoveCalls atomic.Int64
	volumeRemoveCalls  atomic.Int64
	volumeCreateCalls  atomic.Int64
}

type fakeImagePullResponse struct {
	io.ReadCloser
}

func (r fakeImagePullResponse) JSONMessages(_ context.Context) iter.Seq2[jsonstream.Message, error] {
	return func(func(jsonstream.Message, error) bool) {}
}

func (r fakeImagePullResponse) Wait(_ context.Context) error {
	_, err := io.Copy(io.Discard, r.ReadCloser)
	return err
}

func (f *fakeDockerClient) ImagePull(_ context.Context, _ string, _ client.ImagePullOptions) (client.ImagePullResponse, error) {
	f.pullCalls.Add(1)
	f.mu.Lock()
	body := f.pullBody
	f.mu.Unlock()
	return fakeImagePullResponse{ReadCloser: io.NopCloser(bytes.NewReader(body))}, nil
}

func (f *fakeDockerClient) ContainerCreate(_ context.Context, options client.ContainerCreateOptions) (client.ContainerCreateResult, error) {
	return client.ContainerCreateResult{ID: "container-id-for-" + options.Name + "-1234567890ab"}, nil
}

func (f *fakeDockerClient) ContainerStart(_ context.Context, _ string, _ client.ContainerStartOptions) (client.ContainerStartResult, error) {
	f.startCalls.Add(1)
	return client.ContainerStartResult{}, nil
}

func (f *fakeDockerClient) ContainerList(ctx context.Context, _ client.ContainerListOptions) (client.ContainerListResult, error) {
	f.mu.Lock()
	hook := f.onContainerList
	f.mu.Unlock()
	if hook != nil {
		if err := hook(ctx); err != nil {
			return client.ContainerListResult{}, err
		}
	}
	return client.ContainerListResult{}, nil
}

func (f *fakeDockerClient) ContainerStop(_ context.Context, _ string, _ client.ContainerStopOptions) (client.ContainerStopResult, error) {
	return client.ContainerStopResult{}, nil
}

func (f *fakeDockerClient) ContainerRemove(_ context.Context, _ string, _ client.ContainerRemoveOptions) (client.ContainerRemoveResult, error) {
	return client.ContainerRemoveResult{}, nil
}

func (f *fakeDockerClient) NetworkList(_ context.Context, opts client.NetworkListOptions) (client.NetworkListResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []network.Summary
	for _, n := range f.networksOnList {
		if matchesLabelFilter(filterValues(opts.Filters, "label"), n.Labels) {
			out = append(out, n)
		}
	}
	return client.NetworkListResult{Items: out}, nil
}

func (f *fakeDockerClient) NetworkCreate(_ context.Context, _ string, _ client.NetworkCreateOptions) (client.NetworkCreateResult, error) {
	return client.NetworkCreateResult{ID: "net-1234567890ab"}, nil
}

func (f *fakeDockerClient) NetworkConnect(_ context.Context, _ string, _ client.NetworkConnectOptions) (client.NetworkConnectResult, error) {
	return client.NetworkConnectResult{}, nil
}

func (f *fakeDockerClient) NetworkRemove(_ context.Context, _ string, _ client.NetworkRemoveOptions) (client.NetworkRemoveResult, error) {
	f.networkRemoveCalls.Add(1)
	return client.NetworkRemoveResult{}, nil
}

func (f *fakeDockerClient) VolumeCreate(_ context.Context, _ client.VolumeCreateOptions) (client.VolumeCreateResult, error) {
	f.volumeCreateCalls.Add(1)
	return client.VolumeCreateResult{Volume: volume.Volume{}}, nil
}

func (f *fakeDockerClient) VolumeList(_ context.Context, opts client.VolumeListOptions) (client.VolumeListResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []volume.Volume
	for _, v := range f.volumesOnList {
		if matchesLabelFilter(filterValues(opts.Filters, "label"), v.Labels) {
			out = append(out, v)
		}
	}
	return client.VolumeListResult{Items: out}, nil
}

// matchesLabelFilter mimics Docker's label filter: every requested "k=v"
// must exist in the resource's labels.
func matchesLabelFilter(wanted []string, labels map[string]string) bool {
	for _, w := range wanted {
		k, v, ok := strings.Cut(w, "=")
		if !ok {
			if _, present := labels[w]; !present {
				return false
			}
			continue
		}
		if labels[k] != v {
			return false
		}
	}
	return true
}

func filterValues(filters client.Filters, key string) []string {
	values, ok := filters[key]
	if !ok {
		return nil
	}
	out := make([]string, 0, len(values))
	for v := range values {
		out = append(out, v)
	}
	return out
}

func (f *fakeDockerClient) VolumeRemove(_ context.Context, _ string, _ client.VolumeRemoveOptions) (client.VolumeRemoveResult, error) {
	f.volumeRemoveCalls.Add(1)
	return client.VolumeRemoveResult{}, nil
}

func TestManager_Remove_DeletesProjectAndCleansNetworks(t *testing.T) {
	dir := t.TempDir()
	cli := &fakeDockerClient{
		networksOnList: []network.Summary{
			{Network: network.Network{ID: "net-1", Name: "myapp_default", Labels: map[string]string{"com.docker.compose.project": "myapp"}}},
		},
		volumesOnList: []volume.Volume{
			{Name: "myapp_data", Labels: map[string]string{"com.docker.compose.project": "myapp"}},
		},
	}
	mgr := NewManager(cli, dir)
	ctx := context.Background()

	require.NoError(t, mgr.Deploy(ctx, "myapp", []byte(testCompose), nil))
	require.DirExists(t, filepath.Join(dir, "myapp"))

	require.NoError(t, mgr.Remove(ctx, "myapp", false, nil))

	assert.NoDirExists(t, filepath.Join(dir, "myapp"))
	assert.Equal(t, int64(1), cli.networkRemoveCalls.Load(), "project-labeled network should be removed")
	assert.Equal(t, int64(0), cli.volumeRemoveCalls.Load(), "volumes must be preserved by default")

	_, err := mgr.ListVersions("myapp")
	assert.Error(t, err, "ListVersions on a removed project should error")
}

func TestManager_Remove_WithVolumes(t *testing.T) {
	dir := t.TempDir()
	cli := &fakeDockerClient{
		volumesOnList: []volume.Volume{
			{Name: "myapp_data", Labels: map[string]string{"com.docker.compose.project": "myapp"}},
			{Name: "other_project_data", Labels: map[string]string{"com.docker.compose.project": "other"}},
		},
	}
	mgr := NewManager(cli, dir)
	ctx := context.Background()

	require.NoError(t, mgr.Deploy(ctx, "myapp", []byte(testCompose), nil))
	require.NoError(t, mgr.Remove(ctx, "myapp", true, nil))

	assert.Equal(t, int64(1), cli.volumeRemoveCalls.Load(), "only myapp's volume should be removed")
}

func TestManager_Remove_NonexistentProject(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(&fakeDockerClient{}, dir)

	err := mgr.Remove(context.Background(), "ghost", false, nil)
	require.Error(t, err)
}

func TestManager_Deploy_CreatesNewProject(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(&fakeDockerClient{}, dir)

	err := mgr.Deploy(context.Background(), "myapp", []byte(testCompose), nil)
	require.NoError(t, err)

	versions, err := mgr.ListVersions("myapp")
	require.NoError(t, err)
	require.Len(t, versions, 1)
	assert.Equal(t, "Initial deployment", versions[0].Message)

	assert.FileExists(t, filepath.Join(dir, "myapp", composeFilename))
}

func TestManager_Deploy_UpdatesExistingProject(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(&fakeDockerClient{}, dir)
	ctx := context.Background()

	require.NoError(t, mgr.Deploy(ctx, "myapp", []byte(testCompose), nil))
	require.NoError(t, mgr.Deploy(ctx, "myapp", []byte(testComposeUpdated), nil))

	versions, err := mgr.ListVersions("myapp")
	require.NoError(t, err)
	require.Len(t, versions, 2)
	assert.Equal(t, "Update config", versions[0].Message)
	assert.Equal(t, "Initial deployment", versions[1].Message)
}

func TestManager_Deploy_PropagatesUpdateError(t *testing.T) {
	dir := t.TempDir()
	wantErr := errors.New("docker daemon is unreachable")
	cli := &fakeDockerClient{onContainerList: func(context.Context) error { return wantErr }}
	mgr := NewManager(cli, dir)

	// Pre-create the project so Deploy takes the update branch.
	cli.mu.Lock()
	cli.onContainerList = nil
	cli.mu.Unlock()
	require.NoError(t, mgr.Deploy(context.Background(), "myapp", []byte(testCompose), nil))

	cli.mu.Lock()
	cli.onContainerList = func(context.Context) error { return wantErr }
	cli.mu.Unlock()

	err := mgr.Deploy(context.Background(), "myapp", []byte(testComposeUpdated), nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, wantErr, "update error must propagate verbatim, not be masked by a fallback create")
}

func TestManager_Deploy_SerializesSameProject(t *testing.T) {
	dir := t.TempDir()

	gate := make(chan struct{})
	var active, maxActive atomic.Int32

	cli := &fakeDockerClient{}
	cli.onContainerList = func(ctx context.Context) error {
		n := active.Add(1)
		defer active.Add(-1)
		for {
			if m := maxActive.Load(); n > m {
				if maxActive.CompareAndSwap(m, n) {
					break
				}
				continue
			}
			break
		}
		select {
		case <-gate:
		case <-ctx.Done():
			return ctx.Err()
		}
		return nil
	}

	mgr := NewManager(cli, dir)

	var wg sync.WaitGroup
	wg.Add(2)
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() {
			defer wg.Done()
			errs <- mgr.Deploy(context.Background(), "shared", []byte(testCompose), nil)
		}()
	}

	// Let the first goroutine enter the docker call, then confirm the second is blocked.
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, int32(1), active.Load(), "second Deploy on same project must wait for the lock")

	close(gate)
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	assert.Equal(t, int32(1), maxActive.Load(), "same-project Deploys must not overlap")
}

func TestManager_Deploy_ParallelizesDifferentProjects(t *testing.T) {
	dir := t.TempDir()

	gate := make(chan struct{})
	reached := make(chan struct{}, 2)

	cli := &fakeDockerClient{}
	cli.onContainerList = func(ctx context.Context) error {
		reached <- struct{}{}
		select {
		case <-gate:
		case <-ctx.Done():
			return ctx.Err()
		}
		return nil
	}

	mgr := NewManager(cli, dir)

	var wg sync.WaitGroup
	wg.Add(2)
	errs := make(chan error, 2)
	go func() {
		defer wg.Done()
		errs <- mgr.Deploy(context.Background(), "alpha", []byte(testCompose), nil)
	}()
	go func() {
		defer wg.Done()
		errs <- mgr.Deploy(context.Background(), "beta", []byte(testCompose), nil)
	}()

	// Both must reach the docker call before either is released.
	for i := 0; i < 2; i++ {
		select {
		case <-reached:
		case <-time.After(2 * time.Second):
			t.Fatal("different-project Deploys serialized; expected parallelism")
		}
	}

	close(gate)
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
}

func TestManager_RejectsUnsafeProjectNames(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(&fakeDockerClient{}, dir)
	ctx := context.Background()

	cases := []string{"../etc", "foo/bar", "..", "", "Foo", "foo bar"}
	for _, name := range cases {
		err := mgr.Deploy(ctx, name, []byte(testCompose), nil)
		assert.Error(t, err, "expected Deploy to reject unsafe project name %q", name)
	}
}

func TestManager_PullError_Surfaces(t *testing.T) {
	dir := t.TempDir()
	cli := &fakeDockerClient{
		// Simulate Docker pull stream emitting an error JSON object.
		pullBody: []byte(`{"errorDetail":{"message":"pull access denied"},"error":"pull access denied"}`),
	}
	mgr := NewManager(cli, dir)
	err := mgr.Deploy(context.Background(), "myapp", []byte(testCompose), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pull access denied")
}

func TestManager_CreateVolumes_Idempotent(t *testing.T) {
	const compose = `
services:
  web:
    image: nginx
    volumes:
      - data:/data
volumes:
  data:
`
	dir := t.TempDir()
	cli := &fakeDockerClient{}
	mgr := NewManager(cli, dir)
	ctx := context.Background()

	// First deploy creates the volume.
	require.NoError(t, mgr.Deploy(ctx, "myapp", []byte(compose), nil))
	firstCreates := cli.volumeCreateCalls.Load()

	// Pretend the volume now exists so the second deploy must skip it.
	cli.mu.Lock()
	cli.volumesOnList = []volume.Volume{{Name: "myapp_data", Labels: map[string]string{"com.docker.compose.project": "myapp"}}}
	cli.mu.Unlock()

	require.NoError(t, mgr.Deploy(ctx, "myapp", []byte(compose), nil))
	assert.Equal(t, firstCreates, cli.volumeCreateCalls.Load(), "existing volume must not be re-created")
}

func TestManager_RollbackVersion_SerializesWithDeploy(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(&fakeDockerClient{}, dir)
	ctx := context.Background()

	require.NoError(t, mgr.Deploy(ctx, "myapp", []byte(testCompose), nil))
	require.NoError(t, mgr.Deploy(ctx, "myapp", []byte(testComposeUpdated), nil))

	versions, err := mgr.ListVersions("myapp")
	require.NoError(t, err)
	require.Len(t, versions, 2)
	firstHash := versions[1].Hash

	// Kick off concurrent rollback + deploy. They must serialize — no git corruption, both succeed.
	var wg sync.WaitGroup
	wg.Add(2)
	errs := make(chan error, 2)
	go func() {
		defer wg.Done()
		errs <- mgr.RollbackVersion(ctx, "myapp", firstHash, nil)
	}()
	go func() {
		defer wg.Done()
		errs <- mgr.Deploy(ctx, "myapp", []byte(testComposeV3), nil)
	}()
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}

	versions, err = mgr.ListVersions("myapp")
	require.NoError(t, err)
	assert.Len(t, versions, 4, "two concurrent writes should each land one commit")
}

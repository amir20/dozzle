package docker

import (
	"context"
	"testing"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/container/swap"
	docker "github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/image"
	"github.com/moby/moby/api/types/system"
	"github.com/moby/moby/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func (m *mockedProxy) ImageInspect(ctx context.Context, ref string, _ ...client.ImageInspectOption) (client.ImageInspectResult, error) {
	args := m.Called(ctx, ref)
	return client.ImageInspectResult{InspectResponse: args.Get(0).(image.InspectResponse)}, args.Error(1)
}

func TestFindContainerFillsImageIdentity(t *testing.T) {
	proxy := new(mockedProxy)
	proxy.On("ContainerInspect", mock.Anything, "abcdefghijkl").Return(docker.InspectResponse{
		ID:         "abcdefghijklmnopqrst",
		Image:      oldImageID,
		State:      &docker.State{Status: "running"},
		HostConfig: &docker.HostConfig{},
		Config:     &docker.Config{Image: "nginx:latest"},
	}, nil)
	proxy.On("ImageInspect", mock.Anything, oldImageID).Return(image.InspectResponse{
		ID:          oldImageID,
		RepoDigests: []string{"ghcr.io/other/nginx@sha256:aaa", "nginx@sha256:bbb"},
	}, nil).Once()
	cli := &Client{cli: proxy, host: container.Host{ID: "localhost"}, info: system.Info{}}

	for range 2 {
		c, err := cli.FindContainer(context.Background(), "abcdefghijkl")
		require.NoError(t, err)
		assert.Equal(t, oldImageID, c.ImageID)
		assert.Equal(t, "nginx@sha256:bbb", c.ImageDigest, "the digest of the repository the container was created from")
	}
	proxy.AssertExpectations(t) // the second inspect read the cache
}

func TestFindContainerLocalImageHasNoDigest(t *testing.T) {
	proxy := new(mockedProxy)
	proxy.On("ContainerInspect", mock.Anything, "abcdefghijkl").Return(docker.InspectResponse{
		ID:         "abcdefghijklmnopqrst",
		Image:      oldImageID,
		State:      &docker.State{Status: "running"},
		HostConfig: &docker.HostConfig{},
		Config:     &docker.Config{Image: "myapp"},
	}, nil)
	proxy.On("ImageInspect", mock.Anything, oldImageID).Return(image.InspectResponse{ID: oldImageID}, nil)
	cli := &Client{cli: proxy, host: container.Host{ID: "localhost"}, info: system.Info{}}

	c, err := cli.FindContainer(context.Background(), "abcdefghijkl")
	require.NoError(t, err)
	assert.Equal(t, oldImageID, c.ImageID)
	assert.Empty(t, c.ImageDigest)
}

func TestListedContainerHasImageID(t *testing.T) {
	c := newContainer(docker.Summary{ID: "abcdefghijklmnop", Image: "nginx:latest", ImageID: oldImageID}, "localhost")
	assert.Equal(t, oldImageID, c.ImageID)
}

func TestRepoDigest(t *testing.T) {
	digests := []string{"ghcr.io/other/nginx@sha256:aaa", "nginx@sha256:bbb"}
	assert.Equal(t, "nginx@sha256:bbb", RepoDigest(digests, "nginx:latest"))
	assert.Equal(t, "nginx@sha256:bbb", RepoDigest(digests, "docker.io/library/nginx:1.27"))
	assert.Equal(t, "ghcr.io/other/nginx@sha256:aaa", RepoDigest(digests, "ghcr.io/other/nginx"))
	assert.Equal(t, "ghcr.io/other/nginx@sha256:aaa", RepoDigest(digests, "quay.io/x/y"), "no match falls back to the first")
	assert.Empty(t, RepoDigest(nil, "nginx"))
}

// The tracker only knows the label by its value.
func TestRestoredRefLabelMatchesSwap(t *testing.T) {
	assert.Equal(t, "dev.dozzle.self-update.image", swap.ImageRefLabel)
}

func TestUpdateContainerStampsSourceAndRun(t *testing.T) {
	cli := newUpdateClient(t, appInspect(nil))
	run := runUpdate(cli, container.UpdateOptions{Source: container.UpdateSourceSchedule, RunID: "run-1"})
	require.NoError(t, run.err)

	labels := cli.engine.Created[0].Config.Labels
	assert.Equal(t, container.UpdateSourceSchedule, labels[container.UpdateSourceLabel])
	assert.Equal(t, "run-1", labels[container.UpdateRunLabel])
}

// An update outside a run is Dozzle's by default, and drops the run the old
// container was updated in.
func TestUpdateContainerClearsStaleRun(t *testing.T) {
	cli := newUpdateClient(t, appInspect(map[string]string{
		container.UpdateSourceLabel: container.UpdateSourceSchedule,
		container.UpdateRunLabel:    "run-0",
	}))
	run := runUpdate(cli, container.UpdateOptions{})
	require.NoError(t, run.err)

	labels := cli.engine.Created[0].Config.Labels
	assert.Equal(t, container.UpdateSourceDozzle, labels[container.UpdateSourceLabel])
	assert.NotContains(t, labels, container.UpdateRunLabel)
}

// idleClient is an engine with no containers and no events, enough to run a
// store whose update events a test reads.
type idleClient struct {
	container.Client
}

func (idleClient) Host() container.Host { return container.Host{ID: "host1"} }
func (idleClient) ListContainers(context.Context, container.ContainerLabels) ([]container.Container, error) {
	return nil, nil
}
func (idleClient) ContainerEvents(ctx context.Context, _ chan<- container.ContainerEvent) error {
	<-ctx.Done()
	return ctx.Err()
}

type idleStats struct{}

func (idleStats) Start(context.Context) bool                                { return false }
func (idleStats) Subscribe(context.Context, chan<- container.ContainerStat) {}
func (idleStats) Stop()                                                     {}

func TestUpdateContainerRecordsRollback(t *testing.T) {
	old := appInspect(nil)
	old.State.StartedAt = "2026-10-03T03:00:00Z"
	cli := newUpdateClient(t, old)
	cli.engine.StartErrFor = "new1"
	cli.images[newImageID] = image.InspectResponse{ID: newImageID, RepoDigests: []string{"nginx@sha256:ccc"}}

	store := container.NewStore(t.Context(), idleClient{}, idleStats{}, container.ContainerLabels{})
	svc := &Service{client: cli, store: store}
	ch := make(chan container.UpdateProgress, 100)
	_, err := svc.UpdateContainer(context.Background(), container.Container{ID: appID, Name: "app", Host: "host1"}, container.UpdateOptions{Source: container.UpdateSourceCloud, RunID: "run-2"}, ch)
	require.ErrorContains(t, err, "rolled back")

	events := svc.RecentUpdates()
	require.Len(t, events, 1)
	event := events[0]
	assert.True(t, event.RolledBack)
	assert.Equal(t, "host1", event.Host)
	assert.Equal(t, "app", event.Name)
	assert.Equal(t, appID[:12], event.OldID)
	assert.Equal(t, appID[:12], event.NewID, "the old container runs under the name again")
	assert.Equal(t, oldImageID, event.FromImageID)
	assert.Equal(t, newImageID, event.ToImageID)
	assert.Equal(t, "nginx@sha256:bbb", event.FromDigest)
	assert.Equal(t, "nginx@sha256:ccc", event.ToDigest)
	assert.Equal(t, "nginx:latest", event.FromRef)
	assert.Equal(t, "2026-10-03T03:00:00Z", event.OldStartedAt.Format("2006-01-02T15:04:05Z07:00"))
	assert.Equal(t, container.UpdateSourceCloud, event.Source)
	assert.Equal(t, "run-2", event.RunID)
	assert.False(t, event.At.IsZero())
}

func TestUpdateContainerCommitRecordsNoRollback(t *testing.T) {
	cli := newUpdateClient(t, appInspect(nil))
	store := container.NewStore(t.Context(), idleClient{}, idleStats{}, container.ContainerLabels{})
	svc := &Service{client: cli, store: store}
	ch := make(chan container.UpdateProgress, 100)
	_, err := svc.UpdateContainer(context.Background(), container.Container{ID: appID, Name: "app"}, container.UpdateOptions{}, ch)
	require.NoError(t, err)
	assert.Empty(t, svc.RecentUpdates(), "a committed update is recorded from its start, not by the swap")
}

package docker

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"time"

	"github.com/amir20/dozzle/internal/container"

	"testing"

	"net/netip"

	docker "github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/mount"
	"github.com/moby/moby/api/types/system"
	"github.com/moby/moby/api/types/volume"
	"github.com/moby/moby/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockedProxy struct {
	mock.Mock
	CLI
}

func (m *mockedProxy) ContainerList(context.Context, client.ContainerListOptions) (client.ContainerListResult, error) {
	args := m.Called()
	containers, ok := args.Get(0).([]docker.Summary)
	if !ok && args.Get(0) != nil {
		panic("containers is not of type []docker.Summary")
	}
	return client.ContainerListResult{Items: containers}, args.Error(1)
}

func (m *mockedProxy) DiskUsage(context.Context, client.DiskUsageOptions) (client.DiskUsageResult, error) {
	args := m.Called()
	return args.Get(0).(client.DiskUsageResult), args.Error(1)
}

func (m *mockedProxy) ContainerLogs(ctx context.Context, id string, options client.ContainerLogsOptions) (client.ContainerLogsResult, error) {
	args := m.Called(ctx, id, options)
	reader, ok := args.Get(0).(io.ReadCloser)
	if !ok && args.Get(0) != nil {
		panic("reader is not of type io.ReadCloser")
	}
	return reader, args.Error(1)
}

func (m *mockedProxy) ContainerInspect(ctx context.Context, containerID string, _ client.ContainerInspectOptions) (client.ContainerInspectResult, error) {
	args := m.Called(ctx, containerID)
	return client.ContainerInspectResult{Container: args.Get(0).(docker.InspectResponse)}, args.Error(1)
}

func (m *mockedProxy) ContainerStats(ctx context.Context, containerID string, options client.ContainerStatsOptions) (client.ContainerStatsResult, error) {
	_ = ctx
	_ = containerID
	_ = options
	return client.ContainerStatsResult{}, nil
}

func (m *mockedProxy) ContainerStart(ctx context.Context, containerID string, options client.ContainerStartOptions) (client.ContainerStartResult, error) {

	args := m.Called(ctx, containerID, options)
	return client.ContainerStartResult{}, args.Error(0)
}

func (m *mockedProxy) ContainerStop(ctx context.Context, containerID string, options client.ContainerStopOptions) (client.ContainerStopResult, error) {
	args := m.Called(ctx, containerID, options)
	return client.ContainerStopResult{}, args.Error(0)
}

func (m *mockedProxy) ContainerRestart(ctx context.Context, containerID string, options client.ContainerRestartOptions) (client.ContainerRestartResult, error) {

	args := m.Called(ctx, containerID, options)
	return client.ContainerRestartResult{}, args.Error(0)
}

func Test_dockerClient_ListContainers_null(t *testing.T) {
	proxy := new(mockedProxy)
	proxy.On("ContainerList", mock.Anything, mock.Anything).Return(nil, nil)
	client := &Client{cli: proxy, host: container.Host{ID: "localhost"}, info: system.Info{}}

	list, err := client.ListContainers(context.Background(), container.ContainerLabels{})
	assert.Empty(t, list, "list should be empty")
	require.NoError(t, err, "error should not return an error.")

	proxy.AssertExpectations(t)
}

func Test_dockerClient_ListContainers_error(t *testing.T) {
	proxy := new(mockedProxy)
	proxy.On("ContainerList", mock.Anything, mock.Anything).Return(nil, errors.New("test"))
	client := &Client{cli: proxy, host: container.Host{ID: "localhost"}, info: system.Info{}}

	list, err := client.ListContainers(context.Background(), container.ContainerLabels{})
	assert.Nil(t, list, "list should be nil")
	require.Error(t, err, "test.")

	proxy.AssertExpectations(t)
}

func Test_dockerClient_ListContainers_happy(t *testing.T) {
	containers := []docker.Summary{
		{
			ID:    "abcdefghijklmnopqrst",
			Names: []string{"/z_test_container"},
		},
		{
			ID:    "1234567890_abcxyzdef",
			Names: []string{"/a_test_container"},
		},
	}

	proxy := new(mockedProxy)
	proxy.On("ContainerList", mock.Anything, mock.Anything).Return(containers, nil)
	client := &Client{cli: proxy, host: container.Host{ID: "localhost"}, info: system.Info{}}

	list, err := client.ListContainers(context.Background(), container.ContainerLabels{})
	require.NoError(t, err, "error should not return an error.")

	Ids := []string{"1234567890_a", "abcdefghijkl"}
	for i, container := range list {
		assert.Equal(t, container.ID, Ids[i])
	}

	proxy.AssertExpectations(t)
}

func Test_dockerClient_ContainerLogs_happy(t *testing.T) {
	id := "123456"

	proxy := new(mockedProxy)
	expected := "INFO Testing logs..."
	b := make([]byte, 8)

	binary.BigEndian.PutUint32(b[4:], uint32(len(expected)))
	b = append(b, []byte(expected)...)

	reader := io.NopCloser(bytes.NewReader(b))
	since := time.Date(2021, time.January, 1, 0, 0, 0, 0, time.UTC)
	options := client.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Tail:       "100",
		Timestamps: true,
		Since:      "2020-12-31T23:59:59.95Z"}
	proxy.On("ContainerLogs", mock.Anything, id, options).Return(reader, nil)

	client := &Client{cli: proxy, host: container.Host{ID: "localhost"}, info: system.Info{}}
	logReader, _ := client.ContainerLogs(context.Background(), id, since, container.STDALL)

	actual, _ := io.ReadAll(logReader)
	assert.Equal(t, string(b), string(actual), "message doesn't match expected")
	proxy.AssertExpectations(t)
}

func Test_dockerClient_ContainerLogs_error(t *testing.T) {
	id := "123456"
	proxy := new(mockedProxy)

	proxy.On("ContainerLogs", mock.Anything, id, mock.Anything).Return(nil, errors.New("test"))

	client := &Client{cli: proxy, host: container.Host{ID: "localhost"}, info: system.Info{}}

	reader, err := client.ContainerLogs(context.Background(), id, time.Time{}, container.STDALL)

	assert.Nil(t, reader, "reader should be nil")
	assert.Error(t, err, "error should have been returned")
	proxy.AssertExpectations(t)
}

func Test_dockerClient_FindContainer_happy(t *testing.T) {
	proxy := new(mockedProxy)

	state := &docker.State{Status: "running", StartedAt: time.Now().Format(time.RFC3339Nano)}

	json := docker.InspectResponse{
		ID:         "abcdefghijklmnopqrst",
		State:      state,
		HostConfig: &docker.HostConfig{},
		Config:     &docker.Config{Tty: false},
	}
	proxy.On("ContainerInspect", mock.Anything, "abcdefghijkl").Return(json, nil)

	client := &Client{cli: proxy, host: container.Host{ID: "localhost"}, info: system.Info{}}

	container, err := client.FindContainer(context.Background(), "abcdefghijkl")
	require.NoError(t, err, "error should not be thrown")

	assert.Equal(t, container.ID, "abcdefghijkl")

	proxy.AssertExpectations(t)
}

func Test_dockerClient_FindContainer_error(t *testing.T) {
	proxy := new(mockedProxy)
	proxy.On("ContainerInspect", mock.Anything, "not_valid").Return(docker.InspectResponse{}, errors.New("not found"))
	client := &Client{cli: proxy, host: container.Host{ID: "localhost"}, info: system.Info{}}

	_, err := client.FindContainer(context.Background(), "not_valid")
	require.Error(t, err, "error should be thrown")

	proxy.AssertExpectations(t)
}

func Test_dockerClient_ContainerSizes(t *testing.T) {
	proxy := new(mockedProxy)
	proxy.On("ContainerList", mock.Anything, mock.Anything).Return([]docker.Summary{
		{ID: "abcdefghijklmnopqrst", SizeRw: 2048},
		{ID: "1234567890_abcxyzdef"},
	}, nil)
	client := &Client{cli: proxy, host: container.Host{ID: "localhost"}}

	sizes, err := client.ContainerSizes(context.Background())
	require.NoError(t, err)
	assert.Equal(t, map[string]int64{"abcdefghijkl": 2048, "1234567890_a": 0}, sizes, "keyed by the short ID the store uses")
}

func Test_dockerClient_DiskUsage(t *testing.T) {
	proxy := new(mockedProxy)
	proxy.On("DiskUsage").Return(client.DiskUsageResult{
		Images:     client.ImagesDiskUsage{TotalCount: 30, ActiveCount: 12, Reclaimable: 4 << 30},
		BuildCache: client.BuildCacheDiskUsage{Reclaimable: 7},
		Volumes: client.VolumesDiskUsage{Items: []volume.Volume{
			{Name: "clickhouse_data", UsageData: &volume.UsageData{Size: 23 << 30, RefCount: 1}},
			{Name: "shared", UsageData: &volume.UsageData{Size: 10, RefCount: 2}},
			{Name: "plugin", UsageData: &volume.UsageData{Size: -1, RefCount: 1}},
			{Name: "orphan", UsageData: &volume.UsageData{Size: 99, RefCount: 0}},
			{Name: "orphan-empty", UsageData: &volume.UsageData{Size: 0, RefCount: 0}},
			{Name: "orphan-plugin", UsageData: &volume.UsageData{Size: -1, RefCount: 0}},
		}},
	}, nil)
	proxy.On("ContainerList", mock.Anything, mock.Anything).Return([]docker.Summary{
		{ID: "abcdefghijklmnopqrst", State: "exited", Mounts: []docker.MountPoint{
			{Type: mount.TypeVolume, Name: "clickhouse_data", Destination: "/var/lib/clickhouse"},
			{Type: mount.TypeVolume, Name: "shared", Destination: "/shared"},
			{Type: mount.TypeVolume, Name: "plugin", Destination: "/remote"},
			{Type: mount.TypeBind, Source: "/data/pg", Destination: "/var/lib/postgresql/data"},
		}},
		{ID: "1234567890_abcxyzdef", Mounts: []docker.MountPoint{
			{Type: mount.TypeVolume, Name: "shared", Destination: "/shared"},
		}},
	}, nil)
	client := &Client{cli: proxy, host: container.Host{ID: "localhost"}}

	usage, err := client.DiskUsage(context.Background())
	require.NoError(t, err)
	assert.Equal(t, map[string][]container.VolumeUsage{
		"abcdefghijkl": {
			{Name: "clickhouse_data", Destination: "/var/lib/clickhouse", Size: 23 << 30, Links: 1},
			{Name: "shared", Destination: "/shared", Size: 10, Links: 2},
		},
		"1234567890_a": {{Name: "shared", Destination: "/shared", Size: 10, Links: 2}},
	}, usage.Volumes, "bind mounts and volumes the driver cannot size are left out")
	assert.Equal(t, container.Reclaimable{
		Images: 18, ImagesSize: 4 << 30,
		// a volume the driver cannot size is neither counted nor sized
		Volumes: 2, VolumesSize: 99,
		BuildCacheSize: 7,
	}, usage.Reclaimable, "stopped containers are left to the store")
}

func Test_dockerClient_ContainerSize(t *testing.T) {
	size := int64(4096)
	proxy := new(mockedProxy)
	proxy.On("ContainerInspect", mock.Anything, "measured").Return(docker.InspectResponse{SizeRw: &size}, nil)
	proxy.On("ContainerInspect", mock.Anything, "unmeasured").Return(docker.InspectResponse{}, nil)
	client := &Client{cli: proxy, host: container.Host{ID: "localhost"}}

	got, err := client.ContainerSize(context.Background(), "measured")
	require.NoError(t, err)
	assert.Equal(t, int64(4096), got)

	_, err = client.ContainerSize(context.Background(), "unmeasured")
	assert.Error(t, err, "a missing size is not 0 bytes")
}

func Test_dockerClient_ContainerActions_happy(t *testing.T) {
	proxy := new(mockedProxy)
	client := &Client{cli: proxy, host: container.Host{ID: "localhost"}, info: system.Info{}}

	state := &docker.State{Status: "running", StartedAt: time.Now().Format(time.RFC3339Nano)}
	json := docker.InspectResponse{ID: "abcdefghijkl", State: state, HostConfig: &docker.HostConfig{}, Config: &docker.Config{Tty: false}}

	proxy.On("ContainerInspect", mock.Anything, "abcdefghijkl").Return(json, nil)
	proxy.On("ContainerStart", mock.Anything, "abcdefghijkl", mock.Anything).Return(nil)
	proxy.On("ContainerStop", mock.Anything, "abcdefghijkl", mock.Anything).Return(nil)
	proxy.On("ContainerRestart", mock.Anything, "abcdefghijkl", mock.Anything).Return(nil)

	c, err := client.FindContainer(context.Background(), "abcdefghijkl")
	require.NoError(t, err, "error should not be thrown")

	assert.Equal(t, c.ID, "abcdefghijkl")

	actions := []string{"start", "stop", "restart"}
	for _, action := range actions {
		err := client.ContainerActions(context.Background(), container.ContainerAction(action), c.ID)
		require.NoError(t, err, "error should not be thrown")
		assert.Equal(t, err, nil)
	}

	proxy.AssertExpectations(t)
}

func Test_dockerClient_ContainerActions_error(t *testing.T) {

	proxy := new(mockedProxy)
	client := &Client{cli: proxy, host: container.Host{ID: "localhost"}, info: system.Info{}}
	proxy.On("ContainerInspect", mock.Anything, "random-id").Return(docker.InspectResponse{}, errors.New("not found"))
	proxy.On("ContainerStart", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("test"))
	proxy.On("ContainerStop", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("test"))
	proxy.On("ContainerRestart", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("test"))

	c, err := client.FindContainer(context.Background(), "random-id")
	require.Error(t, err, "error should be thrown")

	actions := []string{"start", "stop", "restart"}
	for _, action := range actions {
		err := client.ContainerActions(context.Background(), container.ContainerAction(action), c.ID)
		require.Error(t, err, "error should be thrown")
		assert.Error(t, err, "error should have been returned")
	}

	proxy.AssertExpectations(t)
}

func Test_newContainer_labelPriority(t *testing.T) {
	tests := []struct {
		name          string
		labels        map[string]string
		containerName string
		expectedName  string
		expectedGroup string
	}{
		{
			name:          "dozzle labels take priority",
			labels:        map[string]string{"dev.dozzle.name": "dozzle-name", "coolify.serviceName": "coolify-name", "dev.dozzle.group": "dozzle-group", "coolify.projectName": "coolify-project"},
			containerName: "/docker-name",
			expectedName:  "dozzle-name",
			expectedGroup: "dozzle-group",
		},
		{
			name:          "coolify labels as fallback",
			labels:        map[string]string{"coolify.serviceName": "coolify-name", "coolify.projectName": "coolify-project"},
			containerName: "/docker-name",
			expectedName:  "coolify-name",
			expectedGroup: "coolify-project",
		},
		{
			name:          "docker name as final fallback",
			labels:        map[string]string{},
			containerName: "/docker-name",
			expectedName:  "docker-name",
			expectedGroup: "",
		},
		{
			name:          "coolify name with dozzle group",
			labels:        map[string]string{"coolify.serviceName": "coolify-name", "dev.dozzle.group": "dozzle-group"},
			containerName: "/docker-name",
			expectedName:  "coolify-name",
			expectedGroup: "dozzle-group",
		},
		{
			name:          "coolify preview includes PR id",
			labels:        map[string]string{"coolify.serviceName": "coolify-name", "coolify.pullRequestId": "1241", "coolify.projectName": "coolify-project"},
			containerName: "/docker-name",
			expectedName:  "PR 1241 · coolify-name",
			expectedGroup: "coolify-project",
		},
		{
			name:          "coolify pullRequestId 0 is not a preview",
			labels:        map[string]string{"coolify.serviceName": "coolify-name", "coolify.pullRequestId": "0", "coolify.projectName": "coolify-project"},
			containerName: "/docker-name",
			expectedName:  "coolify-name",
			expectedGroup: "coolify-project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := docker.Summary{
				ID:     "abcdefghijklmnopqrst",
				Names:  []string{tt.containerName},
				Labels: tt.labels,
			}
			c := newContainer(summary, "localhost")
			assert.Equal(t, tt.expectedName, c.Name)
			assert.Equal(t, tt.expectedGroup, c.Group)
		})
	}
}

func Test_newContainer_ports(t *testing.T) {
	summary := docker.Summary{
		ID:    "abcdefghijklmnopqrst",
		Names: []string{"/app"},
		Ports: []docker.PortSummary{
			{IP: netip.MustParseAddr("0.0.0.0"), PublicPort: 8080, PrivatePort: 80, Type: "tcp"},
			{IP: netip.MustParseAddr("::"), PublicPort: 8080, PrivatePort: 80, Type: "tcp"},
			{PrivatePort: 9000, Type: "tcp"},
		},
	}

	c := newContainer(summary, "localhost")
	assert.Equal(t, []string{"0.0.0.0:8080->80/tcp", ":::8080->80/tcp"}, c.Ports)
}

func Test_newContainerFromJSON_labelPriority(t *testing.T) {
	tests := []struct {
		name          string
		labels        map[string]string
		containerName string
		expectedName  string
		expectedGroup string
	}{
		{
			name:          "dozzle labels take priority",
			labels:        map[string]string{"dev.dozzle.name": "dozzle-name", "coolify.serviceName": "coolify-name", "dev.dozzle.group": "dozzle-group", "coolify.projectName": "coolify-project"},
			containerName: "/docker-name",
			expectedName:  "dozzle-name",
			expectedGroup: "dozzle-group",
		},
		{
			name:          "coolify labels as fallback",
			labels:        map[string]string{"coolify.serviceName": "coolify-name", "coolify.projectName": "coolify-project"},
			containerName: "/docker-name",
			expectedName:  "coolify-name",
			expectedGroup: "coolify-project",
		},
		{
			name:          "docker name as final fallback",
			labels:        map[string]string{},
			containerName: "/docker-name",
			expectedName:  "docker-name",
			expectedGroup: "",
		},
		{
			name:          "coolify preview includes PR id",
			labels:        map[string]string{"coolify.serviceName": "coolify-name", "coolify.pullRequestId": "1241", "coolify.projectName": "coolify-project"},
			containerName: "/docker-name",
			expectedName:  "PR 1241 · coolify-name",
			expectedGroup: "coolify-project",
		},
		{
			name:          "coolify pullRequestId 0 is not a preview",
			labels:        map[string]string{"coolify.serviceName": "coolify-name", "coolify.pullRequestId": "0", "coolify.projectName": "coolify-project"},
			containerName: "/docker-name",
			expectedName:  "coolify-name",
			expectedGroup: "coolify-project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := &docker.State{Status: "running", StartedAt: time.Now().Format(time.RFC3339Nano)}
			json := docker.InspectResponse{
				ID:         "abcdefghijklmnopqrst",
				Name:       tt.containerName,
				State:      state,
				HostConfig: &docker.HostConfig{},
				Config:     &docker.Config{Labels: tt.labels},
			}
			c := newContainerFromJSON(json, "localhost")
			assert.Equal(t, tt.expectedName, c.Name)
			assert.Equal(t, tt.expectedGroup, c.Group)
		})
	}
}

func Test_newContainerFromJSON_healthOnlyWhileRunning(t *testing.T) {
	inspect := func(state *docker.State) docker.InspectResponse {
		return docker.InspectResponse{
			ID:         "abcdefghijklmnopqrst",
			Name:       "/health",
			State:      state,
			HostConfig: &docker.HostConfig{},
			Config:     &docker.Config{},
		}
	}

	running := newContainerFromJSON(inspect(&docker.State{
		Status:  "running",
		Running: true,
		Health:  &docker.Health{Status: "healthy"},
	}), "localhost")
	assert.Equal(t, "healthy", running.Health)

	// Docker reports the last check of a stopped container, flipped to unhealthy by the kill.
	exited := newContainerFromJSON(inspect(&docker.State{
		Status: "exited",
		Health: &docker.Health{Status: "unhealthy"},
	}), "localhost")
	assert.Empty(t, exited.Health)
}

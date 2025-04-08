package docker

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"time"

	"testing"

	"github.com/amir20/dozzle/internal/container"
	docker "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/system"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockedProxy struct {
	mock.Mock
	DockerCLI
}

func (m *mockedProxy) ContainerList(context.Context, docker.ListOptions) ([]docker.Summary, error) {
	args := m.Called()
	containers, ok := args.Get(0).([]docker.Summary)
	if !ok && args.Get(0) != nil {
		panic("containers is not of type []docker.Summary")
	}
	return containers, args.Error(1)
}

func (m *mockedProxy) ContainerLogs(ctx context.Context, id string, options docker.LogsOptions) (io.ReadCloser, error) {
	args := m.Called(ctx, id, options)
	reader, ok := args.Get(0).(io.ReadCloser)
	if !ok && args.Get(0) != nil {
		panic("reader is not of type io.ReadCloser")
	}
	return reader, args.Error(1)
}

func (m *mockedProxy) ContainerInspect(ctx context.Context, containerID string) (docker.InspectResponse, error) {
	args := m.Called(ctx, containerID)
	return args.Get(0).(docker.InspectResponse), args.Error(1)
}

func (m *mockedProxy) ContainerStats(ctx context.Context, containerID string, stream bool) (docker.StatsResponseReader, error) {
	return docker.StatsResponseReader{}, nil
}

func (m *mockedProxy) ContainerStart(ctx context.Context, containerID string, options docker.StartOptions) error {

	args := m.Called(ctx, containerID, options)
	err := args.Get(0)

	if err != nil {
		return args.Error(0)
	}

	return nil
}

func (m *mockedProxy) ContainerStop(ctx context.Context, containerID string, options docker.StopOptions) error {
	args := m.Called(ctx, containerID, options)
	err := args.Get(0)

	if err != nil {
		return args.Error(0)
	}

	return nil
}

func (m *mockedProxy) ContainerRestart(ctx context.Context, containerID string, options docker.StopOptions) error {

	args := m.Called(ctx, containerID, options)
	err := args.Get(0)

	if err != nil {
		return args.Error(0)
	}

	return nil
}

func Test_dockerClient_ListContainers_null(t *testing.T) {
	proxy := new(mockedProxy)
	proxy.On("ContainerList", mock.Anything, mock.Anything).Return(nil, nil)
	client := &DockerClient{proxy, container.Host{ID: "localhost"}, system.Info{}}

	list, err := client.ListContainers(context.Background(), container.ContainerLabels{})
	assert.Empty(t, list, "list should be empty")
	require.NoError(t, err, "error should not return an error.")

	proxy.AssertExpectations(t)
}

func Test_dockerClient_ListContainers_error(t *testing.T) {
	proxy := new(mockedProxy)
	proxy.On("ContainerList", mock.Anything, mock.Anything).Return(nil, errors.New("test"))
	client := &DockerClient{proxy, container.Host{ID: "localhost"}, system.Info{}}

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
	client := &DockerClient{proxy, container.Host{ID: "localhost"}, system.Info{}}

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
	options := docker.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Tail:       "100",
		Timestamps: true,
		Since:      "2020-12-31T23:59:59.95Z"}
	proxy.On("ContainerLogs", mock.Anything, id, options).Return(reader, nil)

	client := &DockerClient{proxy, container.Host{ID: "localhost"}, system.Info{}}
	logReader, _ := client.ContainerLogs(context.Background(), id, since, container.STDALL)

	actual, _ := io.ReadAll(logReader)
	assert.Equal(t, string(b), string(actual), "message doesn't match expected")
	proxy.AssertExpectations(t)
}

func Test_dockerClient_ContainerLogs_error(t *testing.T) {
	id := "123456"
	proxy := new(mockedProxy)

	proxy.On("ContainerLogs", mock.Anything, id, mock.Anything).Return(nil, errors.New("test"))

	client := &DockerClient{proxy, container.Host{ID: "localhost"}, system.Info{}}

	reader, err := client.ContainerLogs(context.Background(), id, time.Time{}, container.STDALL)

	assert.Nil(t, reader, "reader should be nil")
	assert.Error(t, err, "error should have been returned")
	proxy.AssertExpectations(t)
}

func Test_dockerClient_FindContainer_happy(t *testing.T) {
	proxy := new(mockedProxy)

	state := &docker.State{Status: "running", StartedAt: time.Now().Format(time.RFC3339Nano)}

	json := docker.InspectResponse{
		ContainerJSONBase: &docker.ContainerJSONBase{ID: "abcdefghijklmnopqrst", State: state, HostConfig: &docker.HostConfig{}},
		Config:            &docker.Config{Tty: false},
	}
	proxy.On("ContainerInspect", mock.Anything, "abcdefghijkl").Return(json, nil)

	client := &DockerClient{proxy, container.Host{ID: "localhost"}, system.Info{}}

	container, err := client.FindContainer(context.Background(), "abcdefghijkl")
	require.NoError(t, err, "error should not be thrown")

	assert.Equal(t, container.ID, "abcdefghijkl")

	proxy.AssertExpectations(t)
}

func Test_dockerClient_FindContainer_error(t *testing.T) {
	proxy := new(mockedProxy)
	proxy.On("ContainerInspect", mock.Anything, "not_valid").Return(docker.InspectResponse{}, errors.New("not found"))
	client := &DockerClient{proxy, container.Host{ID: "localhost"}, system.Info{}}

	_, err := client.FindContainer(context.Background(), "not_valid")
	require.Error(t, err, "error should be thrown")

	proxy.AssertExpectations(t)
}

func Test_dockerClient_ContainerActions_happy(t *testing.T) {
	proxy := new(mockedProxy)
	client := &DockerClient{proxy, container.Host{ID: "localhost"}, system.Info{}}

	state := &docker.State{Status: "running", StartedAt: time.Now().Format(time.RFC3339Nano)}
	json := docker.InspectResponse{ContainerJSONBase: &docker.ContainerJSONBase{ID: "abcdefghijkl", State: state, HostConfig: &docker.HostConfig{}}, Config: &docker.Config{Tty: false}}

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
	client := &DockerClient{proxy, container.Host{ID: "localhost"}, system.Info{}}
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

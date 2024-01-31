package docker

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"

	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockedProxy struct {
	mock.Mock
	DockerCLI
}

func (m *mockedProxy) ContainerList(context.Context, container.ListOptions) ([]types.Container, error) {
	args := m.Called()
	containers, ok := args.Get(0).([]types.Container)
	if !ok && args.Get(0) != nil {
		panic("containers is not of type []types.Container")
	}
	return containers, args.Error(1)

}

func (m *mockedProxy) ContainerLogs(ctx context.Context, id string, options container.LogsOptions) (io.ReadCloser, error) {
	args := m.Called(ctx, id, options)
	reader, ok := args.Get(0).(io.ReadCloser)
	if !ok && args.Get(0) != nil {
		panic("reader is not of type io.ReadCloser")
	}
	return reader, args.Error(1)
}

func (m *mockedProxy) ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error) {
	args := m.Called(ctx, containerID)
	return args.Get(0).(types.ContainerJSON), args.Error(1)
}

func (m *mockedProxy) ContainerStats(ctx context.Context, containerID string, stream bool) (types.ContainerStats, error) {
	return types.ContainerStats{}, nil
}

func (m *mockedProxy) ContainerStart(ctx context.Context, containerID string, options container.StartOptions) error {

	args := m.Called(ctx, containerID, options)
	err := args.Get(0)

	if err != nil {
		return args.Error(0)
	}

	return nil
}

func (m *mockedProxy) ContainerStop(ctx context.Context, containerID string, options container.StopOptions) error {

	args := m.Called(ctx, containerID, options)
	err := args.Get(0)

	if err != nil {
		return args.Error(0)
	}

	return nil
}

func (m *mockedProxy) ContainerRestart(ctx context.Context, containerID string, options container.StopOptions) error {

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
	client := &_client{proxy, filters.NewArgs(), &Host{ID: "localhost"}}

	list, err := client.ListContainers()
	assert.Empty(t, list, "list should be empty")
	require.NoError(t, err, "error should not return an error.")

	proxy.AssertExpectations(t)
}

func Test_dockerClient_ListContainers_error(t *testing.T) {
	proxy := new(mockedProxy)
	proxy.On("ContainerList", mock.Anything, mock.Anything).Return(nil, errors.New("test"))
	client := &_client{proxy, filters.NewArgs(), &Host{ID: "localhost"}}

	list, err := client.ListContainers()
	assert.Nil(t, list, "list should be nil")
	require.Error(t, err, "test.")

	proxy.AssertExpectations(t)
}

func Test_dockerClient_ListContainers_happy(t *testing.T) {
	containers := []types.Container{
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
	client := &_client{proxy, filters.NewArgs(), &Host{ID: "localhost"}}

	list, err := client.ListContainers()
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
	options := container.LogsOptions{ShowStdout: true, ShowStderr: true, Follow: true, Tail: "300", Timestamps: true, Since: "since"}
	proxy.On("ContainerLogs", mock.Anything, id, options).Return(reader, nil)

	client := &_client{proxy, filters.NewArgs(), &Host{ID: "localhost"}}
	logReader, _ := client.ContainerLogs(context.Background(), id, "since", STDALL)

	actual, _ := io.ReadAll(logReader)
	assert.Equal(t, string(b), string(actual), "message doesn't match expected")
	proxy.AssertExpectations(t)
}

func Test_dockerClient_ContainerLogs_error(t *testing.T) {
	id := "123456"
	proxy := new(mockedProxy)

	proxy.On("ContainerLogs", mock.Anything, id, mock.Anything).Return(nil, errors.New("test"))

	client := &_client{proxy, filters.NewArgs(), &Host{ID: "localhost"}}

	reader, err := client.ContainerLogs(context.Background(), id, "", STDALL)

	assert.Nil(t, reader, "reader should be nil")
	assert.Error(t, err, "error should have been returned")
	proxy.AssertExpectations(t)
}

func Test_dockerClient_FindContainer_happy(t *testing.T) {
	containers := []types.Container{
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

	json := types.ContainerJSON{Config: &container.Config{Tty: false}}
	proxy.On("ContainerInspect", mock.Anything, "abcdefghijkl").Return(json, nil)

	client := &_client{proxy, filters.NewArgs(), &Host{ID: "localhost"}}

	container, err := client.FindContainer("abcdefghijkl")
	require.NoError(t, err, "error should not be thrown")

	assert.Equal(t, container.ID, "abcdefghijkl")

	proxy.AssertExpectations(t)
}
func Test_dockerClient_FindContainer_error(t *testing.T) {
	containers := []types.Container{
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
	client := &_client{proxy, filters.NewArgs(), &Host{ID: "localhost"}}

	_, err := client.FindContainer("not_valid")
	require.Error(t, err, "error should be thrown")

	proxy.AssertExpectations(t)
}

func Test_dockerClient_ContainerActions_happy(t *testing.T) {
	containers := []types.Container{
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
	client := &_client{proxy, filters.NewArgs(), &Host{ID: "localhost"}}
	json := types.ContainerJSON{Config: &container.Config{Tty: false}}
	proxy.On("ContainerList", mock.Anything, mock.Anything).Return(containers, nil)
	proxy.On("ContainerInspect", mock.Anything, "abcdefghijkl").Return(json, nil)
	proxy.On("ContainerStart", mock.Anything, "abcdefghijkl", mock.Anything).Return(nil)
	proxy.On("ContainerStop", mock.Anything, "abcdefghijkl", mock.Anything).Return(nil)
	proxy.On("ContainerRestart", mock.Anything, "abcdefghijkl", mock.Anything).Return(nil)

	container, err := client.FindContainer("abcdefghijkl")
	require.NoError(t, err, "error should not be thrown")

	assert.Equal(t, container.ID, "abcdefghijkl")

	actions := []string{"start", "stop", "restart"}
	for _, action := range actions {
		err := client.ContainerActions(action, container.ID)
		require.NoError(t, err, "error should not be thrown")
		assert.Equal(t, err, nil)
	}

	proxy.AssertExpectations(t)
}

func Test_dockerClient_ContainerActions_error(t *testing.T) {
	containers := []types.Container{
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
	client := &_client{proxy, filters.NewArgs(), &Host{ID: "localhost"}}

	proxy.On("ContainerList", mock.Anything, mock.Anything).Return(containers, nil)
	proxy.On("ContainerStart", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("test"))
	proxy.On("ContainerStop", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("test"))
	proxy.On("ContainerRestart", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("test"))

	container, err := client.FindContainer("random-id")
	require.Error(t, err, "error should be thrown")

	actions := []string{"start", "stop", "restart"}
	for _, action := range actions {
		err := client.ContainerActions(action, container.ID)
		require.Error(t, err, "error should be thrown")
		assert.Error(t, err, "error should have been returned")
	}

	proxy.AssertExpectations(t)
}

package docker

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"github.com/docker/docker/api/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"io"
	"io/ioutil"
	"testing"
)

type mockedProxy struct {
	mock.Mock
	dockerProxy
}

func (m *mockedProxy) ContainerList(context.Context, types.ContainerListOptions) ([]types.Container, error) {
	args := m.Called()
	containers, ok := args.Get(0).([]types.Container)
	if !ok && args.Get(0) != nil {
		panic("containers is not of type []types.Container")
	}
	return containers, args.Error(1)

}

func (m *mockedProxy) ContainerLogs(ctx context.Context, id string, options types.ContainerLogsOptions) (io.ReadCloser, error) {
	args := m.Called(ctx, id, options)
	reader, ok := args.Get(0).(io.ReadCloser)
	if !ok && args.Get(0) != nil {
		panic("reader is not of type io.ReadCloser")
	}
	return reader, args.Error(1)
}

func Test_dockerClient_ListContainers_null(t *testing.T) {
	proxy := new(mockedProxy)
	proxy.On("ContainerList", mock.Anything, mock.Anything).Return(nil, nil)
	client := &dockerClient{proxy}

	list, err := client.ListContainers()
	assert.Empty(t, list, "list should be empty")
	require.NoError(t, err, "error should not return an error.")

	proxy.AssertExpectations(t)
}

func Test_dockerClient_ListContainers_error(t *testing.T) {
	proxy := new(mockedProxy)
	proxy.On("ContainerList", mock.Anything, mock.Anything).Return(nil, errors.New("test"))
	client := &dockerClient{proxy}

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
	client := &dockerClient{proxy}

	list, err := client.ListContainers()
	require.NoError(t, err, "error should not return an error.")

	assert.Equal(t, list, []Container{
		{
			ID:    "1234567890_a",
			Name:  "a_test_container",
			Names: []string{"/a_test_container"},
		},
		{
			ID:    "abcdefghijkl",
			Name:  "z_test_container",
			Names: []string{"/z_test_container"},
		},
	})

	proxy.AssertExpectations(t)
}

func Test_dockerClient_ContainerLogs_happy(t *testing.T) {
	id := "123456"

	proxy := new(mockedProxy)
	expected := "INFO Testing logs..."
	b := make([]byte, 8)

	binary.BigEndian.PutUint32(b[4:], uint32(len(expected)))
	b = append(b, []byte(expected)...)

	var reader io.ReadCloser
	reader = ioutil.NopCloser(bytes.NewReader(b))
	options := types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true, Follow: true, Tail: "300", Timestamps: true}
	proxy.On("ContainerLogs", mock.Anything, id, options).Return(reader, nil)

	client := &dockerClient{proxy}
	messages, _ := client.ContainerLogs(context.Background(), id, 300)

	actual, _ := <-messages
	assert.Equal(t, expected, actual, "message doesn't match expected")

	_, ok := <-messages
	assert.False(t, ok, "channel should have been closed")
	proxy.AssertExpectations(t)
}

func Test_dockerClient_ContainerLogs_error(t *testing.T) {
	id := "123456"
	proxy := new(mockedProxy)

	proxy.On("ContainerLogs", mock.Anything, id, mock.Anything).Return(nil, errors.New("test"))

	client := &dockerClient{proxy}

	messages, err := client.ContainerLogs(context.Background(), id, 300)

	assert.Nil(t, messages, "messages should be nil")

	e, _ := <-err
	assert.Error(t, e, "error should have been returned")
	_, ok := <-err
	assert.False(t, ok, "error channel should have been closed")
	proxy.AssertExpectations(t)
}

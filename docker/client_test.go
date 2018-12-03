package docker

import (
	"context"
	"errors"
	"github.com/docker/docker/api/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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
func Test_dockerClient_ListContainers_null(t *testing.T) {
	proxy := mockedProxy{}
	proxy.On("ContainerList", mock.Anything, mock.Anything).Return(nil, nil)
	client := &dockerClient{&proxy}

	list, err := client.ListContainers()
	assert.Empty(t, list, "list should be empty")
	require.NoError(t, err, "error should not return an error.")

	proxy.AssertExpectations(t)
}

func Test_dockerClient_ListContainers_error(t *testing.T) {
	proxy := mockedProxy{}
	proxy.On("ContainerList", mock.Anything, mock.Anything).Return(nil, errors.New("test"))
	client := &dockerClient{&proxy}

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

	proxy := mockedProxy{}
	proxy.On("ContainerList", mock.Anything, mock.Anything).Return(containers, nil)
	client := &dockerClient{&proxy}

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

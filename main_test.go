package main

import (
	"context"
	"errors"
	"testing"

	"github.com/amir20/dozzle/internal/docker"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type fakeCLI struct {
	docker.DockerCLI
	mock.Mock
}

func (f *fakeCLI) ContainerList(context.Context, types.ContainerListOptions) ([]types.Container, error) {
	args := f.Called()
	return args.Get(0).([]types.Container), args.Error(1)
}

func Test_valid_localhost(t *testing.T) {
	client := new(fakeCLI)
	client.On("ContainerList").Return([]types.Container{}, nil)
	fakeClientFactory := func(filter map[string][]string) (*docker.Client, error) {
		return docker.NewClient(client, filters.NewArgs(), &docker.Host{
			ID: "localhost",
		}), nil
	}

	args := args{}

	actualClient := createLocalClient(args, fakeClientFactory)

	assert.NotNil(t, actualClient)
	client.AssertExpectations(t)
}

func Test_invalid_localhost(t *testing.T) {
	client := new(fakeCLI)
	client.On("ContainerList").Return([]types.Container{}, errors.New("error"))
	fakeClientFactory := func(filter map[string][]string) (*docker.Client, error) {
		return docker.NewClient(client, filters.NewArgs(), &docker.Host{
			ID: "localhost",
		}), nil
	}

	args := args{}

	actualClient := createLocalClient(args, fakeClientFactory)

	assert.Nil(t, actualClient)
	client.AssertExpectations(t)
}

func Test_valid_remote(t *testing.T) {
	local := new(fakeCLI)
	local.On("ContainerList").Return([]types.Container{}, errors.New("error"))
	fakeLocalClientFactory := func(filter map[string][]string) (*docker.Client, error) {
		return docker.NewClient(local, filters.NewArgs(), &docker.Host{
			ID: "localhost",
		}), nil
	}

	remote := new(fakeCLI)
	remote.On("ContainerList").Return([]types.Container{}, nil)
	fakeRemoteClientFactory := func(filter map[string][]string, host docker.Host) (*docker.Client, error) {
		return docker.NewClient(remote, filters.NewArgs(), &docker.Host{
			ID: "test",
		}), nil
	}

	args := args{
		RemoteHost: []string{"tcp://test:2375"},
	}

	clients := createClients(args, fakeLocalClientFactory, fakeRemoteClientFactory, "")

	assert.Equal(t, 1, len(clients))
	assert.Contains(t, clients, "test")
	assert.NotContains(t, clients, "localhost")
	local.AssertExpectations(t)
	remote.AssertExpectations(t)
}

func Test_valid_remote_and_local(t *testing.T) {
	local := new(fakeCLI)
	local.On("ContainerList").Return([]types.Container{}, nil)
	fakeLocalClientFactory := func(filter map[string][]string) (*docker.Client, error) {
		return docker.NewClient(local, filters.NewArgs(), &docker.Host{
			ID: "localhost",
		}), nil
	}

	remote := new(fakeCLI)
	remote.On("ContainerList").Return([]types.Container{}, nil)
	fakeRemoteClientFactory := func(filter map[string][]string, host docker.Host) (*docker.Client, error) {
		return docker.NewClient(remote, filters.NewArgs(), &docker.Host{
			ID: "test",
		}), nil
	}
	args := args{
		RemoteHost: []string{"tcp://test:2375"},
	}

	clients := createClients(args, fakeLocalClientFactory, fakeRemoteClientFactory, "")

	assert.Equal(t, 2, len(clients))
	assert.Contains(t, clients, "test")
	assert.Contains(t, clients, "localhost")
	local.AssertExpectations(t)
	remote.AssertExpectations(t)
}

func Test_no_clients(t *testing.T) {
	local := new(fakeCLI)
	local.On("ContainerList").Return([]types.Container{}, errors.New("error"))
	fakeLocalClientFactory := func(filter map[string][]string) (*docker.Client, error) {

		return docker.NewClient(local, filters.NewArgs(), &docker.Host{
			ID: "localhost",
		}), nil
	}
	fakeRemoteClientFactory := func(filter map[string][]string, host docker.Host) (*docker.Client, error) {
		client := new(fakeCLI)
		return docker.NewClient(client, filters.NewArgs(), &docker.Host{
			ID: "test",
		}), nil
	}

	args := args{}

	clients := createClients(args, fakeLocalClientFactory, fakeRemoteClientFactory, "")

	assert.Equal(t, 0, len(clients))
	local.AssertExpectations(t)
}

package main

import (
	"errors"
	"testing"

	"github.com/amir20/dozzle/docker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type fakeClient struct {
	docker.Client
	mock.Mock
}

func (f *fakeClient) ListContainers() ([]docker.Container, error) {
	args := f.Called()
	return args.Get(0).([]docker.Container), args.Error(1)
}

func Test_valid_localhost(t *testing.T) {
	fakeClientFactory := func(filter map[string][]string) (docker.Client, error) {
		client := new(fakeClient)
		client.On("ListContainers").Return([]docker.Container{}, nil)
		return client, nil
	}

	args := args{}

	actualClient := createLocalClient(args, fakeClientFactory)

	assert.NotNil(t, actualClient)
}

func Test_invalid_localhost(t *testing.T) {
	fakeClientFactory := func(filter map[string][]string) (docker.Client, error) {
		client := new(fakeClient)
		client.On("ListContainers").Return([]docker.Container{}, errors.New("error"))
		return client, nil
	}

	args := args{}

	actualClient := createLocalClient(args, fakeClientFactory)

	assert.Nil(t, actualClient)
}

func Test_valid_remote(t *testing.T) {
	fakeLocalClientFactory := func(filter map[string][]string) (docker.Client, error) {
		client := new(fakeClient)
		client.On("ListContainers").Return([]docker.Container{}, errors.New("error"))
		return client, nil
	}

	fakeRemoteClientFactory := func(filter map[string][]string, host string) (docker.Client, error) {
		client := new(fakeClient)
		return client, nil
	}

	args := args{
		RemoteHost: []string{"tcp://localhost:2375"},
	}

	clients := createClients(args, fakeLocalClientFactory, fakeRemoteClientFactory)

	assert.Equal(t, 1, len(clients))
	assert.Contains(t, clients, "tcp://localhost:2375")
	assert.NotContains(t, clients, "localhost")
}

func Test_valid_remote_and_local(t *testing.T) {
	fakeLocalClientFactory := func(filter map[string][]string) (docker.Client, error) {
		client := new(fakeClient)
		client.On("ListContainers").Return([]docker.Container{}, nil)
		return client, nil
	}

	fakeRemoteClientFactory := func(filter map[string][]string, host string) (docker.Client, error) {
		client := new(fakeClient)
		return client, nil
	}

	args := args{
		RemoteHost: []string{"tcp://localhost:2375"},
	}

	clients := createClients(args, fakeLocalClientFactory, fakeRemoteClientFactory)

	assert.Equal(t, 2, len(clients))
	assert.Contains(t, clients, "tcp://localhost:2375")
	assert.Contains(t, clients, "localhost")
}

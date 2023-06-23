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

func (f *fakeClient) Host() string {
	args := f.Called()
	return args.String(0)
}

func Test_valid_localhost(t *testing.T) {
	fakeClientFactory := func(filter map[string][]string) (docker.Client, error) {
		client := new(fakeClient)
		client.On("ListContainers").Return([]docker.Container{}, nil)
		client.On("Host").Return("localhost")
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
		client.On("Host").Return("localhost")
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
		client.On("Host").Return("localhost")

		return client, nil
	}

	fakeRemoteClientFactory := func(filter map[string][]string, host string) (docker.Client, error) {
		client := new(fakeClient)
		client.On("Host").Return("test")
		return client, nil
	}

	args := args{
		RemoteHost: []string{"tcp://test:2375"},
	}

	clients := createClients(args, fakeLocalClientFactory, fakeRemoteClientFactory)

	assert.Equal(t, 1, len(clients))
	assert.Contains(t, clients, "test")
	assert.NotContains(t, clients, "localhost")
}

func Test_valid_remote_and_local(t *testing.T) {
	fakeLocalClientFactory := func(filter map[string][]string) (docker.Client, error) {
		client := new(fakeClient)
		client.On("ListContainers").Return([]docker.Container{}, nil)
		client.On("Host").Return("localhost")
		return client, nil
	}

	fakeRemoteClientFactory := func(filter map[string][]string, host string) (docker.Client, error) {
		client := new(fakeClient)
		client.On("Host").Return("test")
		return client, nil
	}

	args := args{
		RemoteHost: []string{"tcp://test:2375"},
	}

	clients := createClients(args, fakeLocalClientFactory, fakeRemoteClientFactory)

	assert.Equal(t, 2, len(clients))
	assert.Contains(t, clients, "test")
	assert.Contains(t, clients, "localhost")
}

func Test_no_clients(t *testing.T) {
	fakeLocalClientFactory := func(filter map[string][]string) (docker.Client, error) {
		client := new(fakeClient)
		client.On("ListContainers").Return([]docker.Container{}, errors.New("error"))
		client.On("Host").Return("localhost")
		return client, nil
	}

	fakeRemoteClientFactory := func(filter map[string][]string, host string) (docker.Client, error) {
		client := new(fakeClient)
		client.On("Host").Return("test")
		return client, nil
	}

	args := args{}

	clients := createClients(args, fakeLocalClientFactory, fakeRemoteClientFactory)

	assert.Equal(t, 0, len(clients))
}

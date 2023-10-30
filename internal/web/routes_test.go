package web

import (
	"context"
	"time"

	"io"
	"io/fs"

	"github.com/amir20/dozzle/internal/docker"
	"github.com/go-chi/chi/v5"

	"github.com/stretchr/testify/mock"

	"github.com/spf13/afero"
)

type MockedClient struct {
	mock.Mock
	DockerClient
}

func (m *MockedClient) FindContainer(id string) (docker.Container, error) {
	args := m.Called(id)
	return args.Get(0).(docker.Container), args.Error(1)
}

func (m *MockedClient) ListContainers() ([]docker.Container, error) {
	args := m.Called()
	return args.Get(0).([]docker.Container), args.Error(1)
}

func (m *MockedClient) ContainerLogs(ctx context.Context, id string, since string, stdType docker.StdType) (io.ReadCloser, error) {
	args := m.Called(ctx, id, since, stdType)
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func (m *MockedClient) Events(ctx context.Context, events chan<- docker.ContainerEvent) <-chan error {
	args := m.Called(ctx, events)
	return args.Get(0).(chan error)
}

func (m *MockedClient) ContainerStats(context.Context, string, chan<- docker.ContainerStat) error {
	return nil
}

func (m *MockedClient) ContainerLogsBetweenDates(ctx context.Context, id string, from time.Time, to time.Time, stdType docker.StdType) (io.ReadCloser, error) {
	args := m.Called(ctx, id, from, to, stdType)
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func (m *MockedClient) Host() *docker.Host {
	args := m.Called()
	return args.Get(0).(*docker.Host)
}

func createHandler(client DockerClient, content fs.FS, config Config) *chi.Mux {
	if client == nil {
		client = new(MockedClient)
		client.(*MockedClient).On("ListContainers").Return([]docker.Container{}, nil)
		client.(*MockedClient).On("Host").Return(&docker.Host{
			ID: "localhost",
		})
	}

	if content == nil {
		fs := afero.NewMemMapFs()
		afero.WriteFile(fs, "index.html", []byte("index page"), 0644)
		content = afero.NewIOFS(fs)
	}

	clients := map[string]DockerClient{
		"localhost": client,
	}
	return createRouter(&handler{
		clients: clients,
		content: content,
		config:  &config,
	})
}

func createDefaultHandler(client DockerClient) *chi.Mux {
	return createHandler(client, nil, Config{Base: "/", AuthProvider: NONE})
}

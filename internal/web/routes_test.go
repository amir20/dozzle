package web

import (
	"context"
	"time"

	"io"
	"io/fs"

	"github.com/amir20/dozzle/internal/docker"
	docker_support "github.com/amir20/dozzle/internal/support/docker"
	"github.com/docker/docker/api/types/system"
	"github.com/go-chi/chi/v5"

	"github.com/stretchr/testify/mock"

	"github.com/spf13/afero"
)

type MockedClient struct {
	mock.Mock
	docker.Client
}

func (m *MockedClient) FindContainer(id string) (docker.Container, error) {
	args := m.Called(id)
	return args.Get(0).(docker.Container), args.Error(1)
}

func (m *MockedClient) ContainerActions(action docker.ContainerAction, containerID string) error {
	args := m.Called(action, containerID)
	return args.Error(0)
}

func (m *MockedClient) ContainerEvents(ctx context.Context, events chan<- docker.ContainerEvent) error {
	args := m.Called(ctx, events)
	return args.Error(0)
}

func (m *MockedClient) ListContainers() ([]docker.Container, error) {
	args := m.Called()
	return args.Get(0).([]docker.Container), args.Error(1)
}

func (m *MockedClient) ContainerLogs(ctx context.Context, id string, since time.Time, stdType docker.StdType) (io.ReadCloser, error) {
	args := m.Called(ctx, id, since, stdType)
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func (m *MockedClient) ContainerStats(context.Context, string, chan<- docker.ContainerStat) error {
	return nil
}

func (m *MockedClient) ContainerLogsBetweenDates(ctx context.Context, id string, from time.Time, to time.Time, stdType docker.StdType) (io.ReadCloser, error) {
	args := m.Called(ctx, id, from, to, stdType)
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func (m *MockedClient) Host() docker.Host {
	args := m.Called()
	return args.Get(0).(docker.Host)
}

func (m *MockedClient) IsSwarmMode() bool {
	return false
}

func (m *MockedClient) SystemInfo() system.Info {
	return system.Info{ID: "123"}
}

func createHandler(client docker.Client, content fs.FS, config Config) *chi.Mux {
	if client == nil {
		client = new(MockedClient)
		client.(*MockedClient).On("ListContainers").Return([]docker.Container{}, nil)
		client.(*MockedClient).On("Host").Return(docker.Host{
			ID: "localhost",
		})
		client.(*MockedClient).On("ContainerEvents", mock.Anything, mock.AnythingOfType("chan<- docker.ContainerEvent")).Return(nil)
	}

	if content == nil {
		fs := afero.NewMemMapFs()
		afero.WriteFile(fs, "index.html", []byte("index page"), 0644)
		content = afero.NewIOFS(fs)
	}

	multiHostService := docker_support.NewMultiHostService([]docker_support.ClientService{docker_support.NewDockerClientService(client)})
	return createRouter(&handler{
		multiHostService: multiHostService,
		content:          content,
		config:           &config,
	})
}

func createDefaultHandler(client docker.Client) *chi.Mux {
	return createHandler(client, nil, Config{Base: "/", Authorization: Authorization{Provider: NONE}})
}

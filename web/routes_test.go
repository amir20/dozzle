package web

import (
	"context"
	"time"

	"io"
	"io/fs"

	"github.com/gorilla/mux"

	"github.com/amir20/dozzle/docker"

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

func (m *MockedClient) ListContainers() ([]docker.Container, error) {
	args := m.Called()
	return args.Get(0).([]docker.Container), args.Error(1)
}

func (m *MockedClient) ContainerLogs(ctx context.Context, id string, since string) (io.ReadCloser, error) {
	args := m.Called(ctx, id, since)
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func (m *MockedClient) Events(ctx context.Context) (<-chan docker.ContainerEvent, <-chan error) {
	args := m.Called(ctx)
	channel, ok := args.Get(0).(chan docker.ContainerEvent)
	if !ok {
		panic("channel is not of type chan events.Message")
	}

	err, ok := args.Get(1).(chan error)
	if !ok {
		panic("error is not of type chan error")
	}
	return channel, err
}

func (m *MockedClient) ContainerStats(context.Context, string, chan<- docker.ContainerStat) error {
	return nil
}

func (m *MockedClient) ContainerLogsBetweenDates(ctx context.Context, id string, from time.Time, to time.Time) (io.ReadCloser, error) {
	args := m.Called(ctx, id, from, to)
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func createHandler(client docker.Client, content fs.FS, config Config) *mux.Router {
	if client == nil {
		client = new(MockedClient)
		client.(*MockedClient).On("ListContainers").Return([]docker.Container{}, nil)
	}

	if content == nil {
		fs := afero.NewMemMapFs()
		afero.WriteFile(fs, "index.html", []byte("index page"), 0644)
		content = afero.NewIOFS(fs)
	}

	clients := map[string]docker.Client{
		"localhost": client,
	}
	return createRouter(&handler{
		clients: clients,
		content: content,
		config:  &config,
	})
}

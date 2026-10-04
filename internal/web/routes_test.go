package web

import (
	"context"
	"crypto/tls"
	"errors"
	"time"

	"io"
	"io/fs"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/container/docker"
	"github.com/amir20/dozzle/internal/container/docker/swap"
	"github.com/amir20/dozzle/internal/container/docker/swap/swaptest"
	"github.com/amir20/dozzle/internal/hostservice"
	"github.com/go-chi/chi/v5"
	docker_types "github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/image"
	"github.com/moby/moby/api/types/system"

	"github.com/stretchr/testify/mock"

	"github.com/spf13/afero"
)

type MockedClient struct {
	mock.Mock
	container.Client
	// engine is what an update swaps containers with.
	engine *swaptest.Fake
}

func (m *MockedClient) FindContainer(ctx context.Context, id string) (container.Container, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(container.Container), args.Error(1)
}

func (m *MockedClient) ContainerActions(ctx context.Context, action container.ContainerAction, containerID string) error {
	args := m.Called(ctx, action, containerID)
	return args.Error(0)
}

func (m *MockedClient) ImagePull(ctx context.Context, imageName string) (io.ReadCloser, error) {
	args := m.Called(ctx, imageName)
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func (m *MockedClient) ImageRepoDigests(ctx context.Context, imageID string) ([]string, error) {
	args := m.Called(ctx, imageID)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockedClient) ImageID(ctx context.Context, ref string) (string, error) {
	args := m.Called(ctx, ref)
	return args.String(0), args.Error(1)
}

func (m *MockedClient) ImageInspect(ctx context.Context, ref string) (image.InspectResponse, error) {
	return image.InspectResponse{}, errors.New("no such image")
}

func (m *MockedClient) SwapAPI() swap.API {
	if m.engine == nil {
		m.engine = swaptest.New()
	}
	return m.engine
}

func (m *MockedClient) ContainerInspect(ctx context.Context, containerID string) (docker_types.InspectResponse, error) {
	args := m.Called(ctx, containerID)
	return args.Get(0).(docker_types.InspectResponse), args.Error(1)
}

func (m *MockedClient) ContainerRemove(ctx context.Context, containerID string) error {
	args := m.Called(ctx, containerID)
	return args.Error(0)
}

func (m *MockedClient) ContainerCreate(ctx context.Context, inspectResp docker_types.InspectResponse, name string) (string, error) {
	args := m.Called(ctx, inspectResp, name)
	return args.Get(0).(string), args.Error(1)
}

func (m *MockedClient) NetworkDependents(ctx context.Context, id string, name string) ([]string, error) {
	args := m.Called(ctx, id, name)
	ids, _ := args.Get(0).([]string)
	return ids, args.Error(1)
}

func (m *MockedClient) ServiceUpdate(ctx context.Context, serviceID string, imageName string) error {
	args := m.Called(ctx, serviceID, imageName)
	return args.Error(0)
}

func (m *MockedClient) ContainerEvents(ctx context.Context, events chan<- container.ContainerEvent) error {
	args := m.Called(ctx, events)
	return args.Error(0)
}

func (m *MockedClient) ListContainers(ctx context.Context, labels container.ContainerLabels) ([]container.Container, error) {
	args := m.Called(ctx, labels)
	return args.Get(0).([]container.Container), args.Error(1)
}

func (m *MockedClient) ContainerLogs(ctx context.Context, id string, since time.Time, stdType container.StdType) (io.ReadCloser, error) {
	args := m.Called(ctx, id, since, stdType)
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func (m *MockedClient) ContainerStats(context.Context, string, chan<- container.ContainerStat) error {
	return nil
}

func (m *MockedClient) ContainerLogsBetweenDates(ctx context.Context, id string, from time.Time, to time.Time, stdType container.StdType) (io.ReadCloser, error) {
	args := m.Called(ctx, id, from, to, stdType)
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func (m *MockedClient) ContainerLogsTail(ctx context.Context, id string, lines int) (io.ReadCloser, error) {
	args := m.Called(ctx, id, lines)
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func (m *MockedClient) Host() container.Host {
	args := m.Called()
	return args.Get(0).(container.Host)
}

func (m *MockedClient) IsSwarmMode() bool {
	return false
}

func (m *MockedClient) SystemInfo() system.Info {
	return system.Info{ID: "123"}
}

func createHandler(client docker.UpdateClient, content fs.FS, config Config) *chi.Mux {
	if client == nil {
		client = new(MockedClient)
		client.(*MockedClient).On("ListContainers", mock.Anything, mock.Anything).Return([]container.Container{}, nil)
		client.(*MockedClient).On("Host").Return(container.Host{
			ID: "localhost",
		})
		client.(*MockedClient).On("ContainerEvents", mock.Anything, mock.AnythingOfType("chan<- container.ContainerEvent")).Return(nil)
	}

	if content == nil {
		fs := afero.NewMemMapFs()
		afero.WriteFile(fs, "index.html", []byte("index page"), 0644)
		content = afero.NewIOFS(fs)
	}

	manager := hostservice.NewRetriableClientManager(nil, nil, 3*time.Second, tls.Certificate{}, docker.NewService(client, container.ContainerLabels{}))
	multiHostService := hostservice.NewMultiHostService(manager, 3*time.Second)
	return createRouter(&handler{
		hostService: multiHostService,
		content:     content,
		config:      &config,
	})
}

func createDefaultHandler(client docker.UpdateClient) *chi.Mux {
	return createHandler(client, nil, Config{Base: "/", Authorization: Authorization{Provider: NONE}})
}

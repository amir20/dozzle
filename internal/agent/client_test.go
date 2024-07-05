package agent

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"os"
	"path"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/docker"
	"github.com/amir20/dozzle/internal/utils"
	"github.com/docker/docker/api/types/system"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

var lis *bufconn.Listener
var certs tls.Certificate
var client *MockedClient

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

func init() {
	lis = bufconn.Listen(bufSize)

	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	root := path.Join(cwd, "../../")
	certs, err = tls.LoadX509KeyPair(path.Join(root, "shared_cert.pem"), path.Join(root, "shared_key.pem"))
	if err != nil {
		panic(err)
	}

	client = &MockedClient{}
	client.On("ListContainers").Return([]docker.Container{
		{
			ID:   "123456",
			Name: "test",
			Host: "localhost",
		},
	}, nil)
	client.On("Host").Return(docker.Host{
		ID:       "localhost",
		Endpoint: "local",
		Name:     "local",
	})
	client.On("ContainerEvents", mock.Anything, mock.AnythingOfType("chan<- docker.ContainerEvent")).Return(nil).Run(func(args mock.Arguments) {
		time.Sleep(5 * time.Second)
	})

	client.On("FindContainer", "123456").Return(docker.Container{
		ID:        "123456",
		Name:      "test",
		Host:      "localhost",
		Image:     "test",
		ImageID:   "test",
		StartedAt: &time.Time{},
		State:     "running",
		Status:    "running",
		Health:    "healthy",
		Group:     "test",
		Command:   "test",
		Created:   time.Time{},
		Tty:       true,
		Labels: map[string]string{
			"test": "test",
		},
		Stats: utils.NewRingBuffer[docker.ContainerStat](300),
	}, nil)

	go RunServer(client, certs, lis)
}

func bufDialer(ctx context.Context, address string) (net.Conn, error) {
	return lis.Dial()
}

func TestFindContainer(t *testing.T) {
	rpc, err := NewClient("passthrough://bufnet", certs, grpc.WithContextDialer(bufDialer))
	if err != nil {
		t.Fatal(err)
	}

	container, _ := rpc.FindContainer("123456")

	assert.Equal(t, container, docker.Container{
		ID:        "123456",
		Name:      "test",
		Host:      "localhost",
		Image:     "test",
		ImageID:   "test",
		StartedAt: &time.Time{},
		State:     "running",
		Status:    "running",
		Health:    "healthy",
		Group:     "test",
		Command:   "test",
		Created:   time.Time{},
		Tty:       true,
		Labels: map[string]string{
			"test": "test",
		},
		Stats: utils.NewRingBuffer[docker.ContainerStat](300),
	})
}

func TestListContainers(t *testing.T) {
	rpc, err := NewClient("passthrough://bufnet", certs, grpc.WithContextDialer(bufDialer))
	if err != nil {
		t.Fatal(err)
	}

	containers, _ := rpc.ListContainers()

	assert.Equal(t, containers, []docker.Container{
		{
			ID:        "123456",
			Name:      "test",
			Host:      "localhost",
			Image:     "test",
			ImageID:   "test",
			StartedAt: &time.Time{},
			State:     "running",
			Status:    "running",
			Health:    "healthy",
			Group:     "test",
			Command:   "test",
			Created:   time.Time{},
			Tty:       true,
			Labels: map[string]string{
				"test": "test",
			},
			Stats: utils.NewRingBuffer[docker.ContainerStat](300),
		},
	})
}

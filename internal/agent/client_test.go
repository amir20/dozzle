package agent

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/utils"
	"github.com/docker/docker/api/types/system"
	"github.com/go-faker/faker/v4"
	"github.com/go-faker/faker/v4/pkg/options"
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
	container.Client
}

func (m *MockedClient) FindContainer(ctx context.Context, id string) (container.Container, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(container.Container), args.Error(1)
}

func (m *MockedClient) ContainerActions(ctx context.Context, action container.ContainerAction, containerID string) error {
	args := m.Called(ctx, action, containerID)
	return args.Error(0)
}

func (m *MockedClient) ContainerEvents(ctx context.Context, events chan<- container.ContainerEvent) error {
	args := m.Called(ctx, events)
	return args.Error(0)
}

func (m *MockedClient) ListContainers(ctx context.Context, filter container.ContainerLabels) ([]container.Container, error) {
	args := m.Called(ctx, filter)
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

var wantedContainer = container.Container{}

func init() {
	faker.FakeData(&wantedContainer, options.WithFieldsToIgnore("Stats"))
	wantedContainer.FinishedAt = wantedContainer.FinishedAt.UTC()
	wantedContainer.Created = wantedContainer.Created.UTC()
	wantedContainer.StartedAt = wantedContainer.StartedAt.UTC()
	wantedContainer.Stats = utils.NewRingBuffer[container.ContainerStat](300)

	fmt.Printf("Fake data generated %+v", wantedContainer)
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
	client.On("ListContainers", mock.Anything, mock.Anything).Return([]container.Container{
		{
			ID:    "123456",
			Name:  "test",
			Host:  "localhost",
			State: "running",
		},
	}, nil)

	client.On("Host").Return(container.Host{
		ID:       "localhost",
		Endpoint: "local",
		Name:     "local",
	})

	client.On("ContainerEvents", mock.Anything, mock.AnythingOfType("chan<- container.ContainerEvent")).Return(nil).Run(func(args mock.Arguments) {
		time.Sleep(5 * time.Second)
	})

	client.On("FindContainer", mock.Anything, "123456").Return(wantedContainer, nil)
	server, _ := NewServer(client, certs, "test", container.ContainerLabels{})
	go server.Serve(lis)
}

func bufDialer(ctx context.Context, address string) (net.Conn, error) {
	return lis.Dial()
}

func TestFindContainer(t *testing.T) {
	rpc, err := NewClient("passthrough://bufnet", certs, grpc.WithContextDialer(bufDialer))
	if err != nil {
		t.Fatal(err)
	}

	c, _ := rpc.FindContainer(context.Background(), "123456")

	assert.Equal(t, wantedContainer, c)
}

func TestListContainers(t *testing.T) {
	rpc, err := NewClient("passthrough://bufnet", certs, grpc.WithContextDialer(bufDialer))
	if err != nil {
		t.Fatal(err)
	}

	containers, _ := rpc.ListContainers(context.Background(), container.ContainerLabels{})

	assert.Equal(t, []container.Container{
		wantedContainer,
	}, containers)
}

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
var mockService *MockedClientService

type MockedClientService struct {
	mock.Mock
}

func (m *MockedClientService) FindContainer(ctx context.Context, id string, labels container.ContainerLabels) (container.Container, error) {
	args := m.Called(ctx, id, labels)
	return args.Get(0).(container.Container), args.Error(1)
}

func (m *MockedClientService) ListContainers(ctx context.Context, filter container.ContainerLabels) ([]container.Container, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]container.Container), args.Error(1)
}

func (m *MockedClientService) Host(ctx context.Context) (container.Host, error) {
	args := m.Called(ctx)
	return args.Get(0).(container.Host), args.Error(1)
}

func (m *MockedClientService) ContainerAction(ctx context.Context, c container.Container, action container.ContainerAction) error {
	args := m.Called(ctx, c, action)
	return args.Error(0)
}

func (m *MockedClientService) LogsBetweenDates(ctx context.Context, c container.Container, from time.Time, to time.Time, stdTypes container.StdType) (<-chan *container.LogEvent, error) {
	args := m.Called(ctx, c, from, to, stdTypes)
	return args.Get(0).(<-chan *container.LogEvent), args.Error(1)
}

func (m *MockedClientService) RawLogs(ctx context.Context, c container.Container, from time.Time, to time.Time, stdTypes container.StdType) (io.ReadCloser, error) {
	args := m.Called(ctx, c, from, to, stdTypes)
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func (m *MockedClientService) SubscribeStats(ctx context.Context, stats chan<- container.ContainerStat) {
	m.Called(ctx, stats)
}

func (m *MockedClientService) SubscribeEvents(ctx context.Context, events chan<- container.ContainerEvent) {
	m.Called(ctx, events)
}

func (m *MockedClientService) SubscribeContainersStarted(ctx context.Context, containers chan<- container.Container) {
	m.Called(ctx, containers)
}

func (m *MockedClientService) StreamLogs(ctx context.Context, c container.Container, from time.Time, stdTypes container.StdType, events chan<- *container.LogEvent) error {
	args := m.Called(ctx, c, from, stdTypes, events)
	return args.Error(0)
}

func (m *MockedClientService) Attach(ctx context.Context, c container.Container, events container.ExecEventReader, stdout io.Writer) error {
	args := m.Called(ctx, c, events, stdout)
	return args.Error(0)
}

func (m *MockedClientService) Exec(ctx context.Context, c container.Container, cmd []string, events container.ExecEventReader, stdout io.Writer) error {
	args := m.Called(ctx, c, cmd, events, stdout)
	return args.Error(0)
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

	mockService = &MockedClientService{}
	mockService.On("ListContainers", mock.Anything, mock.Anything).Return([]container.Container{
		wantedContainer,
	}, nil)

	mockService.On("Host", mock.Anything).Return(container.Host{
		ID:       "localhost",
		Endpoint: "local",
		Name:     "local",
	}, nil)

	mockService.On("SubscribeEvents", mock.Anything, mock.AnythingOfType("chan<- container.ContainerEvent")).Return().Run(func(args mock.Arguments) {
		time.Sleep(5 * time.Second)
	})

	mockService.On("SubscribeStats", mock.Anything, mock.AnythingOfType("chan<- container.ContainerStat")).Return()

	mockService.On("SubscribeContainersStarted", mock.Anything, mock.AnythingOfType("chan<- container.Container")).Return()

	mockService.On("FindContainer", mock.Anything, "123456", mock.Anything).Return(wantedContainer, nil)

	mockService.On("Client").Return(nil)

	server, _ := NewServer(mockService, certs, "test", nil)
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

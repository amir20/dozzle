package agent

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/agentcerts"
	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/container/agent/pb"
	"github.com/amir20/dozzle/internal/imagecheck"
	"github.com/amir20/dozzle/internal/notification/dispatcher"
	"github.com/amir20/dozzle/internal/utils"
	"github.com/amir20/dozzle/types"
	"github.com/go-faker/faker/v4"
	"github.com/go-faker/faker/v4/pkg/options"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

var lis *bufconn.Listener
var certs tls.Certificate
var mockService *MockedClientService

type mockNotificationHandler struct{}

func (m *mockNotificationHandler) HandleNotificationConfig(subscriptions []types.SubscriptionConfig, dispatchers []types.DispatcherConfig) error {
	return nil
}

func (m *mockNotificationHandler) SetCloudDispatcher(d dispatcher.Dispatcher) {}
func (m *mockNotificationHandler) SetCloudStreamLogs(enabled *bool)           {}
func (m *mockNotificationHandler) ClearCloudDispatcher()                      {}

func (m *mockNotificationHandler) GetNotificationStats() []types.SubscriptionStats {
	return nil
}

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

func (m *MockedClientService) LogHistogram(ctx context.Context, c container.Container, from time.Time, to time.Time, width time.Duration) (container.LogHistogram, error) {
	args := m.Called(ctx, c, from, to, width)
	return args.Get(0).(container.LogHistogram), args.Error(1)
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

func (m *MockedClientService) UpdateContainer(ctx context.Context, c container.Container, progressCh chan<- container.UpdateProgress) (bool, error) {
	defer close(progressCh)
	args := m.Called(ctx, c)
	for _, p := range args.Get(0).([]container.UpdateProgress) {
		progressCh <- p
	}
	return args.Bool(1), args.Error(2)
}

func (m *MockedClientService) RollbackContainer(ctx context.Context, c container.Container, opts container.RollbackOptions, progressCh chan<- container.UpdateProgress) error {
	defer close(progressCh)
	args := m.Called(ctx, c, opts)
	for _, p := range args.Get(0).([]container.UpdateProgress) {
		progressCh <- p
	}
	return args.Error(1)
}

func (m *MockedClientService) CheckImageUpdate(ctx context.Context, c container.Container, force bool) (imagecheck.Result, error) {
	args := m.Called(ctx, c, force)
	return args.Get(0).(imagecheck.Result), args.Error(1)
}

var wantedContainer = container.Container{}

func init() {
	faker.FakeData(&wantedContainer, options.WithFieldsToIgnore("Stats", "MountStats", "Ports", "ImageDigest"))
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
	root := path.Join(cwd, "../../../")
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

	mockService.On("UpdateContainer", mock.Anything, mock.Anything).Return([]container.UpdateProgress{
		{Status: container.UpdateRecreating},
		{Status: container.UpdateVerifying},
		{Status: container.UpdateRolledBack, Error: "replacement is unhealthy", Result: &rolledBackResult},
	}, false, nil)

	mockService.On("RollbackContainer", mock.Anything, mock.Anything, container.RollbackOptions{ExpectedFromDigest: "sha256:now"}).Return([]container.UpdateProgress{
		{Status: container.UpdateRecreating},
		{Status: container.UpdateVerifying},
		{Status: container.UpdateDone, Result: &rollbackResult},
	}, nil)
	mockService.On("RollbackContainer", mock.Anything, mock.Anything, container.RollbackOptions{ExpectedFromDigest: "sha256:moved"}).Return([]container.UpdateProgress{
		{Status: container.UpdateError, Error: "container no longer runs the expected image"},
	}, container.ErrDigestMismatch)

	mockService.On("StreamLogs", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		events := args.Get(4).(chan<- *container.LogEvent)
		for _, e := range streamedLogEvents {
			events <- e
		}
	})

	server, _ := NewServer(mockService, certs, "test", &mockNotificationHandler{})
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

	c, _ := rpc.FindContainer(context.Background(), "123456", container.ContainerLabels{})

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

// The swap's statuses come back from the agent as they are.
func TestUpdateContainerCarriesStatuses(t *testing.T) {
	rpc, err := NewClient("passthrough://bufnet", certs, grpc.WithContextDialer(bufDialer))
	require.NoError(t, err)

	progress := make(chan container.UpdateProgress, 10)
	updated, err := rpc.UpdateContainer(context.Background(), "123456", progress)
	require.NoError(t, err)
	assert.False(t, updated, "a rolled back update did not update anything")

	var got []container.UpdateProgress
	for p := range progress {
		got = append(got, p)
	}
	assert.Equal(t, []container.UpdateProgress{
		{Status: container.UpdateRecreating},
		{Status: container.UpdateVerifying},
		{Status: container.UpdateRolledBack, Error: "replacement is unhealthy", Result: &rolledBackResult},
	}, got, "the result travels back, so the server can record the update")
}

var rolledBackResult = container.UpdateResult{
	OldID:        "abc000000000",
	NewID:        "abc000000000",
	FromImageID:  "sha256:old",
	ToImageID:    "sha256:new",
	FromDigest:   "nginx@sha256:aaa",
	ToDigest:     "nginx@sha256:bbb",
	OldStartedAt: time.Date(2026, 9, 1, 3, 0, 0, 0, time.UTC),
	RolledBack:   true,
}

var rollbackResult = container.UpdateResult{OldID: "abc000000000", NewID: "def000000000", FromImageID: "sha256:new", ToImageID: "sha256:old"}

func TestRollbackContainerCarriesOptionsAndStatuses(t *testing.T) {
	rpc, err := NewClient("passthrough://bufnet", certs, grpc.WithContextDialer(bufDialer))
	require.NoError(t, err)

	progress := make(chan container.UpdateProgress, 10)
	err = rpc.RollbackContainer(context.Background(), "123456", container.RollbackOptions{ExpectedFromDigest: "sha256:now"}, progress)
	require.NoError(t, err)

	var got []container.UpdateProgress
	for p := range progress {
		got = append(got, p)
	}
	assert.Equal(t, []container.UpdateProgress{
		{Status: container.UpdateRecreating},
		{Status: container.UpdateVerifying},
		{Status: container.UpdateDone, Result: &rollbackResult},
	}, got)
}

// A refusal on the agent reaches the caller as an error and an error status.
func TestRollbackContainerReturnsAgentError(t *testing.T) {
	rpc, err := NewClient("passthrough://bufnet", certs, grpc.WithContextDialer(bufDialer))
	require.NoError(t, err)

	progress := make(chan container.UpdateProgress, 10)
	err = rpc.RollbackContainer(context.Background(), "123456", container.RollbackOptions{ExpectedFromDigest: "sha256:moved"}, progress)
	require.ErrorContains(t, err, "expected image")

	var got []container.UpdateProgress
	for p := range progress {
		got = append(got, p)
	}
	assert.Equal(t, []container.UpdateProgress{{Status: container.UpdateError, Error: "container no longer runs the expected image"}}, got)
}

// oldAgent is an agent that predates rollbacks: the RPC fails at the first
// Recv with Unimplemented.
type oldAgent struct {
	pb.AgentServiceClient
}

type unimplementedStream struct {
	grpc.ServerStreamingClient[pb.UpdateContainerProgress]
}

func (unimplementedStream) Recv() (*pb.UpdateContainerProgress, error) {
	return nil, status.Error(codes.Unimplemented, "unknown method RollbackContainer for service protobuf.AgentService")
}

func (oldAgent) RollbackContainer(context.Context, *pb.RollbackContainerRequest, ...grpc.CallOption) (grpc.ServerStreamingClient[pb.UpdateContainerProgress], error) {
	return unimplementedStream{}, nil
}

func TestRollbackContainerOnOldAgent(t *testing.T) {
	rpc := &Client{client: oldAgent{}, endpoint: "10.0.0.5:7007", nameOverride: "nas"}
	progress := make(chan container.UpdateProgress, 1)
	err := rpc.RollbackContainer(context.Background(), "123456", container.RollbackOptions{}, progress)
	require.EqualError(t, err, "agent on host nas is too old to roll back; upgrade it")
	_, open := <-progress
	assert.False(t, open)
}

var streamedLogEvents = []*container.LogEvent{
	{Id: 1, Type: container.LogTypeSingle, Message: "2026-09-13T22:28:56Z INF ready", RawMessage: "2026-09-13T22:28:56Z INF ready", Timestamp: 1789424936000, Level: "info", Stream: "stdout", TimestampPrefix: 21},
	{Id: 2, Type: container.LogTypeGroup, Message: []container.LogFragment{
		{Message: "2026-09-13T22:28:56Z ERR boom", TimestampPrefix: 21},
		{Message: "  at main.go:1"},
	}, Timestamp: 1789424936000, Level: "error", Stream: "stderr"},
}

func TestStreamContainerLogsKeepsTimestampPrefix(t *testing.T) {
	rpc, err := NewClient("passthrough://bufnet", certs, grpc.WithContextDialer(bufDialer))
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	events := make(chan *container.LogEvent, len(streamedLogEvents))
	go rpc.StreamContainerLogs(ctx, "123456", time.Time{}, container.STDALL, events)

	for _, want := range streamedLogEvents {
		select {
		case got := <-events:
			assert.Equal(t, want.Message, got.Message)
			assert.Equal(t, want.TimestampPrefix, got.TimestampPrefix)
		case <-ctx.Done():
			t.Fatal("timed out waiting for log event")
		}
	}
}

func TestHostWithAgentMetadata(t *testing.T) {
	rpc, err := NewClient("passthrough://bufnet|Web-1|Production", certs, grpc.WithContextDialer(bufDialer))
	if err != nil {
		t.Fatal(err)
	}

	host, err := rpc.Host(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, "passthrough://bufnet", host.Endpoint)
	assert.Equal(t, "Web-1", host.Name)
	assert.Equal(t, "Production", host.Group)
}

func TestHostWithAgentGroupAndDefaultName(t *testing.T) {
	rpc, err := NewClient("passthrough://bufnet||Production", certs, grpc.WithContextDialer(bufDialer))
	if err != nil {
		t.Fatal(err)
	}

	host, err := rpc.Host(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, "passthrough://bufnet", host.Endpoint)
	assert.Equal(t, "local", host.Name)
	assert.Equal(t, "Production", host.Group)
}

func TestNewClientRejectsInvalidAgentEndpoint(t *testing.T) {
	_, err := NewClient("passthrough://bufnet|Web-1|Production|extra", certs, grpc.WithContextDialer(bufDialer))

	assert.Error(t, err)
}

func TestParseEndpoint(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantAddr  string
		wantName  string
		wantGroup string
		wantErr   bool
	}{
		{name: "address only", input: "host:7007", wantAddr: "host:7007"},
		{name: "address and name", input: "host:7007|web-1", wantAddr: "host:7007", wantName: "web-1"},
		{name: "address, name, group", input: "host:7007|web-1|prod", wantAddr: "host:7007", wantName: "web-1", wantGroup: "prod"},
		{name: "trailing empty group", input: "host:7007|web-1|", wantAddr: "host:7007", wantName: "web-1"},
		{name: "empty name with group", input: "host:7007||prod", wantAddr: "host:7007", wantGroup: "prod"},
		{name: "empty address rejected", input: "|web-1|prod", wantErr: true},
		{name: "too many segments rejected", input: "host:7007|web-1|prod|extra", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr, name, group, err := ParseEndpoint(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.wantAddr, addr)
			assert.Equal(t, tt.wantName, name)
			assert.Equal(t, tt.wantGroup, group)
		})
	}
}

func TestVerifyAgentCert(t *testing.T) {
	pool := func(c tls.Certificate) *x509.CertPool {
		p := x509.NewCertPool()
		leaf, err := x509.ParseCertificate(c.Certificate[0])
		assert.NoError(t, err)
		p.AddCert(leaf)
		return p
	}
	pair := func() tls.Certificate {
		p, err := agentcerts.Generate()
		assert.NoError(t, err)
		c, err := tls.X509KeyPair(p.Cert, p.Key)
		assert.NoError(t, err)
		return c
	}
	private, other := pair(), pair()

	assert.NoError(t, verifyAgentCert(certs.Certificate, pool(certs)), "the shared cert trusts itself")
	assert.NoError(t, verifyAgentCert(private.Certificate, pool(private)), "a private pair trusts itself")
	assert.Error(t, verifyAgentCert(other.Certificate, pool(private)), "an agent with another pair is refused")
	assert.Error(t, verifyAgentCert(certs.Certificate, pool(private)), "the public shared cert is refused by a private hub")
	assert.Error(t, verifyAgentCert(nil, pool(private)))
}

// A request that ends before the swap starts cancels the stream (a pull on a
// wedged daemon must not pin it); once detached, the stream outlives it.
func TestSwapStreamContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	streamCtx, _, stop := swapStreamContext(ctx)
	defer stop()
	cancel()
	select {
	case <-streamCtx.Done():
	case <-time.After(time.Second):
		t.Fatal("stream not cancelled with the request before the swap started")
	}

	ctx, cancel = context.WithCancel(context.Background())
	streamCtx, detach, stop2 := swapStreamContext(ctx)
	defer stop2()
	assert.True(t, detach())
	cancel()
	assert.NoError(t, streamCtx.Err(), "detached stream outlives the request")
}

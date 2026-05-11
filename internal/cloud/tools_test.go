package cloud

import (
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/container"
	container_support "github.com/amir20/dozzle/internal/support/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestAvailableTools_WithActionsEnabled(t *testing.T) {
	tools := AvailableTools(true)

	names := make([]string, len(tools))
	for i, tool := range tools {
		names[i] = tool.Name
	}

	assert.Contains(t, names, "list_hosts")
	assert.Contains(t, names, "find_containers")
	assert.Contains(t, names, "list_running_containers")
	assert.Contains(t, names, "list_all_containers")
	assert.Contains(t, names, "get_running_container_stats")
	assert.Contains(t, names, "fetch_container_logs")
	assert.Contains(t, names, "start_container")
	assert.Contains(t, names, "stop_container")
	assert.Contains(t, names, "restart_container")
	assert.Contains(t, names, "remove_container")
	assert.Contains(t, names, "list_notifications")
	assert.Contains(t, names, "create_log_notification")
	assert.Contains(t, names, "create_metric_notification")
	assert.Contains(t, names, "create_event_notification")
	assert.Len(t, tools, 17)
}

func TestAvailableTools_WithActionsDisabled(t *testing.T) {
	tools := AvailableTools(false)

	names := make([]string, len(tools))
	for i, tool := range tools {
		names[i] = tool.Name
	}

	assert.Contains(t, names, "list_hosts")
	assert.Contains(t, names, "find_containers")
	assert.Contains(t, names, "list_running_containers")
	assert.Contains(t, names, "list_all_containers")
	assert.Contains(t, names, "get_running_container_stats")
	assert.Contains(t, names, "fetch_container_logs")
	assert.Contains(t, names, "list_notifications")
	assert.Len(t, tools, 9)
}

func TestAvailableTools_ParametersAreValid(t *testing.T) {
	tools := AvailableTools(true)

	for _, tool := range tools {
		assert.NotEmpty(t, tool.Name)
		assert.NotEmpty(t, tool.Description)
		assert.NotEmpty(t, tool.ParametersJson)
	}
}

// MockHostService mocks the HostService interface for testing
type MockHostService struct {
	mock.Mock
}

func (m *MockHostService) ListAllContainers(labels container.ContainerLabels) ([]container.Container, []error) {
	args := m.Called(labels)
	containers := args.Get(0).([]container.Container)
	var errs []error
	if args.Get(1) != nil {
		errs = args.Get(1).([]error)
	}
	return containers, errs
}

func (m *MockHostService) FindContainer(host string, id string, labels container.ContainerLabels) (*container_support.ContainerService, error) {
	args := m.Called(host, id, labels)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*container_support.ContainerService), args.Error(1)
}

func (m *MockHostService) Hosts() []container.Host {
	args := m.Called()
	return args.Get(0).([]container.Host)
}

type MockClientService struct {
	mock.Mock
}

func (m *MockClientService) FindContainer(_ context.Context, _ string, _ container.ContainerLabels) (container.Container, error) {
	return container.Container{}, nil
}
func (m *MockClientService) ListContainers(_ context.Context, _ container.ContainerLabels) ([]container.Container, error) {
	return nil, nil
}
func (m *MockClientService) Host(_ context.Context) (container.Host, error) {
	return container.Host{}, nil
}
func (m *MockClientService) ContainerAction(ctx context.Context, c container.Container, action container.ContainerAction) error {
	args := m.Called(ctx, c, action)
	return args.Error(0)
}
func (m *MockClientService) LogsBetweenDates(_ context.Context, _ container.Container, _ time.Time, _ time.Time, _ container.StdType) (<-chan *container.LogEvent, error) {
	return nil, nil
}
func (m *MockClientService) RawLogs(_ context.Context, _ container.Container, _ time.Time, _ time.Time, _ container.StdType) (io.ReadCloser, error) {
	return nil, nil
}
func (m *MockClientService) SubscribeStats(_ context.Context, _ chan<- container.ContainerStat) {}
func (m *MockClientService) SubscribeEvents(_ context.Context, _ chan<- container.ContainerEvent) {
}
func (m *MockClientService) SubscribeContainersStarted(_ context.Context, _ chan<- container.Container) {
}
func (m *MockClientService) StreamLogs(_ context.Context, _ container.Container, _ time.Time, _ container.StdType, _ chan<- *container.LogEvent) error {
	return nil
}
func (m *MockClientService) Attach(_ context.Context, _ container.Container, _ container.ExecEventReader, _ io.Writer) error {
	return nil
}
func (m *MockClientService) Exec(_ context.Context, _ container.Container, _ []string, _ container.ExecEventReader, _ io.Writer) error {
	return nil
}

func (m *MockClientService) UpdateContainer(_ context.Context, _ container.Container, progressCh chan<- container.UpdateProgress) (bool, error) {
	close(progressCh)
	return false, nil
}

func TestExecuteTool_ListRunningContainers(t *testing.T) {
	mockHost := &MockHostService{}
	mockHost.On("ListAllContainers", container.ContainerLabels(nil)).Return([]container.Container{
		{ID: "abc123", Name: "nginx", Image: "nginx:latest", State: "running", Host: "local"},
		{ID: "def456", Name: "redis", Image: "redis:7", State: "running", Host: "local"},
		{ID: "ghi789", Name: "stopped", Image: "alpine:latest", State: "exited", Host: "local"},
	}, nil)
	mockHost.On("Hosts").Return([]container.Host{{ID: "local", Name: "my-server"}})

	resp := ExecuteTool(context.Background(), "list_running_containers", "", ToolDeps{HostService: mockHost})
	assert.True(t, resp.Success)

	result := resp.GetListContainers()
	assert.NotNil(t, result)
	assert.Len(t, result.Containers, 2)
	assert.Equal(t, "abc123", result.Containers[0].Id)
	assert.Equal(t, "nginx", result.Containers[0].Name)
	assert.Equal(t, "my-server", result.Containers[0].HostName)
}

func TestExecuteTool_ListAllContainers(t *testing.T) {
	mockHost := &MockHostService{}
	mockHost.On("ListAllContainers", container.ContainerLabels(nil)).Return([]container.Container{
		{ID: "abc123", Name: "nginx", Image: "nginx:latest", State: "running", Host: "local"},
		{ID: "def456", Name: "redis", Image: "redis:7", State: "exited", Host: "local"},
	}, nil)
	mockHost.On("Hosts").Return([]container.Host{{ID: "local", Name: "my-server"}})

	resp := ExecuteTool(context.Background(), "list_all_containers", "", ToolDeps{HostService: mockHost})
	assert.True(t, resp.Success)

	result := resp.GetListContainers()
	assert.NotNil(t, result)
	assert.Len(t, result.Containers, 2)
	assert.Equal(t, "abc123", result.Containers[0].Id)
	assert.Equal(t, "def456", result.Containers[1].Id)
	assert.Equal(t, "my-server", result.Containers[0].HostName)
}

func TestExecuteTool_RestartContainer(t *testing.T) {
	mockClient := &MockClientService{}
	mockClient.On("ContainerAction", mock.Anything, mock.Anything, container.Restart).Return(nil)

	cs := container_support.NewContainerService(mockClient, container.Container{ID: "abc123"})

	mockHost := &MockHostService{}
	mockHost.On("FindContainer", "local", "abc123", container.ContainerLabels(nil)).Return(cs, nil)

	argsJSON := `{"container_id": "abc123", "host_id": "local"}`
	resp := ExecuteTool(context.Background(), "restart_container", argsJSON, ToolDeps{HostService: mockHost, EnableActions: true})
	assert.True(t, resp.Success)

	action := resp.GetAction()
	assert.NotNil(t, action)
	assert.True(t, action.Success)
	assert.Equal(t, "abc123", action.ContainerId)

	mockClient.AssertCalled(t, "ContainerAction", mock.Anything, mock.Anything, container.Restart)
}

func TestExecuteTool_RemoveContainer(t *testing.T) {
	mockClient := &MockClientService{}
	mockClient.On("ContainerAction", mock.Anything, mock.Anything, container.Remove).Return(nil)

	cs := container_support.NewContainerService(mockClient, container.Container{ID: "abc123"})

	mockHost := &MockHostService{}
	mockHost.On("FindContainer", "local", "abc123", container.ContainerLabels(nil)).Return(cs, nil)

	argsJSON := `{"container_id": "abc123", "host_id": "local"}`
	resp := ExecuteTool(context.Background(), "remove_container", argsJSON, ToolDeps{HostService: mockHost, EnableActions: true})
	assert.True(t, resp.Success)

	action := resp.GetAction()
	assert.NotNil(t, action)
	assert.True(t, action.Success)
	assert.Equal(t, "abc123", action.ContainerId)
	assert.Equal(t, "remove", action.Action)

	mockClient.AssertCalled(t, "ContainerAction", mock.Anything, mock.Anything, container.Remove)
}

func TestExecuteTool_RestartContainer_ActionsDisabled(t *testing.T) {
	mockHost := &MockHostService{}

	argsJSON := `{"container_id": "abc123"}`
	resp := ExecuteTool(context.Background(), "restart_container", argsJSON, ToolDeps{HostService: mockHost})
	assert.False(t, resp.Success)
	assert.Contains(t, resp.Error, "container actions are not enabled")
}

func TestExecuteTool_ListRunningContainers_PartialHostError(t *testing.T) {
	mockHost := &MockHostService{}
	mockHost.On("ListAllContainers", container.ContainerLabels(nil)).Return(
		[]container.Container{
			{ID: "abc123", Name: "nginx", Image: "nginx:latest", State: "running", Host: "local"},
		},
		[]error{fmt.Errorf("host2 unreachable")},
	)
	mockHost.On("Hosts").Return([]container.Host{{ID: "local", Name: "my-server"}})

	resp := ExecuteTool(context.Background(), "list_running_containers", "", ToolDeps{HostService: mockHost})
	assert.True(t, resp.Success)

	result := resp.GetListContainers()
	assert.NotNil(t, result)
	assert.Len(t, result.Containers, 1)
}

func TestExecuteTool_ListHosts(t *testing.T) {
	mockHost := &MockHostService{}
	mockHost.On("Hosts").Return([]container.Host{
		{ID: "host1", Name: "server-1", NCPU: 4, MemTotal: 8589934592, DockerVersion: "24.0.7", Available: true},
		{ID: "host2", Name: "server-2", NCPU: 8, MemTotal: 17179869184, DockerVersion: "25.0.1", Available: false},
	})

	resp := ExecuteTool(context.Background(), "list_hosts", "", ToolDeps{HostService: mockHost})
	assert.True(t, resp.Success)

	result := resp.GetListHosts()
	assert.NotNil(t, result)
	assert.Len(t, result.Hosts, 2)
	assert.Equal(t, "host1", result.Hosts[0].Id)
	assert.Equal(t, "server-1", result.Hosts[0].Name)
	assert.Equal(t, true, result.Hosts[0].Available)
	assert.Equal(t, false, result.Hosts[1].Available)
}

func TestExecuteTool_UnknownTool(t *testing.T) {
	mockHost := &MockHostService{}

	resp := ExecuteTool(context.Background(), "unknown_tool", "", ToolDeps{HostService: mockHost})
	assert.False(t, resp.Success)
	assert.Contains(t, resp.Error, "unknown tool")
}

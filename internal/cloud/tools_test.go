package cloud

import (
	"context"
	"encoding/json"
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

	assert.Contains(t, names, "find_containers")
	assert.Contains(t, names, "start_container")
	assert.Contains(t, names, "stop_container")
	assert.Contains(t, names, "restart_container")
	assert.Len(t, tools, 4)
}

func TestAvailableTools_WithActionsDisabled(t *testing.T) {
	tools := AvailableTools(false)

	names := make([]string, len(tools))
	for i, tool := range tools {
		names[i] = tool.Name
	}

	assert.Contains(t, names, "find_containers")
	assert.Len(t, tools, 1)
}

func TestAvailableTools_ParametersAreValid(t *testing.T) {
	tools := AvailableTools(true)

	for _, tool := range tools {
		assert.NotEmpty(t, tool.Name)
		assert.NotEmpty(t, tool.Description)
		assert.NotNil(t, tool.Parameters)
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

func TestExecuteTool_ListContainers(t *testing.T) {
	mockHost := &MockHostService{}
	mockHost.On("ListAllContainers", container.ContainerLabels(nil)).Return([]container.Container{
		{ID: "abc123", Name: "nginx", Image: "nginx:latest", State: "running", Host: "local"},
		{ID: "def456", Name: "redis", Image: "redis:7", State: "running", Host: "local"},
	}, nil)

	result, err := ExecuteTool(context.Background(), "find_containers", "", false, mockHost, nil)
	assert.NoError(t, err)

	var containers []map[string]any
	err = json.Unmarshal([]byte(result), &containers)
	assert.NoError(t, err)
	assert.Len(t, containers, 2)
	assert.Equal(t, "abc123", containers[0]["id"])
	assert.Equal(t, "nginx", containers[0]["name"])
}

func TestExecuteTool_RestartContainer(t *testing.T) {
	mockClient := &MockClientService{}
	mockClient.On("ContainerAction", mock.Anything, mock.Anything, container.Restart).Return(nil)

	cs := container_support.NewContainerService(mockClient, container.Container{ID: "abc123"})

	mockHost := &MockHostService{}
	mockHost.On("FindContainer", "", "abc123", container.ContainerLabels(nil)).Return(cs, nil)

	argsJSON := `{"container_id": "abc123"}`
	result, err := ExecuteTool(context.Background(), "restart_container", argsJSON, true, mockHost, nil)
	assert.NoError(t, err)
	assert.Contains(t, result, "success")

	mockClient.AssertCalled(t, "ContainerAction", mock.Anything, mock.Anything, container.Restart)
}

func TestExecuteTool_RestartContainer_ActionsDisabled(t *testing.T) {
	mockHost := &MockHostService{}

	argsJSON := `{"container_id": "abc123"}`
	_, err := ExecuteTool(context.Background(), "restart_container", argsJSON, false, mockHost, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "container actions are not enabled")
}

func TestExecuteTool_ListContainers_PartialHostError(t *testing.T) {
	mockHost := &MockHostService{}
	mockHost.On("ListAllContainers", container.ContainerLabels(nil)).Return(
		[]container.Container{
			{ID: "abc123", Name: "nginx", Image: "nginx:latest", State: "running", Host: "local"},
		},
		[]error{fmt.Errorf("host2 unreachable")},
	)

	result, err := ExecuteTool(context.Background(), "find_containers", "", false, mockHost, nil)
	assert.NoError(t, err)

	var containers []map[string]any
	err = json.Unmarshal([]byte(result), &containers)
	assert.NoError(t, err)
	assert.Len(t, containers, 1)
}

func TestExecuteTool_UnknownTool(t *testing.T) {
	mockHost := &MockHostService{}

	_, err := ExecuteTool(context.Background(), "unknown_tool", "", false, mockHost, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown tool")
}

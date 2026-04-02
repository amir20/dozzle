package cloud

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/amir20/dozzle/internal/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestAvailableTools_WithActionsEnabled(t *testing.T) {
	tools := AvailableTools(true)

	names := make([]string, len(tools))
	for i, tool := range tools {
		names[i] = tool.Name
	}

	assert.Contains(t, names, "list_containers")
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

	assert.Contains(t, names, "list_containers")
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
	return args.Get(0).([]container.Container), nil
}

func (m *MockHostService) FindContainer(host string, id string, labels container.ContainerLabels) (ContainerActioner, error) {
	args := m.Called(host, id, labels)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(ContainerActioner), args.Error(1)
}

type MockContainerActioner struct {
	mock.Mock
}

func (m *MockContainerActioner) Action(ctx context.Context, action container.ContainerAction) error {
	args := m.Called(ctx, action)
	return args.Error(0)
}

func TestExecuteTool_ListContainers(t *testing.T) {
	mockHost := &MockHostService{}
	mockHost.On("ListAllContainers", container.ContainerLabels(nil)).Return([]container.Container{
		{ID: "abc123", Name: "nginx", Image: "nginx:latest", State: "running", Host: "local"},
		{ID: "def456", Name: "redis", Image: "redis:7", State: "running", Host: "local"},
	}, nil)

	result, err := ExecuteTool(context.Background(), "list_containers", "", mockHost, nil)
	assert.NoError(t, err)

	var containers []map[string]any
	err = json.Unmarshal([]byte(result), &containers)
	assert.NoError(t, err)
	assert.Len(t, containers, 2)
	assert.Equal(t, "abc123", containers[0]["id"])
	assert.Equal(t, "nginx", containers[0]["name"])
}

func TestExecuteTool_RestartContainer(t *testing.T) {
	mockActioner := &MockContainerActioner{}
	mockActioner.On("Action", mock.Anything, container.Restart).Return(nil)

	mockHost := &MockHostService{}
	mockHost.On("FindContainer", "", "abc123", container.ContainerLabels(nil)).Return(mockActioner, nil)

	argsJSON := `{"container_id": "abc123"}`
	result, err := ExecuteTool(context.Background(), "restart_container", argsJSON, mockHost, nil)
	assert.NoError(t, err)
	assert.Contains(t, result, "success")

	mockActioner.AssertCalled(t, "Action", mock.Anything, container.Restart)
}

func TestExecuteTool_UnknownTool(t *testing.T) {
	mockHost := &MockHostService{}

	_, err := ExecuteTool(context.Background(), "unknown_tool", "", mockHost, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown tool")
}

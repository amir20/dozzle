package cloud

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync/atomic"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/auth"
	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/imagecheck"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestAvailableTools_WithActionsEnabled(t *testing.T) {
	tools := AvailableTools(ToolDeps{EnableActions: true})

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
	assert.Contains(t, names, "rollback_container")
	assert.Len(t, tools, 19)
}

func TestAvailableTools_WithActionsDisabled(t *testing.T) {
	tools := AvailableTools(ToolDeps{})

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
	assert.Len(t, tools, 10)
}

func TestAvailableTools_ParametersAreValid(t *testing.T) {
	tools := AvailableTools(ToolDeps{EnableActions: true})

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

func (m *MockHostService) FindContainer(host string, id string, labels container.ContainerLabels) (*container.ContainerService, error) {
	args := m.Called(host, id, labels)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*container.ContainerService), args.Error(1)
}

func (m *MockHostService) Hosts() []container.Host {
	args := m.Called()
	return args.Get(0).([]container.Host)
}

// withResolver wires up the ListAllContainers + Hosts mocks that
// resolveContainerRef needs, so container-scoped tool tests can drive the
// resolver. Derives the host list from the containers' Host fields.
func withResolver(m *MockHostService, containers ...container.Container) {
	seen := map[string]bool{}
	hosts := []container.Host{}
	for _, c := range containers {
		if c.Host != "" && !seen[c.Host] {
			seen[c.Host] = true
			hosts = append(hosts, container.Host{ID: c.Host, Name: c.Host})
		}
	}
	m.On("ListAllContainers", container.ContainerLabels(nil)).Return(containers, nil).Maybe()
	m.On("Hosts").Return(hosts).Maybe()
}

type MockClientService struct {
	mock.Mock
	// imageResults answers CheckImageUpdate by container ID; anything absent
	// is up to date.
	imageResults map[string]imagecheck.Result
	// imageForced records the force flag of the last CheckImageUpdate.
	imageForced atomic.Bool
	// updateProgress is what UpdateContainer reports, and updateErr what it returns.
	updateProgress []container.UpdateProgress
	updateErr      error
	// rollbackOpts records the options of the last RollbackContainer, and
	// rollbackErr is what it returns.
	rollbackOpts container.RollbackOptions
	rollbackErr  error
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
func (m *MockClientService) LogHistogram(context.Context, container.Container, time.Time, time.Time, time.Duration) (container.LogHistogram, error) {
	return container.LogHistogram{}, nil
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

func (m *MockClientService) CheckImageUpdate(_ context.Context, c container.Container, force bool) (imagecheck.Result, error) {
	m.imageForced.Store(force)
	if r, ok := m.imageResults[c.ID]; ok {
		return r, nil
	}
	return imagecheck.Result{Image: c.Image, Status: imagecheck.StatusUpToDate}, nil
}

func (m *MockClientService) UpdateContainer(_ context.Context, _ container.Container, progressCh chan<- container.UpdateProgress) (bool, error) {
	defer close(progressCh)
	for _, p := range m.updateProgress {
		progressCh <- p
	}
	return false, m.updateErr
}

func (m *MockClientService) RollbackContainer(_ context.Context, _ container.Container, opts container.RollbackOptions, progressCh chan<- container.UpdateProgress) error {
	defer close(progressCh)
	m.rollbackOpts = opts
	if m.rollbackErr != nil {
		progressCh <- container.UpdateProgress{Status: container.UpdateError, Error: m.rollbackErr.Error()}
		return m.rollbackErr
	}
	progressCh <- container.UpdateProgress{Status: container.UpdateDone, Result: &container.UpdateResult{OldID: "abc123", NewID: "rolled123456", FromImageID: "sha256:new", ToImageID: "sha256:old"}}
	return nil
}

func TestExecuteTool_RollbackContainer(t *testing.T) {
	mockClient := &MockClientService{}
	mockHost := &MockHostService{}
	c := container.Container{ID: "abc123", Name: "immich", Host: "rollback-host", State: "running"}
	withResolver(mockHost, c)
	mockHost.On("FindContainer", "rollback-host", "abc123", container.ContainerLabels(nil)).Return(container.NewContainerService(mockClient, c), nil)

	deps := ToolDeps{HostService: mockHost, EnableActions: true}

	// The confirmation is in code: no digest, no rollback.
	resp := ExecuteTool(context.Background(), toolRollbackContainer, `{"container_id":"abc123"}`, deps)
	assert.False(t, resp.Success)
	assert.Contains(t, resp.Error, "expected_from_digest")
	assert.Empty(t, mockClient.rollbackOpts.ExpectedFromDigest, "nothing reached the container")

	resp = ExecuteTool(context.Background(), toolRollbackContainer, `{"container_id":"abc123","expected_from_digest":"ghcr.io/immich@sha256:new"}`, deps)
	require.True(t, resp.Success, resp.Error)
	assert.Equal(t, container.RollbackOptions{ExpectedFromDigest: "ghcr.io/immich@sha256:new"}, mockClient.rollbackOpts)
	action := resp.GetAction()
	require.NotNil(t, action)
	assert.Equal(t, "rollback", action.Action)
	assert.Equal(t, "abc123", action.ContainerId)
	assert.Contains(t, action.Message, "immich")

	record, ok := container.Updates.Latest("rollback-host", "rolled123456")
	require.True(t, ok, "the rollback is recorded, for the log marker and Dozzle Cloud")
	assert.Equal(t, container.UpdateSourceRollback, record.Source)
	assert.Equal(t, "immich", record.Name)

	// A container that moved on since is refused.
	mockClient.rollbackErr = fmt.Errorf("%w: it runs ghcr.io/immich@sha256:newer", container.ErrDigestMismatch)
	resp = ExecuteTool(context.Background(), toolRollbackContainer, `{"container_id":"abc123","expected_from_digest":"ghcr.io/immich@sha256:new"}`, deps)
	assert.False(t, resp.Success)
	assert.Contains(t, resp.Error, "no longer runs the expected image")
}

func TestRollbackContainerNeedsActions(t *testing.T) {
	assert.Equal(t, auth.Actions, mutatingTools[toolRollbackContainer])
	assert.Error(t, APIKeyPrincipal(nil).mayCall(toolRollbackContainer, false), "gated behind --enable-actions")
	assert.Error(t, InstancePrincipal(nil).mayCall(toolRollbackContainer, true), "never background work")
	assert.Error(t, Principal{Kind: PrincipalUser}.mayCall(toolRollbackContainer, true), "a user without the actions role")
	assert.NoError(t, Principal{Kind: PrincipalUser, Roles: auth.Actions}.mayCall(toolRollbackContainer, true))

	var names []string
	for _, tool := range AvailableTools(ToolDeps{EnableActions: true}) {
		names = append(names, tool.Name)
	}
	assert.Contains(t, names, toolRollbackContainer)
	for _, tool := range AvailableTools(ToolDeps{}) {
		assert.NotEqual(t, toolRollbackContainer, tool.Name, "hidden without actions")
	}
}

func TestExecuteTool_UpdateContainerRolledBackFails(t *testing.T) {
	mockClient := &MockClientService{}
	mockHost := &MockHostService{}
	c := container.Container{ID: "abc123", Name: "nginx", Host: "local", State: "running"}
	withResolver(mockHost, c)
	mockHost.On("FindContainer", "local", "abc123", container.ContainerLabels(nil)).Return(container.NewContainerService(mockClient, c), nil)

	deps := ToolDeps{HostService: mockHost, EnableActions: true}
	resp := ExecuteTool(context.Background(), "update_container", `{"container_id":"abc123"}`, deps)
	assert.True(t, resp.Success, resp.Error)

	// A rolled back update is a failure, and says so.
	mockClient.updateProgress = []container.UpdateProgress{{Status: container.UpdateRolledBack, Error: "replacement is unhealthy"}}
	mockClient.updateErr = errors.New("update rolled back: replacement is unhealthy")
	resp = ExecuteTool(context.Background(), "update_container", `{"container_id":"abc123"}`, deps)
	assert.False(t, resp.Success)
	assert.Contains(t, resp.Error, "rolled back")
}

// A stopped container is refused before the host is asked, for an update and
// a rollback alike.
func TestExecuteTool_UpdateAndRollbackRefuseStoppedContainer(t *testing.T) {
	mockClient := &MockClientService{updateErr: errors.New("the host was asked")}
	mockHost := &MockHostService{}
	c := container.Container{ID: "abc123", Name: "backup", Host: "local", State: "exited"}
	withResolver(mockHost, c)
	mockHost.On("FindContainer", "local", "abc123", container.ContainerLabels(nil)).Return(container.NewContainerService(mockClient, c), nil)

	deps := ToolDeps{HostService: mockHost, EnableActions: true}
	resp := ExecuteTool(context.Background(), toolUpdateContainer, `{"container_id":"abc123"}`, deps)
	assert.False(t, resp.Success)
	assert.Contains(t, resp.Error, "Start the container first")

	resp = ExecuteTool(context.Background(), toolRollbackContainer, `{"container_id":"abc123","expected_from_digest":"backup@sha256:new"}`, deps)
	assert.False(t, resp.Success)
	assert.Contains(t, resp.Error, "Start the container first")
	assert.Empty(t, mockClient.rollbackOpts.ExpectedFromDigest, "nothing reached the container")
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

	cs := container.NewContainerService(mockClient, container.Container{ID: "abc123", Name: "nginx", Host: "local"})

	mockHost := &MockHostService{}
	withResolver(mockHost, container.Container{ID: "abc123", Name: "nginx", Host: "local"})
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

	cs := container.NewContainerService(mockClient, container.Container{ID: "abc123", Name: "nginx", Host: "local"})

	mockHost := &MockHostService{}
	withResolver(mockHost, container.Container{ID: "abc123", Name: "nginx", Host: "local"})
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

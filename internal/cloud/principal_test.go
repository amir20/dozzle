package cloud

import (
	"context"
	"testing"

	"github.com/amir20/dozzle/internal/auth"
	"github.com/amir20/dozzle/internal/container"
	pb "github.com/amir20/dozzle/proto/cloud"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func toolNames(tools []*pb.ToolDefinition) map[string]struct{} {
	names := make(map[string]struct{}, len(tools))
	for _, t := range tools {
		names[t.Name] = struct{}{}
	}
	return names
}

// The zero principal is what every tool call meant before principals existed,
// so nothing about the Telegram and Discord path may change.
func Test_APIKeyPrincipal_keepsTodaysGating(t *testing.T) {
	p := Principal{}
	assert.Equal(t, PrincipalAPIKey, p.Kind)

	require.NoError(t, p.mayCall(toolStartContainer, true))
	require.NoError(t, p.mayCall(toolCreateLogNotification, true))
	require.Error(t, p.mayCall(toolStartContainer, false))
	require.NoError(t, p.mayCall(toolFetchContainerLogs, false))
	// No user means no roles to consult, so a role-gated read stays open.
	require.NoError(t, p.mayCall(toolListNotifications, false))
}

// Background work reads. It never acts, even on an instance that enabled
// actions for the people asking.
func Test_InstancePrincipal_isReadOnly(t *testing.T) {
	p := InstancePrincipal(nil)

	require.NoError(t, p.mayCall(toolFetchContainerLogs, true))
	require.NoError(t, p.mayCall(toolListAllContainers, true))
	require.Error(t, p.mayCall(toolStartContainer, true))
	require.Error(t, p.mayCall(toolCreateLogNotification, true))
}

func Test_UserPrincipal_needsTheRole(t *testing.T) {
	viewer := UserPrincipal(nil, auth.Download)
	operator := UserPrincipal(nil, auth.Actions)
	notifier := UserPrincipal(nil, auth.Notifications)

	// Reads that carry no role are open to anyone signed in.
	require.NoError(t, viewer.mayCall(toolFetchContainerLogs, true))
	require.NoError(t, viewer.mayCall(toolListAllContainers, true))

	require.Error(t, viewer.mayCall(toolStartContainer, true))
	require.NoError(t, operator.mayCall(toolStartContainer, true))
	// The role is not a way around --enable-actions.
	require.Error(t, operator.mayCall(toolStartContainer, false))

	require.Error(t, operator.mayCall(toolCreateLogNotification, true))
	require.NoError(t, notifier.mayCall(toolCreateLogNotification, true))

	// Notification rules are instance wide, so listing them needs the role
	// even though it only reads.
	require.Error(t, viewer.mayCall(toolListNotifications, true))
	require.NoError(t, notifier.mayCall(toolListNotifications, true))
}

// The model should never be offered a tool it will then be refused for.
func Test_AvailableTools_followsThePrincipal(t *testing.T) {
	all := toolNames(AvailableTools(true, Principal{}))
	assert.Contains(t, all, toolStartContainer)
	assert.Contains(t, all, toolCreateLogNotification)

	viewer := toolNames(AvailableTools(true, UserPrincipal(nil, auth.Download)))
	assert.NotContains(t, viewer, toolStartContainer)
	assert.NotContains(t, viewer, toolCreateLogNotification)
	assert.NotContains(t, viewer, toolListNotifications)
	assert.Contains(t, viewer, toolFetchContainerLogs)

	operator := toolNames(AvailableTools(true, UserPrincipal(nil, auth.Actions)))
	assert.Contains(t, operator, toolStartContainer)
	assert.NotContains(t, operator, toolCreateLogNotification)

	background := toolNames(AvailableTools(true, InstancePrincipal(nil)))
	assert.NotContains(t, background, toolStartContainer)
	assert.Contains(t, background, toolListAllContainers)
}

// executeTool must refuse before it reaches the host service, not after.
func Test_executeTool_refusesBeforeTouchingDocker(t *testing.T) {
	host := new(MockHostService)
	deps := ToolDeps{
		EnableActions: true,
		HostService:   host,
		Principal:     UserPrincipal(nil, auth.Download),
	}

	_, err := executeTool(context.Background(), toolStartContainer, `{"container":"whatever"}`, deps)
	require.Error(t, err)
	host.AssertNotCalled(t, "FindContainer")
}

// The whole point of scoped(): a tool cannot ask for containers outside the
// principal's filter, because it never gets to choose the labels.
func Test_scoped_usesThePrincipalsLabels(t *testing.T) {
	labels := container.ContainerLabels{"env": {"dev"}}
	host := new(MockHostService)
	host.On("ListAllContainers", labels).Return([]container.Container{}, nil)

	deps := ToolDeps{HostService: host, Principal: UserPrincipal(labels, auth.All)}
	deps.scoped().ListAllContainers()

	host.AssertExpectations(t)
}

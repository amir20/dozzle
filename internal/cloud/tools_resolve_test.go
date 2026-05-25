package cloud

import (
	"context"
	"testing"

	"github.com/amir20/dozzle/internal/container"
	container_support "github.com/amir20/dozzle/internal/support/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// resolverDeps builds ToolDeps whose HostService resolves against the given
// containers. Hosts are derived from the containers' Host fields, optionally
// augmented with explicit hosts (so a host id and a human name can differ).
func resolverDeps(containers []container.Container, hosts ...container.Host) ToolDeps {
	m := &MockHostService{}
	m.On("ListAllContainers", container.ContainerLabels(nil)).Return(containers, nil).Maybe()
	if len(hosts) == 0 {
		seen := map[string]bool{}
		for _, c := range containers {
			if c.Host != "" && !seen[c.Host] {
				seen[c.Host] = true
				hosts = append(hosts, container.Host{ID: c.Host, Name: c.Host})
			}
		}
	}
	m.On("Hosts").Return(hosts).Maybe()
	return ToolDeps{HostService: m}
}

func TestResolveContainerRef_ByName(t *testing.T) {
	deps := resolverDeps([]container.Container{
		{ID: "abc123def456", Name: "nginx", Host: "local"},
		{ID: "fff999", Name: "redis", Host: "local"},
	})

	host, id, err := resolveContainerRef("nginx", "", deps)
	assert.NoError(t, err)
	assert.Equal(t, "local", host)
	assert.Equal(t, "abc123def456", id)
}

func TestResolveContainerRef_ByNameCaseInsensitive(t *testing.T) {
	deps := resolverDeps([]container.Container{
		{ID: "abc123def456", Name: "NginX", Host: "local"},
	})

	host, id, err := resolveContainerRef("nginx", "", deps)
	assert.NoError(t, err)
	assert.Equal(t, "local", host)
	assert.Equal(t, "abc123def456", id)
}

func TestResolveContainerRef_ByFullID(t *testing.T) {
	deps := resolverDeps([]container.Container{
		{ID: "abc123def456", Name: "nginx", Host: "local"},
	})

	host, id, err := resolveContainerRef("abc123def456", "", deps)
	assert.NoError(t, err)
	assert.Equal(t, "local", host)
	assert.Equal(t, "abc123def456", id)
}

func TestResolveContainerRef_ByShortIDPrefix(t *testing.T) {
	deps := resolverDeps([]container.Container{
		{ID: "abc123def4567890", Name: "nginx", Host: "local"},
	})

	// 12-char short id form
	host, id, err := resolveContainerRef("abc123def456", "", deps)
	assert.NoError(t, err)
	assert.Equal(t, "local", host)
	assert.Equal(t, "abc123def4567890", id)
}

func TestResolveContainerRef_BySubstring(t *testing.T) {
	deps := resolverDeps([]container.Container{
		{ID: "id1", Name: "my-app-frontend", Host: "local"},
		{ID: "id2", Name: "database", Host: "local"},
	})

	host, id, err := resolveContainerRef("frontend", "", deps)
	assert.NoError(t, err)
	assert.Equal(t, "local", host)
	assert.Equal(t, "id1", id)
}

func TestResolveContainerRef_ExactNameBeatsSubstring(t *testing.T) {
	// "api" matches "api" exactly and "api-gateway" as a substring. Exact wins,
	// so this is NOT ambiguous.
	deps := resolverDeps([]container.Container{
		{ID: "id1", Name: "api", Host: "local"},
		{ID: "id2", Name: "api-gateway", Host: "local"},
	})

	host, id, err := resolveContainerRef("api", "", deps)
	assert.NoError(t, err)
	assert.Equal(t, "local", host)
	assert.Equal(t, "id1", id)
}

func TestResolveContainerRef_AmbiguousNameAcrossHosts(t *testing.T) {
	// Same name on two different hosts, no host supplied → ambiguous, must list
	// candidates and must NOT silently pick one.
	deps := resolverDeps([]container.Container{
		{ID: "id1", Name: "nginx", Host: "host-a"},
		{ID: "id2", Name: "nginx", Host: "host-b"},
	})

	_, _, err := resolveContainerRef("nginx", "", deps)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "matches multiple containers")
	assert.Contains(t, err.Error(), "id1")
	assert.Contains(t, err.Error(), "id2")
	assert.Contains(t, err.Error(), "host-a")
	assert.Contains(t, err.Error(), "host-b")
	assert.Contains(t, err.Error(), "host_id")
}

func TestResolveContainerRef_AmbiguousSubstring(t *testing.T) {
	// Both candidates are on the same host, so host_id cannot disambiguate — the
	// hint must steer the LLM to the exact id / full name, not a useless retry
	// with host_id.
	deps := resolverDeps([]container.Container{
		{ID: "id1", Name: "app-frontend", Host: "local"},
		{ID: "id2", Name: "app-backend", Host: "local"},
	})

	_, _, err := resolveContainerRef("app", "", deps)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "matches multiple containers")
	assert.Contains(t, err.Error(), "exact container id or the full container name")
	assert.NotContains(t, err.Error(), "host_id")
}

func TestResolveContainerRef_HostDisambiguates(t *testing.T) {
	// Same name on two hosts; supplying the host id resolves cleanly.
	deps := resolverDeps([]container.Container{
		{ID: "id1", Name: "nginx", Host: "host-a"},
		{ID: "id2", Name: "nginx", Host: "host-b"},
	})

	host, id, err := resolveContainerRef("nginx", "host-b", deps)
	assert.NoError(t, err)
	assert.Equal(t, "host-b", host)
	assert.Equal(t, "id2", id)
}

func TestResolveContainerRef_HostByName(t *testing.T) {
	// Host referenced by its human name rather than its id.
	deps := resolverDeps(
		[]container.Container{
			{ID: "id1", Name: "nginx", Host: "h-a"},
			{ID: "id2", Name: "nginx", Host: "h-b"},
		},
		container.Host{ID: "h-a", Name: "server-a"},
		container.Host{ID: "h-b", Name: "server-b"},
	)

	host, id, err := resolveContainerRef("nginx", "server-b", deps)
	assert.NoError(t, err)
	assert.Equal(t, "h-b", host)
	assert.Equal(t, "id2", id)
}

func TestResolveContainerRef_UnknownHost(t *testing.T) {
	deps := resolverDeps([]container.Container{
		{ID: "id1", Name: "nginx", Host: "local"},
	})

	_, _, err := resolveContainerRef("nginx", "nope", deps)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no host matching")
}

func TestResolveContainerRef_NotFound(t *testing.T) {
	deps := resolverDeps([]container.Container{
		{ID: "id1", Name: "nginx", Host: "local"},
	})

	_, _, err := resolveContainerRef("does-not-exist", "", deps)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no container matching")
	assert.Contains(t, err.Error(), "find_containers")
}

func TestResolveContainerRef_EmptyContainer(t *testing.T) {
	deps := resolverDeps([]container.Container{
		{ID: "id1", Name: "nginx", Host: "local"},
	})

	_, _, err := resolveContainerRef("", "", deps)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "container_id is required")
}

func TestResolveContainerRef_LegacyIDWithHostFallsThrough(t *testing.T) {
	// host exists but the container is absent from the listing (e.g. a partial
	// host error). With an explicit host the resolver falls through to the
	// direct (hostID, ref) lookup — preserving the legacy id path exactly.
	deps := resolverDeps(
		[]container.Container{},
		container.Host{ID: "local", Name: "local"},
	)

	host, id, err := resolveContainerRef("abc123", "local", deps)
	assert.NoError(t, err)
	assert.Equal(t, "local", host)
	assert.Equal(t, "abc123", id)
}

func TestResolveContainerRef_IDBeatsName(t *testing.T) {
	// A pathological case: one container's id equals another's name. Id-first
	// ordering means the id match wins and the call is unambiguous — existing
	// id-based callers keep working identically.
	deps := resolverDeps([]container.Container{
		{ID: "shared", Name: "alpha", Host: "local"},
		{ID: "other", Name: "shared", Host: "local"},
	})

	host, id, err := resolveContainerRef("shared", "", deps)
	assert.NoError(t, err)
	assert.Equal(t, "local", host)
	assert.Equal(t, "shared", id)
}

// --- End-to-end tests through ExecuteTool proving the resolver is wired in ---

func TestExecuteTool_InspectContainer_ByName(t *testing.T) {
	mockHost := &MockHostService{}
	withResolver(mockHost, container.Container{ID: "abc123def456", Name: "nginx", Host: "local"})
	cs := container_support.NewContainerService(&MockClientService{}, container.Container{ID: "abc123def456", Name: "nginx", Host: "local"})
	mockHost.On("FindContainer", "local", "abc123def456", container.ContainerLabels(nil)).Return(cs, nil)

	// Pass the NAME in container_id and omit host_id entirely.
	resp := ExecuteTool(context.Background(), "inspect_container", `{"container_id":"nginx"}`, ToolDeps{HostService: mockHost})
	assert.True(t, resp.Success)
	assert.Equal(t, "nginx", resp.GetInspectContainer().Name)
	mockHost.AssertCalled(t, "FindContainer", "local", "abc123def456", container.ContainerLabels(nil))
}

func TestExecuteTool_RestartContainer_AmbiguousName_NoSilentPick(t *testing.T) {
	// Write tool with an ambiguous name across hosts must NOT act — it must
	// return the candidate list and never call FindContainer/ContainerAction.
	mockClient := &MockClientService{}
	mockHost := &MockHostService{}
	withResolver(mockHost,
		container.Container{ID: "id1", Name: "nginx", Host: "host-a"},
		container.Container{ID: "id2", Name: "nginx", Host: "host-b"},
	)

	resp := ExecuteTool(context.Background(), "restart_container", `{"container_id":"nginx"}`, ToolDeps{HostService: mockHost, EnableActions: true})
	assert.False(t, resp.Success)
	assert.Contains(t, resp.Error, "matches multiple containers")
	mockHost.AssertNotCalled(t, "FindContainer", mock.Anything, mock.Anything, mock.Anything)
	mockClient.AssertNotCalled(t, "ContainerAction", mock.Anything, mock.Anything, mock.Anything)
}

func TestExecuteTool_RestartContainer_ByName_HostInferred(t *testing.T) {
	mockClient := &MockClientService{}
	mockClient.On("ContainerAction", mock.Anything, mock.Anything, container.Restart).Return(nil)
	cs := container_support.NewContainerService(mockClient, container.Container{ID: "id1", Name: "nginx", Host: "host-a"})

	mockHost := &MockHostService{}
	withResolver(mockHost, container.Container{ID: "id1", Name: "nginx", Host: "host-a"})
	mockHost.On("FindContainer", "host-a", "id1", container.ContainerLabels(nil)).Return(cs, nil)

	resp := ExecuteTool(context.Background(), "restart_container", `{"container_id":"nginx"}`, ToolDeps{HostService: mockHost, EnableActions: true})
	assert.True(t, resp.Success)
	assert.Equal(t, "id1", resp.GetAction().ContainerId)
	mockClient.AssertCalled(t, "ContainerAction", mock.Anything, mock.Anything, container.Restart)
}

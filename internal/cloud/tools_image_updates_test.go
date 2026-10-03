package cloud

import (
	"context"
	"testing"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/imagecheck"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func imageUpdateHost(client *MockClientService, containers ...container.Container) *MockHostService {
	host := &MockHostService{}
	host.On("ListAllContainers", container.ContainerLabels(nil)).Return(containers, nil)
	host.On("Hosts").Return([]container.Host{{ID: "local", Name: "my-server"}})
	for _, c := range containers {
		host.On("FindContainer", c.Host, c.ID, container.ContainerLabels(nil)).
			Return(container.NewContainerService(client, c), nil).Maybe()
	}
	return host
}

func TestAvailableTools_imageUpdatesFollowTheMode(t *testing.T) {
	has := func(mode imagecheck.Mode) bool {
		_, ok := toolNames(AvailableTools(ToolDeps{ImageCheckMode: mode}))[toolCheckImageUpdates]
		return ok
	}
	assert.True(t, has(imagecheck.ModeAutomatic))
	assert.True(t, has(imagecheck.ModeManual), "a cloud call is someone asking, which manual mode answers")
	assert.False(t, has(imagecheck.ModeOff), "off means Dozzle never contacts a registry")
	assert.False(t, has(""), "unset must not reach registries")

	for _, d := range AvailableTools(ToolDeps{ImageCheckMode: imagecheck.ModeAutomatic}) {
		if d.Name == toolCheckImageUpdates {
			assert.True(t, d.ReadOnly)
		}
	}
}

func TestExecuteTool_checkImageUpdates(t *testing.T) {
	client := &MockClientService{imageResults: map[string]imagecheck.Result{
		"api": {Image: "app:latest", Status: imagecheck.StatusUpdateAvailable, LocalDigest: "sha256:old", RemoteDigest: "sha256:new"},
	}}
	host := imageUpdateHost(client,
		container.Container{ID: "api", Name: "api", Image: "app:latest", State: "running", Host: "local"},
		container.Container{ID: "db", Name: "db", Image: "postgres:18", State: "running", Host: "local"},
		container.Container{ID: "old", Name: "worker.1", Image: "app:latest", State: "exited", Host: "local",
			Labels: map[string]string{"com.docker.swarm.service.id": "svc"}},
	)

	resp := ExecuteTool(context.Background(), toolCheckImageUpdates, "", ToolDeps{HostService: host, ImageCheckMode: imagecheck.ModeAutomatic})
	require.True(t, resp.Success, resp.Error)

	got := resp.GetListContainers().GetContainers()
	require.Len(t, got, 2, "an exited swarm task was already replaced, so it is not checked")
	assert.Equal(t, "api", got[0].Id)
	assert.Equal(t, "my-server", got[0].HostName)
	assert.Equal(t, string(imagecheck.StatusUpdateAvailable), got[0].ImageUpdate.GetStatus())
	assert.Equal(t, "sha256:new", got[0].ImageUpdate.GetRemoteDigest())
	assert.Equal(t, string(imagecheck.StatusUpToDate), got[1].ImageUpdate.GetStatus())
	assert.False(t, client.imageForced.Load(), "the cache answers unless refresh is asked for")
}

func TestExecuteTool_checkImageUpdates_filtersAndRefreshes(t *testing.T) {
	client := &MockClientService{}
	host := imageUpdateHost(client,
		container.Container{ID: "api", Name: "api", Image: "app:latest", State: "running", Host: "local"},
		container.Container{ID: "db", Name: "db", Image: "postgres:18", State: "running", Host: "local"},
	)

	resp := ExecuteTool(context.Background(), toolCheckImageUpdates, `{"image":"POSTGRES","refresh":true}`, ToolDeps{HostService: host, ImageCheckMode: imagecheck.ModeManual})
	require.True(t, resp.Success, resp.Error)

	got := resp.GetListContainers().GetContainers()
	require.Len(t, got, 1)
	assert.Equal(t, "db", got[0].Id)
	assert.True(t, client.imageForced.Load())
	host.AssertNotCalled(t, "FindContainer", "local", "api", container.ContainerLabels(nil))
}

func TestExecuteTool_checkImageUpdates_offNeverTouchesDocker(t *testing.T) {
	host := &MockHostService{}

	resp := ExecuteTool(context.Background(), toolCheckImageUpdates, "", ToolDeps{HostService: host, ImageCheckMode: imagecheck.ModeOff})

	assert.False(t, resp.Success)
	assert.Contains(t, resp.Error, "--image-check-mode")
	host.AssertNotCalled(t, "ListAllContainers", container.ContainerLabels(nil))
}

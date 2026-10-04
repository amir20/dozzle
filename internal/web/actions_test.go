package web

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"testing"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/container/docker/swap/swaptest"
	docker_types "github.com/moby/moby/api/types/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func mockedClient() *MockedClient {
	mockedClient := new(MockedClient)
	c := container.Container{ID: "123", State: "running"}

	mockedClient.On("FindContainer", mock.Anything, "123").Return(c, nil)
	mockedClient.On("FindContainer", mock.Anything, "456").Return(container.Container{}, errors.New("container not found"))
	mockedClient.On("ContainerActions", mock.Anything, container.Start, c.ID).Return(nil)
	mockedClient.On("ContainerActions", mock.Anything, container.Stop, c.ID).Return(nil)
	mockedClient.On("ContainerActions", mock.Anything, container.Restart, c.ID).Return(nil)
	mockedClient.On("ContainerActions", mock.Anything, container.Start, mock.Anything).Return(errors.New("container not found"))
	mockedClient.On("ContainerActions", mock.Anything, container.ContainerAction("something-else"), c.ID).Return(errors.New("unknown action"))
	mockedClient.On("Host").Return(container.Host{ID: "localhost"})
	mockedClient.On("ListContainers", mock.Anything, mock.Anything).Return([]container.Container{c}, nil)
	mockedClient.On("ContainerEvents", mock.Anything, mock.Anything).Return(nil)

	return mockedClient
}

func Test_handler_containerActions_stop(t *testing.T) {
	mockedClient := mockedClient()

	handler := createHandler(mockedClient, nil, Config{Base: "/", EnableActions: true, Authorization: Authorization{Provider: NONE}})
	req, err := http.NewRequest("POST", "/api/hosts/localhost/containers/123/actions/stop", nil)
	require.NoError(t, err, "Request should not return an error.")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, 204, rr.Code)
}

func Test_handler_containerActions_restart(t *testing.T) {
	mockedClient := mockedClient()

	handler := createHandler(mockedClient, nil, Config{Base: "/", EnableActions: true, Authorization: Authorization{Provider: NONE}})
	req, err := http.NewRequest("POST", "/api/hosts/localhost/containers/123/actions/restart", nil)
	require.NoError(t, err, "Request should not return an error.")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, 204, rr.Code)
}

func Test_handler_containerActions_unknown_action(t *testing.T) {
	mockedClient := mockedClient()

	handler := createHandler(mockedClient, nil, Config{Base: "/", EnableActions: true, Authorization: Authorization{Provider: NONE}})
	req, err := http.NewRequest("POST", "/api/hosts/localhost/containers/123/actions/something-else", nil)
	require.NoError(t, err, "Request should not return an error.")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, 400, rr.Code)
}

func Test_handler_containerActions_unknown_container(t *testing.T) {
	mockedClient := mockedClient()

	handler := createHandler(mockedClient, nil, Config{Base: "/", EnableActions: true, Authorization: Authorization{Provider: NONE}})
	req, err := http.NewRequest("POST", "/api/hosts/localhost/containers/456/actions/start", nil)
	require.NoError(t, err, "Request should not return an error.")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, 404, rr.Code)
}

func Test_handler_containerActions_start(t *testing.T) {
	mockedClient := mockedClient()

	handler := createHandler(mockedClient, nil, Config{Base: "/", EnableActions: true, Authorization: Authorization{Provider: NONE}})
	req, err := http.NewRequest("POST", "/api/hosts/localhost/containers/123/actions/start", nil)
	require.NoError(t, err, "Request should not return an error.")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, 204, rr.Code)
}

func Test_handler_containerUpdate_up_to_date(t *testing.T) {
	mockedClient := mockedClient()

	inspectResp := docker_types.InspectResponse{
		Image: "sha256:current",
		State: &docker_types.State{Running: true, Status: "running"},
		Config: &docker_types.Config{
			Image: "test:v1",
		},
	}
	mockedClient.On("ContainerInspect", mock.Anything, "123").Return(inspectResp, nil)

	pullResp := `{"status":"Already exists","id":"abc123"}` + "\n" +
		`{"status":"Status: Image is up to date for test:v1"}` + "\n"
	mockedClient.On("ImagePull", mock.Anything, "test:v1").Return(io.NopCloser(strings.NewReader(pullResp)), nil)
	// The tag still resolves to what the container runs, so nothing to do.
	mockedClient.On("ImageID", mock.Anything, "test:v1").Return("sha256:current", nil)

	handler := createHandler(mockedClient, nil, Config{Base: "/", EnableActions: true, Authorization: Authorization{Provider: NONE}})
	req, err := http.NewRequest("POST", "/api/hosts/localhost/containers/123/actions/update", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, 200, rr.Code)
	assert.Contains(t, rr.Body.String(), `"up-to-date"`)
}

// A stopped container is refused before the progress stream opens, and the
// host is never asked.
func Test_handler_containerUpdate_stopped(t *testing.T) {
	m := new(MockedClient)
	c := container.Container{ID: "123", State: "exited"}
	m.On("FindContainer", mock.Anything, "123").Return(c, nil)
	m.On("Host").Return(container.Host{ID: "localhost"})
	m.On("ListContainers", mock.Anything, mock.Anything).Return([]container.Container{c}, nil)
	m.On("ContainerEvents", mock.Anything, mock.Anything).Return(nil)

	handler := createHandler(m, nil, Config{Base: "/", EnableActions: true, Authorization: Authorization{Provider: NONE}})
	req, err := http.NewRequest("POST", "/api/hosts/localhost/containers/123/actions/update", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusConflict, rr.Code)
	assert.Contains(t, rr.Body.String(), container.ErrNotRunning.Error())
	m.AssertNotCalled(t, "ContainerInspect", mock.Anything, mock.Anything)
	m.AssertNotCalled(t, "ImagePull", mock.Anything, mock.Anything)
}

func Test_handler_containerUpdate_new_image(t *testing.T) {
	m := new(MockedClient)
	c := container.Container{ID: "123", State: "running"}

	m.On("FindContainer", mock.Anything, "123").Return(c, nil)
	m.On("Host").Return(container.Host{ID: "localhost"})
	m.On("ListContainers", mock.Anything, mock.Anything).Return([]container.Container{c}, nil)
	m.On("ContainerEvents", mock.Anything, mock.Anything).Return(nil)

	inspectResp := docker_types.InspectResponse{
		Name:  "/test-container",
		Image: "sha256:old",
		State: &docker_types.State{Running: true, Status: "running"},
		Config: &docker_types.Config{
			Image: "test:v1",
		},
		HostConfig:      &docker_types.HostConfig{},
		NetworkSettings: &docker_types.NetworkSettings{},
	}
	m.On("ContainerInspect", mock.Anything, "123").Return(inspectResp, nil)

	pullResp := `{"status":"Already exists","id":"abc123"}` + "\n" +
		`{"status":"Status: Downloaded newer image for test:v1"}` + "\n"
	m.On("ImagePull", mock.Anything, "test:v1").Return(io.NopCloser(strings.NewReader(pullResp)), nil)
	m.On("ImageID", mock.Anything, "test:v1").Return("sha256:new", nil)
	m.On("NetworkDependents", mock.Anything, mock.Anything, "test-container").Return(nil, nil)
	swaptest.FastTimings(t)
	m.engine = swaptest.New(inspectResp)
	m.engine.NewIDs = []string{"new-123"}

	handler := createHandler(m, nil, Config{Base: "/", EnableActions: true, Authorization: Authorization{Provider: NONE}})
	req, err := http.NewRequest("POST", "/api/hosts/localhost/containers/123/actions/update", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, 200, rr.Code)
	assert.Contains(t, rr.Body.String(), `"verifying"`)
	assert.Contains(t, rr.Body.String(), `"done"`)
	assert.Equal(t, "sha256:old", m.engine.Created[0].Config.Labels[container.PreviousImageLabel])
}

// A container joined to the updated one's network namespace by id (compose's
// network_mode: service:x) is cut off when the old one goes, so it is
// recreated pointing at the replacement. #5289
func Test_handler_containerUpdate_rejoins_network_dependents(t *testing.T) {
	const oldID = "1230000000000000000000000000000000000000000000000000000000000000"
	m := new(MockedClient)
	c := container.Container{ID: "123", State: "running"}

	m.On("FindContainer", mock.Anything, "123").Return(c, nil)
	m.On("Host").Return(container.Host{ID: "localhost"})
	m.On("ListContainers", mock.Anything, mock.Anything).Return([]container.Container{c}, nil)
	m.On("ContainerEvents", mock.Anything, mock.Anything).Return(nil)

	sidecar := docker_types.InspectResponse{
		ID:              oldID,
		Name:            "/sidecar",
		Image:           "sha256:old",
		State:           &docker_types.State{Running: true, Status: "running"},
		Config:          &docker_types.Config{Image: "test:v1"},
		HostConfig:      &docker_types.HostConfig{},
		NetworkSettings: &docker_types.NetworkSettings{},
	}
	m.On("ContainerInspect", mock.Anything, "123").Return(sidecar, nil)
	swaptest.FastTimings(t)
	m.engine = swaptest.New(sidecar)
	m.engine.NewIDs = []string{"new-123"}
	m.On("ContainerInspect", mock.Anything, "app-id").Return(docker_types.InspectResponse{
		ID:         "app-id",
		Name:       "/app",
		Image:      "sha256:app",
		State:      &docker_types.State{Running: true},
		Config:     &docker_types.Config{Image: "app:latest"},
		HostConfig: &docker_types.HostConfig{NetworkMode: "container:" + oldID},
	}, nil)

	pullResp := `{"status":"Status: Downloaded newer image for test:v1"}` + "\n"
	m.On("ImagePull", mock.Anything, "test:v1").Return(io.NopCloser(strings.NewReader(pullResp)), nil)
	m.On("ImageID", mock.Anything, "test:v1").Return("sha256:new", nil)
	m.On("NetworkDependents", mock.Anything, oldID, "sidecar").Return([]string{"app-id"}, nil)

	m.On("ContainerActions", mock.Anything, container.Stop, "app-id").Return(nil)
	m.On("ContainerRemove", mock.Anything, "app-id").Return(nil)
	m.On("ContainerCreate", mock.Anything, mock.MatchedBy(func(i docker_types.InspectResponse) bool {
		return i.HostConfig.NetworkMode == "container:new-123"
	}), "app").Return("new-app", nil)
	m.On("ContainerActions", mock.Anything, container.Start, "new-app").Return(nil)

	handler := createHandler(m, nil, Config{Base: "/", EnableActions: true, Authorization: Authorization{Provider: NONE}})
	req, err := http.NewRequest("POST", "/api/hosts/localhost/containers/123/actions/update", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, 200, rr.Code)
	assert.Contains(t, rr.Body.String(), `"done"`)
	m.AssertCalled(t, "ContainerActions", mock.Anything, container.Start, "new-app")
}

// A pull that reports nothing new must still recreate when the tag has moved
// on locally, which happens when the image was pulled or built beforehand.
func Test_handler_containerUpdate_recreates_when_image_already_local(t *testing.T) {
	m := new(MockedClient)
	c := container.Container{ID: "123", State: "running"}

	m.On("FindContainer", mock.Anything, "123").Return(c, nil)
	m.On("Host").Return(container.Host{ID: "localhost"})
	m.On("ListContainers", mock.Anything, mock.Anything).Return([]container.Container{c}, nil)
	m.On("ContainerEvents", mock.Anything, mock.Anything).Return(nil)

	inspectResp := docker_types.InspectResponse{
		Name:  "/test-container",
		Image: "sha256:old",
		State: &docker_types.State{Running: true, Status: "running"},
		Config: &docker_types.Config{
			Image: "test:v1",
		},
		HostConfig:      &docker_types.HostConfig{},
		NetworkSettings: &docker_types.NetworkSettings{},
	}
	m.On("ContainerInspect", mock.Anything, "123").Return(inspectResp, nil)

	// The pull finds nothing to download because the image is already here.
	pullResp := `{"status":"Status: Image is up to date for test:v1"}` + "\n"
	m.On("ImagePull", mock.Anything, "test:v1").Return(io.NopCloser(strings.NewReader(pullResp)), nil)
	// The tag nonetheless points somewhere else than the running container.
	m.On("ImageID", mock.Anything, "test:v1").Return("sha256:new", nil)
	m.On("NetworkDependents", mock.Anything, mock.Anything, "test-container").Return(nil, nil)
	swaptest.FastTimings(t)
	m.engine = swaptest.New(inspectResp)
	m.engine.NewIDs = []string{"new-123"}

	handler := createHandler(m, nil, Config{Base: "/", EnableActions: true, Authorization: Authorization{Provider: NONE}})
	req, err := http.NewRequest("POST", "/api/hosts/localhost/containers/123/actions/update", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, 200, rr.Code)
	assert.Contains(t, rr.Body.String(), `"done"`)
}

func Test_handler_containerUpdate_not_found(t *testing.T) {
	mockedClient := mockedClient()

	handler := createHandler(mockedClient, nil, Config{Base: "/", EnableActions: true, Authorization: Authorization{Provider: NONE}})
	req, err := http.NewRequest("POST", "/api/hosts/localhost/containers/456/actions/update", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, 404, rr.Code)
}

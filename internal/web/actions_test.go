package web

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"testing"

	"github.com/amir20/dozzle/internal/container"
	docker_types "github.com/moby/moby/api/types/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func mockedClient() *MockedClient {
	mockedClient := new(MockedClient)
	c := container.Container{ID: "123"}

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
		Config: &docker_types.Config{
			Image: "test:v1",
		},
	}
	mockedClient.On("ContainerInspect", mock.Anything, "123").Return(inspectResp, nil)

	pullResp := `{"status":"Already exists","id":"abc123"}` + "\n" +
		`{"status":"Status: Image is up to date for test:v1"}` + "\n"
	mockedClient.On("ImagePull", mock.Anything, "test:v1").Return(io.NopCloser(strings.NewReader(pullResp)), nil)

	handler := createHandler(mockedClient, nil, Config{Base: "/", EnableActions: true, Authorization: Authorization{Provider: NONE}})
	req, err := http.NewRequest("POST", "/api/hosts/localhost/containers/123/actions/update", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, 200, rr.Code)
	assert.Contains(t, rr.Body.String(), `"up-to-date"`)
}

func Test_handler_containerUpdate_new_image(t *testing.T) {
	m := new(MockedClient)
	c := container.Container{ID: "123"}

	m.On("FindContainer", mock.Anything, "123").Return(c, nil)
	m.On("ContainerActions", mock.Anything, container.Start, "new-123").Return(nil)
	m.On("Host").Return(container.Host{ID: "localhost"})
	m.On("ListContainers", mock.Anything, mock.Anything).Return([]container.Container{c}, nil)
	m.On("ContainerEvents", mock.Anything, mock.Anything).Return(nil)

	inspectResp := docker_types.InspectResponse{
		Name: "/test-container",
		Config: &docker_types.Config{
			Image: "test:v1",
		},
		NetworkSettings: &docker_types.NetworkSettings{},
	}
	m.On("ContainerInspect", mock.Anything, "123").Return(inspectResp, nil)

	pullResp := `{"status":"Already exists","id":"abc123"}` + "\n" +
		`{"status":"Status: Downloaded newer image for test:v1"}` + "\n"
	m.On("ImagePull", mock.Anything, "test:v1").Return(io.NopCloser(strings.NewReader(pullResp)), nil)
	m.On("ContainerRemove", mock.Anything, "123").Return(nil)
	m.On("ContainerCreate", mock.Anything, mock.Anything, "test-container").Return("new-123", nil)

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

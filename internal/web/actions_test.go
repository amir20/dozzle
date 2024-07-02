package web

import (
	"errors"
	"net/http"
	"net/http/httptest"

	"testing"

	"github.com/amir20/dozzle/internal/docker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func mockedClient() *MockedClient {
	mockedClient := new(MockedClient)
	container := docker.Container{ID: "123"}

	mockedClient.On("FindContainer", "123").Return(container, nil)
	mockedClient.On("FindContainer", "456").Return(docker.Container{}, errors.New("container not found"))
	mockedClient.On("ContainerActions", docker.Start, container.ID).Return(nil)
	mockedClient.On("ContainerActions", docker.Stop, container.ID).Return(nil)
	mockedClient.On("ContainerActions", docker.Restart, container.ID).Return(nil)
	mockedClient.On("ContainerActions", docker.Start, mock.Anything).Return(errors.New("container not found"))
	mockedClient.On("ContainerActions", docker.ContainerAction("something-else"), container.ID).Return(errors.New("unknown action"))
	mockedClient.On("Host").Return(docker.Host{ID: "localhost"})
	mockedClient.On("ListContainers").Return([]docker.Container{container}, nil)
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

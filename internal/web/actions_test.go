package web

import (
	"errors"
	"net/http"
	"net/http/httptest"

	"testing"

	"github.com/amir20/dozzle/internal/container"
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

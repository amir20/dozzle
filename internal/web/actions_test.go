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

func get_mocked_client() *MockedClient {
	mockedClient := new(MockedClient)
	container := docker.Container{ID: "123"}

	mockedClient.On("FindContainer", "123").Return(container, nil)
	mockedClient.On("FindContainer", "456").Return(docker.Container{}, errors.New("container not found"))

	mockedClient.On("ContainerActions", "start", container.ID).Return(nil)
	mockedClient.On("ContainerActions", "stop", container.ID).Return(nil)
	mockedClient.On("ContainerActions", "restart", container.ID).Return(nil)
	mockedClient.On("ContainerActions", "something-else", container.ID).Return(errors.New("unknown action"))

	mockedClient.On("ContainerActions", "start", mock.Anything).Return(errors.New("container not found"))

	return mockedClient
}

func Test_handler_containerActions_stop(t *testing.T) {
	mockedClient := get_mocked_client()

	handler := createHandler(mockedClient, nil, Config{Base: "/", EnableActions: true, Authorization: Authorization{Provider: NONE}})
	req, err := http.NewRequest("POST", "/api/hosts/localhost/containers/123/actions/stop", nil)
	require.NoError(t, err, "Request should not return an error.")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, 200, rr.Code)
}

func Test_handler_containerActions_restart(t *testing.T) {
	mockedClient := get_mocked_client()

	handler := createHandler(mockedClient, nil, Config{Base: "/", EnableActions: true, Authorization: Authorization{Provider: NONE}})
	req, err := http.NewRequest("POST", "/api/hosts/localhost/containers/123/actions/restart", nil)
	require.NoError(t, err, "Request should not return an error.")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, 200, rr.Code)
}

func Test_handler_containerActions_unknown_action(t *testing.T) {
	mockedClient := get_mocked_client()

	handler := createHandler(mockedClient, nil, Config{Base: "/", EnableActions: true, Authorization: Authorization{Provider: NONE}})
	req, err := http.NewRequest("POST", "/api/hosts/localhost/containers/123/actions/something-else", nil)
	require.NoError(t, err, "Request should not return an error.")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, 500, rr.Code)
}

func Test_handler_containerActions_unknown_container(t *testing.T) {
	mockedClient := get_mocked_client()

	handler := createHandler(mockedClient, nil, Config{Base: "/", EnableActions: true, Authorization: Authorization{Provider: NONE}})
	req, err := http.NewRequest("POST", "/api/hosts/localhost/containers/456/actions/start", nil)
	require.NoError(t, err, "Request should not return an error.")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, 404, rr.Code)
}

func Test_handler_containerActions_start(t *testing.T) {
	mockedClient := get_mocked_client()

	handler := createHandler(mockedClient, nil, Config{Base: "/", EnableActions: true, Authorization: Authorization{Provider: NONE}})
	req, err := http.NewRequest("POST", "/api/hosts/localhost/containers/123/actions/start", nil)
	require.NoError(t, err, "Request should not return an error.")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, 200, rr.Code)
}

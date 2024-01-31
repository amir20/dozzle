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
	req, err := http.NewRequest("POST", "/api/actions/stop/localhost/123", nil)
	require.NoError(t, err, "Request should not return an error.")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, rr.Code, 200)
}

func Test_handler_containerActions_restart(t *testing.T) {
	mockedClient := get_mocked_client()

	handler := createHandler(mockedClient, nil, Config{Base: "/", EnableActions: true, Authorization: Authorization{Provider: NONE}})
	req, err := http.NewRequest("POST", "/api/actions/restart/localhost/123", nil)
	require.NoError(t, err, "Request should not return an error.")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, rr.Code, 200)
}

func Test_handler_containerActions_unknown_action(t *testing.T) {
	mockedClient := get_mocked_client()

	handler := createHandler(mockedClient, nil, Config{Base: "/", EnableActions: true, Authorization: Authorization{Provider: NONE}})
	req, err := http.NewRequest("POST", "/api/actions/something-else/localhost/123", nil)
	require.NoError(t, err, "Request should not return an error.")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, rr.Code, 500)
}

func Test_handler_containerActions_unknown_container(t *testing.T) {
	mockedClient := get_mocked_client()

	handler := createHandler(mockedClient, nil, Config{Base: "/", EnableActions: true, Authorization: Authorization{Provider: NONE}})
	req, err := http.NewRequest("POST", "/api/actions/start/localhost/456", nil)
	require.NoError(t, err, "Request should not return an error.")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, rr.Code, 404)
}

func Test_handler_containerActions_start(t *testing.T) {
	mockedClient := get_mocked_client()

	handler := createHandler(mockedClient, nil, Config{Base: "/", EnableActions: true, Authorization: Authorization{Provider: NONE}})
	req, err := http.NewRequest("POST", "/api/actions/start/localhost/123", nil)
	require.NoError(t, err, "Request should not return an error.")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, rr.Code, 200)
}

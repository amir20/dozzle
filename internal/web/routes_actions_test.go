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

func Test_handler_containerActions(t *testing.T) {
	mockedClient := new(MockedClient)
	mockedClient.On("FindContainer", "123").Return(docker.Container{ID: "123"}, nil)
	mockedClient.On("FindContainer", "456").Return(docker.Container{}, errors.New("container not found"))

	mockedClient.On("ContainerActions", "start", docker.Container{ID: "123"}).Return(nil)
	mockedClient.On("ContainerActions", "stop", docker.Container{ID: "123"}).Return(nil)
	mockedClient.On("ContainerActions", "restart", docker.Container{ID: "123"}).Return(nil)
	mockedClient.On("ContainerActions", "something-else", docker.Container{ID: "123"}).Return(errors.New("unknown action"))

	mockedClient.On("ContainerActions", "start", mock.Anything).Return(errors.New("container not found"))

	handler := createHandler(mockedClient, nil, Config{Base: "/", EnableActions: true, Authorization: Authorization{Provider: NONE}})
	req, err := http.NewRequest("POST", "/api/actions/start/localhost/123", nil)
	require.NoError(t, err, "Request should not return an error.")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, rr.Code, 200)

	req, err = http.NewRequest("POST", "/api/actions/stop/localhost/123", nil)
	require.NoError(t, err, "Request should not return an error.")

	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, rr.Code, 200)

	req, err = http.NewRequest("POST", "/api/actions/restart/localhost/123", nil)
	require.NoError(t, err, "Request should not return an error.")

	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, rr.Code, 200)

	req, err = http.NewRequest("POST", "/api/actions/something-else/localhost/123", nil)
	require.NoError(t, err, "Request should not return an error.")

	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, rr.Code, 500)

	req, err = http.NewRequest("POST", "/api/actions/start/localhost/456", nil)
	require.NoError(t, err, "Request should not return an error.")

	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, rr.Code, 404)
}

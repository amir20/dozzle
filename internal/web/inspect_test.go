package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/amir20/dozzle/internal/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func doInspect(t *testing.T, enableShell bool) *httptest.ResponseRecorder {
	t.Helper()

	m := new(MockedClient)
	c := container.Container{
		ID:            "123",
		Image:         "test:v1",
		Env:           []string{"DB_PASS=hunter2", "EMPTY="},
		RestartPolicy: "always",
		RestartCount:  3,
		OOMKilled:     true,
		ExitCode:      137,
		FullyLoaded:   true,
	}
	m.On("FindContainer", mock.Anything, "123").Return(c, nil)
	m.On("ListContainers", mock.Anything, mock.Anything).Return([]container.Container{c}, nil)
	m.On("Host").Return(container.Host{ID: "localhost"})
	m.On("ContainerEvents", mock.Anything, mock.Anything).Return(nil)

	handler := createHandler(m, nil, Config{Base: "/", EnableShell: enableShell, Authorization: Authorization{Provider: NONE}})
	req, err := http.NewRequest("GET", "/api/hosts/localhost/containers/123/inspect", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr
}

func Test_handler_inspectContainer_reveals_env_with_shell(t *testing.T) {
	rr := doInspect(t, true)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.JSONEq(t, `{
		"restartPolicy": "always",
		"restartCount": 3,
		"oomKilled": true,
		"exitCode": 137,
		"env": [{"key": "DB_PASS", "value": "hunter2"}, {"key": "EMPTY"}],
		"envRevealed": true
	}`, rr.Body.String())
}

// Without shell there is no other way to read the environment, so the values
// never leave the server.
func Test_handler_inspectContainer_hides_env_values_without_shell(t *testing.T) {
	rr := doInspect(t, false)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.NotContains(t, rr.Body.String(), "hunter2")
	assert.Contains(t, rr.Body.String(), `"envRevealed":false`)
	assert.Contains(t, rr.Body.String(), `"key":"DB_PASS"`)
}

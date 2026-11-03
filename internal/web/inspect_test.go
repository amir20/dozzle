package web

import (
	"context"
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
		Networks:      []container.ContainerNetwork{{Name: "app_default", IP: "172.18.0.4"}},
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
		"networks": [{"name": "app_default", "ip": "172.18.0.4"}],
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

func (m *MockedClient) InspectNetwork(ctx context.Context, name string) (container.Network, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(container.Network), args.Error(1)
}

func (m *MockedClient) InspectVolume(ctx context.Context, name string) (container.Volume, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(container.Volume), args.Error(1)
}

// resourceHandler serves a container on app_default with a data volume, next to
// a neighbour the user's filter allows and one only the engine knows about.
func resourceHandler(t *testing.T) (*MockedClient, http.Handler) {
	t.Helper()

	m := new(MockedClient)
	c := container.Container{
		ID:          "123",
		Name:        "web",
		Networks:    []container.ContainerNetwork{{Name: "app_default", IP: "172.18.0.4"}},
		Mounts:      []container.Mount{{Type: "volume", Name: "data", Destination: "/data"}, {Type: "bind", Source: "/etc/hosts", Destination: "/etc/hosts"}},
		FullyLoaded: true,
	}
	neighbour := container.Container{ID: "456", Name: "db"}
	m.On("FindContainer", mock.Anything, "123").Return(c, nil)
	m.On("FindContainer", mock.Anything, "456").Return(neighbour, nil)
	m.On("ListContainers", mock.Anything, mock.Anything).Return([]container.Container{c, neighbour}, nil)
	m.On("Host").Return(container.Host{ID: "localhost"})
	m.On("ContainerEvents", mock.Anything, mock.Anything).Return(nil)

	return m, createDefaultHandler(m)
}

func getResource(t *testing.T, handler http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	req, err := http.NewRequest("GET", path, nil)
	require.NoError(t, err)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr
}

func Test_handler_inspectNetwork_drops_containers_the_user_cannot_see(t *testing.T) {
	m, handler := resourceHandler(t)
	m.On("InspectNetwork", mock.Anything, "app_default").Return(container.Network{
		Name:    "app_default",
		Driver:  "bridge",
		Subnets: []container.NetworkSubnet{{Subnet: "172.18.0.0/16", Gateway: "172.18.0.1", InUse: 4, Available: 65530}},
		Containers: []container.NetworkEndpoint{
			{ID: "123", Name: "web", IP: "172.18.0.4"},
			{ID: "456", Name: "db", IP: "172.18.0.2"},
			{ID: "789", Name: "hidden", IP: "172.18.0.3"},
		},
	}, nil)

	rr := getResource(t, handler, "/api/hosts/localhost/containers/123/networks/app_default")

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `"subnet":"172.18.0.0/16"`)
	assert.Contains(t, rr.Body.String(), `"name":"db"`)
	assert.NotContains(t, rr.Body.String(), "hidden")
}

// The route goes through a container, so it cannot be used to read a network the
// container is not on.
func Test_handler_inspectNetwork_refuses_a_network_the_container_is_not_on(t *testing.T) {
	m, handler := resourceHandler(t)

	rr := getResource(t, handler, "/api/hosts/localhost/containers/123/networks/other")

	assert.Equal(t, http.StatusNotFound, rr.Code)
	m.AssertNotCalled(t, "InspectNetwork", mock.Anything, mock.Anything)
}

func Test_handler_inspectVolume_lists_who_else_mounts_it(t *testing.T) {
	m, handler := resourceHandler(t)
	m.On("InspectVolume", mock.Anything, "data").Return(container.Volume{
		Name:       "data",
		Driver:     "local",
		Options:    map[string]string{"type": "nfs", "device": ":/exports/data"},
		Containers: []container.VolumeUser{{ID: "123", Name: "web"}, {ID: "456", Name: "db"}, {ID: "789", Name: "hidden"}},
	}, nil)

	rr := getResource(t, handler, "/api/hosts/localhost/containers/123/volumes/data")

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `"device":":/exports/data"`)
	assert.Contains(t, rr.Body.String(), `"name":"db"`)
	assert.NotContains(t, rr.Body.String(), "hidden")
}

// A bind mount has no volume behind it, and an unmounted volume is not this
// container's to show.
func Test_handler_inspectVolume_refuses_what_the_container_does_not_mount(t *testing.T) {
	m, handler := resourceHandler(t)

	assert.Equal(t, http.StatusNotFound, get(t, handler, "/api/hosts/localhost/containers/123/volumes/other").Code)
	m.AssertNotCalled(t, "InspectVolume", mock.Anything, mock.Anything)
}

// An empty host segment used to make hostKey call log.Fatal, taking the whole
// server down. It must fail like any other unknown host.
func Test_handler_empty_host_is_not_found(t *testing.T) {
	handler := createDefaultHandler(nil)
	for _, path := range []string{"/api/hosts//containers/123/inspect", "/api/hosts//logs/stream"} {
		req, err := http.NewRequest("GET", path, nil)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusNotFound, rr.Code, path)
	}
}

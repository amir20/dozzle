package web

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/imagecheck"
	docker_types "github.com/moby/moby/api/types/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func imageCheckHandler(t *testing.T, labels map[string]string) *MockedClient {
	t.Helper()

	m := new(MockedClient)
	c := container.Container{ID: "123", Image: "test:v1", Labels: labels}

	m.On("FindContainer", mock.Anything, "123").Return(c, nil)
	m.On("FindContainer", mock.Anything, "456").Return(container.Container{}, errors.New("container not found"))
	// The container store is seeded from ListContainers, which is what the
	// handler's lookup actually resolves against.
	m.On("ListContainers", mock.Anything, mock.Anything).Return([]container.Container{c}, nil)
	m.On("Host").Return(container.Host{ID: "localhost"})
	m.On("ContainerEvents", mock.Anything, mock.Anything).Return(nil)

	return m
}

func doImageCheck(t *testing.T, m *MockedClient, mode imagecheck.Mode, path string) *httptest.ResponseRecorder {
	t.Helper()

	handler := createHandler(m, nil, Config{Base: "/", ImageCheckMode: mode, Authorization: Authorization{Provider: NONE}})
	req, err := http.NewRequest("GET", path, nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr
}

// The endpoint must not exist at all when checks are off, so an operator can
// verify Dozzle makes no outbound registry requests.
func Test_handler_checkImageUpdate_off_removes_route(t *testing.T) {
	m := imageCheckHandler(t, nil)

	rr := doImageCheck(t, m, imagecheck.ModeOff, "/api/hosts/localhost/containers/123/image/check")

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

// Manual mode answers without contacting a registry unless force is set.
func Test_handler_checkImageUpdate_manual_does_not_reach_registry(t *testing.T) {
	m := imageCheckHandler(t, nil)

	rr := doImageCheck(t, m, imagecheck.ModeManual, "/api/hosts/localhost/containers/123/image/check")

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `"skipped"`)
	m.AssertNotCalled(t, "ImageRepoDigests", mock.Anything, mock.Anything)
}

// A container can opt out of checks entirely with a label.
func Test_handler_checkImageUpdate_respects_skip_label(t *testing.T) {
	m := imageCheckHandler(t, map[string]string{imagecheck.SkipLabel: "false"})

	rr := doImageCheck(t, m, imagecheck.ModeAutomatic, "/api/hosts/localhost/containers/123/image/check")

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `"skipped"`)
	m.AssertNotCalled(t, "ImageRepoDigests", mock.Anything, mock.Anything)
}

// A locally built image has no RepoDigest, so it is reported as uncheckable
// rather than as an error or a false "up to date".
func Test_handler_checkImageUpdate_locally_built_image(t *testing.T) {
	m := imageCheckHandler(t, nil)
	m.On("ContainerInspect", mock.Anything, "123").Return(docker_types.InspectResponse{
		Image:  "sha256:local",
		Config: &docker_types.Config{Image: "my-app:latest"},
	}, nil)
	m.On("ImageRepoDigests", mock.Anything, "sha256:local").Return([]string{}, nil)

	rr := doImageCheck(t, m, imagecheck.ModeAutomatic, "/api/hosts/localhost/containers/123/image/check")

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `"not-checkable"`)
}

// The check must be answered even when actions are disabled: knowing an update
// exists is useful when the user updates the container themselves.
func Test_handler_checkImageUpdate_works_without_actions(t *testing.T) {
	m := imageCheckHandler(t, nil)
	m.On("ContainerInspect", mock.Anything, "123").Return(docker_types.InspectResponse{
		Image:  "sha256:local",
		Config: &docker_types.Config{Image: "my-app:latest"},
	}, nil)
	m.On("ImageRepoDigests", mock.Anything, "sha256:local").Return([]string{}, nil)

	handler := createHandler(m, nil, Config{
		Base:           "/",
		ImageCheckMode: imagecheck.ModeAutomatic,
		EnableActions:  false,
		Authorization:  Authorization{Provider: NONE},
	})
	req, err := http.NewRequest("GET", "/api/hosts/localhost/containers/123/image/check", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func Test_handler_checkImageUpdate_not_found(t *testing.T) {
	m := imageCheckHandler(t, nil)

	rr := doImageCheck(t, m, imagecheck.ModeAutomatic, "/api/hosts/localhost/containers/456/image/check")

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

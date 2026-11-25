package web

import (
	"context"
	"errors"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/hostservice"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeScaler stands in for MultiHostService, which scales swarm services.
type fakeScaler struct {
	HostService
	err      error
	name     string
	replicas uint64
}

func (f *fakeScaler) ScaleService(_ context.Context, name string, replicas uint64, _ container.ContainerLabels) error {
	f.name, f.replicas = name, replicas
	return f.err
}

func postScale(t *testing.T, hostService HostService, mode, body string) int {
	t.Helper()
	var content fs.FS = fstest.MapFS{"index.html": {Data: []byte("index page")}}
	router := createRouter(&handler{
		hostService: hostService,
		content:     content,
		config:      &Config{Base: "/", Mode: mode, EnableActions: true, Authorization: Authorization{Provider: NONE}},
	})
	req, err := http.NewRequest("POST", "/api/swarm/services/shop_web/scale", strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr.Code
}

func Test_handler_scaleService(t *testing.T) {
	scaler := &fakeScaler{}
	assert.Equal(t, http.StatusNoContent, postScale(t, scaler, "swarm", `{"replicas":4}`))
	assert.Equal(t, "shop_web", scaler.name)
	assert.Equal(t, uint64(4), scaler.replicas)
}

// Zero is refused: a service with no tasks drops out of Dozzle along with the
// control that would scale it back up.
func Test_handler_scaleService_badBody(t *testing.T) {
	for _, body := range []string{`{"replicas":0}`, `{}`, `{"replicas":-1}`, `{"replicas":"3"}`, `nope`} {
		scaler := &fakeScaler{}
		assert.Equal(t, http.StatusBadRequest, postScale(t, scaler, "swarm", body), body)
		assert.Empty(t, scaler.name, body)
	}
}

func Test_handler_scaleService_errors(t *testing.T) {
	cases := map[string]struct {
		err  error
		code int
	}{
		"hidden by the label filter": {hostservice.ErrServiceNotFound, http.StatusNotFound},
		"global service":             {container.ErrServiceNotReplicated, http.StatusBadRequest},
		"no manager reachable":       {container.ErrNotSwarmManager, http.StatusConflict},
		"engine failure":             {errors.New("update out of sequence"), http.StatusInternalServerError},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.code, postScale(t, &fakeScaler{err: tc.err}, "swarm", `{"replicas":2}`))
		})
	}
}

// k8s has no swarm services, so the route is not registered there.
func Test_handler_scaleService_notRegisteredInK8s(t *testing.T) {
	scaler := &fakeScaler{}
	assert.NotEqual(t, http.StatusNoContent, postScale(t, scaler, "k8s", `{"replicas":2}`))
	assert.Empty(t, scaler.name)
}

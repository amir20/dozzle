package web

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/hostservice"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeRestarter stands in for K8sClusterService, the only host service that can
// restart a workload.
type fakeRestarter struct {
	HostService
	err    error
	called string
}

func (f *fakeRestarter) RolloutRestart(_ context.Context, namespace, kind, name string, _ container.ContainerLabels) error {
	f.called = namespace + "/" + kind + "/" + name
	return f.err
}

func rolloutRouter(hostService HostService, mode string) http.Handler {
	memFs := afero.NewMemMapFs()
	afero.WriteFile(memFs, "index.html", []byte("index page"), 0644)
	var content fs.FS = afero.NewIOFS(memFs)
	return createRouter(&handler{
		hostService: hostService,
		content:     content,
		config:      &Config{Base: "/", Mode: mode, EnableActions: true, Authorization: Authorization{Provider: NONE}},
	})
}

func postRollout(t *testing.T, router http.Handler) int {
	req, err := http.NewRequest("POST", "/api/k8s/workloads/default/Deployment/api/restart", nil)
	require.NoError(t, err)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr.Code
}

func Test_handler_rolloutRestart(t *testing.T) {
	restarter := &fakeRestarter{}
	assert.Equal(t, http.StatusNoContent, postRollout(t, rolloutRouter(restarter, "k8s")))
	assert.Equal(t, "default/Deployment/api", restarter.called)
}

func Test_handler_rolloutRestart_errors(t *testing.T) {
	cases := map[string]struct {
		err  error
		code int
	}{
		"hidden by the label filter": {hostservice.ErrWorkloadNotFound, http.StatusNotFound},
		"kind cannot be rolled out":  {fmt.Errorf("CronJob: %w", errors.ErrUnsupported), http.StatusBadRequest},
		"api server failure":         {errors.New("forbidden"), http.StatusInternalServerError},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.code, postRollout(t, rolloutRouter(&fakeRestarter{err: tc.err}, "k8s")))
		})
	}
}

// The route only exists in k8s mode, so server mode never reaches the handler.
func Test_handler_rolloutRestart_notRegisteredOutsideK8s(t *testing.T) {
	restarter := &fakeRestarter{}
	assert.NotEqual(t, http.StatusNoContent, postRollout(t, rolloutRouter(restarter, "server")))
	assert.Empty(t, restarter.called)
}

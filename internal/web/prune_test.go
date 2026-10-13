package web

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/amir20/dozzle/internal/auth"
	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/hostservice"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakePruner stands in for MultiHostService.
type fakePruner struct {
	HostService
	err    error
	host   string
	opts   container.PruneOptions
	called bool
}

func (f *fakePruner) PruneHost(_ context.Context, host string, opts container.PruneOptions) (container.PruneResult, *container.Reclaimable, error) {
	f.called, f.host, f.opts = true, host, opts
	if f.err != nil {
		return container.PruneResult{}, nil, f.err
	}
	return container.PruneResult{SpaceReclaimed: 2048}, &container.Reclaimable{Volumes: 1, VolumesSize: 10}, nil
}

func postPrune(t *testing.T, router http.Handler, body string) *httptest.ResponseRecorder {
	req, err := http.NewRequest("POST", "/api/hosts/local/prune", strings.NewReader(body))
	require.NoError(t, err)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

func Test_handler_pruneHost(t *testing.T) {
	pruner := &fakePruner{}
	rr := postPrune(t, rolloutRouter(pruner, "server"), `{"images":true,"buildCache":true}`)

	require.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "local", pruner.host)
	assert.Equal(t, container.PruneOptions{Images: true, BuildCache: true}, pruner.opts)
	var got pruneResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
	assert.Equal(t, pruneResponse{SpaceReclaimed: 2048, Reclaimable: &container.Reclaimable{Volumes: 1, VolumesSize: 10}}, got)
}

func Test_handler_pruneHost_refusesNothingPicked(t *testing.T) {
	for _, body := range []string{`{}`, `{"images":false}`, `not json`} {
		pruner := &fakePruner{}
		assert.Equal(t, http.StatusBadRequest, postPrune(t, rolloutRouter(pruner, "server"), body).Code, body)
		assert.False(t, pruner.called, body)
	}
}

func Test_handler_pruneHost_errors(t *testing.T) {
	cases := map[string]struct {
		err  error
		code int
	}{
		"unknown host":   {hostservice.ErrHostNotFound, http.StatusNotFound},
		"old agent":      {errors.ErrUnsupported, http.StatusNotImplemented},
		"engine failure": {errors.New("pruning volumes: boom"), http.StatusInternalServerError},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.code, postPrune(t, rolloutRouter(&fakePruner{err: tc.err}, "server"), `{"images":true}`).Code)
		})
	}
}

// k8s has nothing to prune, so the route is never registered there.
func Test_handler_pruneHost_notRegisteredInK8s(t *testing.T) {
	pruner := &fakePruner{}
	assert.NotEqual(t, http.StatusOK, postPrune(t, rolloutRouter(pruner, "k8s"), `{"images":true}`).Code)
	assert.False(t, pruner.called)
}

// A user confined by labels could otherwise delete containers and volumes they can't see.
func Test_handler_pruneHost_refusesRestrictedUser(t *testing.T) {
	pruner := &fakePruner{}
	h := &handler{
		hostService: pruner,
		config:      &Config{Base: "/", Mode: "server", EnableActions: true, Authorization: Authorization{Provider: SIMPLE}},
	}
	req := httptest.NewRequest(http.MethodPost, "/api/hosts/local/prune", strings.NewReader(`{"images":true}`))
	req = req.WithContext(auth.WithUser(req.Context(), auth.User{Username: "amir", Roles: auth.All, ContainerLabels: devLabels}))
	rr := httptest.NewRecorder()
	h.pruneHost(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
	assert.False(t, pruner.called)
}

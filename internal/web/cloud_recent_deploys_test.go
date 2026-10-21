package web

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/cloud"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func recentDeploysRequest(t *testing.T, h *handler, r *http.Request) (int, []cloud.RecentDeploy) {
	t.Helper()
	rr := httptest.NewRecorder()
	h.cloudRecentDeploys(rr, r)
	var body []cloud.RecentDeploy
	if rr.Code == http.StatusOK {
		require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
	}
	return rr.Code, body
}

// Cloud answers for the whole instance, so a filtered user only gets updates to
// containers they can see.
func Test_cloudRecentDeploys_scoped_to_viewer(t *testing.T) {
	h := restrictedHandler(t)
	var gotSince int64
	var gotLimit int32
	h.config.Cloud.GetRecentDeploys = func(_ context.Context, sinceNs int64, limit int32) ([]cloud.RecentDeploy, error) {
		gotSince, gotLimit = sinceNs, limit
		return []cloud.RecentDeploy{
			{Host: "localhost", Name: "dev-allowed", NewID: "dev123", Verdict: &cloud.DeployHit{Verdict: "regressed"}},
			{Host: "localhost", Name: "prod-secret", NewID: "prod456"},
			{Host: "localhost", Name: "dev-allowed", NewID: "gone12345678"},
		}, nil
	}

	code, body := recentDeploysRequest(t, h, asRestrictedUser(httptest.NewRequest(http.MethodGet, "/api/cloud/deploys/recent", nil)))
	require.Equal(t, http.StatusOK, code)
	require.Len(t, body, 1)
	assert.Equal(t, "dev123", body[0].NewID)
	assert.Equal(t, "regressed", body[0].Verdict.Verdict)
	assert.Equal(t, int32(updateFeedLimit), gotLimit)
	assert.InDelta(t, time.Now().Add(-updateFeedWindow).UnixNano(), gotSince, float64(time.Minute))

	code, body = recentDeploysRequest(t, h, httptest.NewRequest(http.MethodGet, "/api/cloud/deploys/recent", nil))
	require.Equal(t, http.StatusOK, code)
	assert.Len(t, body, 3, "an unfiltered user sees every update")
}

func Test_cloudRecentDeploys_errors(t *testing.T) {
	h := restrictedHandler(t)
	req := func() *http.Request { return httptest.NewRequest(http.MethodGet, "/api/cloud/deploys/recent", nil) }

	code, _ := recentDeploysRequest(t, h, req())
	assert.Equal(t, http.StatusServiceUnavailable, code, "no hook wired")

	h.config.Cloud.GetRecentDeploys = func(context.Context, int64, int32) ([]cloud.RecentDeploy, error) {
		return nil, status.Error(codes.Unimplemented, "unknown method")
	}
	code, body := recentDeploysRequest(t, h, req())
	assert.Equal(t, http.StatusOK, code, "an older Cloud is an empty list")
	assert.Empty(t, body)

	h.config.Cloud.GetRecentDeploys = func(context.Context, int64, int32) ([]cloud.RecentDeploy, error) {
		return nil, cloud.ErrNotConfigured
	}
	code, _ = recentDeploysRequest(t, h, req())
	assert.Equal(t, http.StatusServiceUnavailable, code)

	h.config.Cloud.GetRecentDeploys = func(context.Context, int64, int32) ([]cloud.RecentDeploy, error) {
		return nil, errors.New("boom")
	}
	code, _ = recentDeploysRequest(t, h, req())
	assert.Equal(t, http.StatusBadGateway, code)
}

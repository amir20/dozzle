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

func hostMetricsRequest(t *testing.T, h *handler, r *http.Request) (int, cloud.HostMetricResult) {
	t.Helper()
	rr := httptest.NewRecorder()
	h.cloudHostMetrics(rr, r)
	var body cloud.HostMetricResult
	if rr.Code == http.StatusOK {
		require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
	}
	return rr.Code, body
}

// A filtered user gets the same host totals: they are sums, with nothing in
// them that names a container.
func Test_cloudHostMetrics_answers_a_filtered_user(t *testing.T) {
	h := restrictedHandler(t)
	h.config.Cloud.GetHostMetrics = func(context.Context, int64, int64, int32) (*cloud.HostMetricResult, error) {
		return &cloud.HostMetricResult{Hosts: []cloud.HostMetrics{{HostID: "h1"}}}, nil
	}

	code, body := hostMetricsRequest(t, h, asRestrictedUser(httptest.NewRequest(http.MethodGet, "/api/cloud/metrics/hosts", nil)))
	assert.Equal(t, http.StatusOK, code)
	assert.Len(t, body.Hosts, 1)
}

func Test_cloudHostMetrics_passes_the_window(t *testing.T) {
	h := restrictedHandler(t)
	var gotSince, gotUntil int64
	var gotBuckets int32
	h.config.Cloud.GetHostMetrics = func(_ context.Context, since, until int64, buckets int32) (*cloud.HostMetricResult, error) {
		gotSince, gotUntil, gotBuckets = since, until, buckets
		return &cloud.HostMetricResult{Hosts: []cloud.HostMetrics{{HostID: "h1", Points: []cloud.HostMetricPoint{{TS: 1, CPU: 5}}}}}, nil
	}

	code, body := hostMetricsRequest(t, h, httptest.NewRequest(http.MethodGet, "/api/cloud/metrics/hosts?window=24h&buckets=200", nil))
	require.Equal(t, http.StatusOK, code)
	require.Len(t, body.Hosts, 1)
	assert.Equal(t, "h1", body.Hosts[0].HostID)
	assert.Equal(t, int32(200), gotBuckets)
	assert.InDelta(t, (24 * time.Hour).Nanoseconds(), gotUntil-gotSince, float64(time.Second))
}

func Test_cloudHostMetrics_errors(t *testing.T) {
	h := restrictedHandler(t)
	req := func() *http.Request { return httptest.NewRequest(http.MethodGet, "/api/cloud/metrics/hosts", nil) }

	code, _ := hostMetricsRequest(t, h, req())
	assert.Equal(t, http.StatusServiceUnavailable, code, "no hook wired")

	h.config.Cloud.GetHostMetrics = func(context.Context, int64, int64, int32) (*cloud.HostMetricResult, error) {
		return nil, status.Error(codes.Unimplemented, "unknown method")
	}
	code, body := hostMetricsRequest(t, h, req())
	assert.Equal(t, http.StatusOK, code, "an older Cloud is no hosts")
	assert.NotNil(t, body.Hosts)
	assert.Empty(t, body.Hosts)

	h.config.Cloud.GetHostMetrics = func(context.Context, int64, int64, int32) (*cloud.HostMetricResult, error) {
		return nil, cloud.ErrNotConfigured
	}
	code, _ = hostMetricsRequest(t, h, req())
	assert.Equal(t, http.StatusServiceUnavailable, code)

	h.config.Cloud.GetHostMetrics = func(context.Context, int64, int64, int32) (*cloud.HostMetricResult, error) {
		return nil, errors.New("boom")
	}
	code, _ = hostMetricsRequest(t, h, req())
	assert.Equal(t, http.StatusBadGateway, code)
}

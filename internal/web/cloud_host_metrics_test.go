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

// A filtered user gets the same host totals, which are sums, but not the
// top-container lists: those name containers they may not be allowed to see.
func Test_cloudHostMetrics_answers_a_filtered_user_without_names(t *testing.T) {
	h := restrictedHandler(t)
	h.config.Cloud.GetHostMetrics = func(context.Context, cloud.HostMetricsQuery) (*cloud.HostMetricResult, error) {
		return &cloud.HostMetricResult{Hosts: []cloud.HostMetrics{{
			HostID:     "h1",
			Containers: 3,
			Top:        &cloud.HostTop{CPU: []cloud.HostTopContainer{{Name: "secret-db", Avg: 50, Peak: 90}}},
		}}}, nil
	}

	code, body := hostMetricsRequest(t, h, asRestrictedUser(httptest.NewRequest(http.MethodGet, "/api/cloud/metrics/hosts", nil)))
	assert.Equal(t, http.StatusOK, code)
	require.Len(t, body.Hosts, 1)
	assert.Equal(t, int32(3), body.Hosts[0].Containers)
	assert.Nil(t, body.Hosts[0].Top)
}

// Everyone else gets the lists as Cloud sent them.
func Test_cloudHostMetrics_keeps_names_for_an_unfiltered_user(t *testing.T) {
	h := restrictedHandler(t)
	h.config.Cloud.GetHostMetrics = func(context.Context, cloud.HostMetricsQuery) (*cloud.HostMetricResult, error) {
		return &cloud.HostMetricResult{Hosts: []cloud.HostMetrics{{
			HostID: "h1",
			Top:    &cloud.HostTop{CPU: []cloud.HostTopContainer{{Name: "api", Avg: 5, Peak: 9}}},
		}}}, nil
	}

	code, body := hostMetricsRequest(t, h, httptest.NewRequest(http.MethodGet, "/api/cloud/metrics/hosts", nil))
	assert.Equal(t, http.StatusOK, code)
	require.Len(t, body.Hosts, 1)
	require.NotNil(t, body.Hosts[0].Top)
	assert.Equal(t, "api", body.Hosts[0].Top.CPU[0].Name)
}

func Test_cloudHostMetrics_passes_the_window(t *testing.T) {
	h := restrictedHandler(t)
	var got cloud.HostMetricsQuery
	h.config.Cloud.GetHostMetrics = func(_ context.Context, q cloud.HostMetricsQuery) (*cloud.HostMetricResult, error) {
		got = q
		return &cloud.HostMetricResult{Hosts: []cloud.HostMetrics{{HostID: "h1", Points: []cloud.HostMetricPoint{{TS: 1, CPU: 5}}}}}, nil
	}

	code, body := hostMetricsRequest(t, h, httptest.NewRequest(http.MethodGet, "/api/cloud/metrics/hosts?window=24h&buckets=200", nil))
	require.Equal(t, http.StatusOK, code)
	require.Len(t, body.Hosts, 1)
	assert.Equal(t, "h1", body.Hosts[0].HostID)
	assert.Equal(t, int32(200), got.Buckets)
	assert.InDelta(t, (24 * time.Hour).Nanoseconds(), got.UntilNs-got.SinceNs, float64(time.Second))
	// No list asked for, none computed.
	assert.Empty(t, got.Top)
	assert.False(t, got.TopOnly)
}

// The panel asks for the one list on screen, and only that list on a tab switch.
// An unknown metric asks for none rather than reaching Cloud as-is.
func Test_cloudHostMetrics_asks_for_one_list(t *testing.T) {
	h := restrictedHandler(t)
	var got cloud.HostMetricsQuery
	h.config.Cloud.GetHostMetrics = func(_ context.Context, q cloud.HostMetricsQuery) (*cloud.HostMetricResult, error) {
		got = q
		return &cloud.HostMetricResult{}, nil
	}

	hostMetricsRequest(t, h, httptest.NewRequest(http.MethodGet, "/api/cloud/metrics/hosts?top=memory&topOnly=1", nil))
	assert.Equal(t, "memory", got.Top)
	assert.True(t, got.TopOnly)

	hostMetricsRequest(t, h, httptest.NewRequest(http.MethodGet, "/api/cloud/metrics/hosts?top=bogus", nil))
	assert.Empty(t, got.Top)
}

// A filtered reader never has a list computed: the charts ask for none, and a
// list-only request is answered empty without a call to Cloud.
func Test_cloudHostMetrics_never_asks_lists_for_a_filtered_user(t *testing.T) {
	h := restrictedHandler(t)
	calls := 0
	var got cloud.HostMetricsQuery
	h.config.Cloud.GetHostMetrics = func(_ context.Context, q cloud.HostMetricsQuery) (*cloud.HostMetricResult, error) {
		calls++
		got = q
		return &cloud.HostMetricResult{}, nil
	}

	hostMetricsRequest(t, h, asRestrictedUser(httptest.NewRequest(http.MethodGet, "/api/cloud/metrics/hosts?top=cpu", nil)))
	assert.Equal(t, 1, calls)
	assert.Empty(t, got.Top)

	code, body := hostMetricsRequest(t, h, asRestrictedUser(httptest.NewRequest(http.MethodGet, "/api/cloud/metrics/hosts?top=cpu&topOnly=1", nil)))
	assert.Equal(t, http.StatusOK, code)
	assert.Empty(t, body.Hosts)
	assert.Equal(t, 1, calls)
}

func Test_cloudHostMetrics_errors(t *testing.T) {
	h := restrictedHandler(t)
	req := func() *http.Request { return httptest.NewRequest(http.MethodGet, "/api/cloud/metrics/hosts", nil) }

	code, _ := hostMetricsRequest(t, h, req())
	assert.Equal(t, http.StatusServiceUnavailable, code, "no hook wired")

	h.config.Cloud.GetHostMetrics = func(context.Context, cloud.HostMetricsQuery) (*cloud.HostMetricResult, error) {
		return nil, status.Error(codes.Unimplemented, "unknown method")
	}
	code, body := hostMetricsRequest(t, h, req())
	assert.Equal(t, http.StatusOK, code, "an older Cloud is no hosts")
	assert.NotNil(t, body.Hosts)
	assert.Empty(t, body.Hosts)

	h.config.Cloud.GetHostMetrics = func(context.Context, cloud.HostMetricsQuery) (*cloud.HostMetricResult, error) {
		return nil, cloud.ErrNotConfigured
	}
	code, _ = hostMetricsRequest(t, h, req())
	assert.Equal(t, http.StatusServiceUnavailable, code)

	h.config.Cloud.GetHostMetrics = func(context.Context, cloud.HostMetricsQuery) (*cloud.HostMetricResult, error) {
		return nil, errors.New("boom")
	}
	code, _ = hostMetricsRequest(t, h, req())
	assert.Equal(t, http.StatusBadGateway, code)
}

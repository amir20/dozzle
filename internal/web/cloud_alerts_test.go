package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/amir20/dozzle/internal/cloud"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type alertCall struct {
	containerIDs     []string
	hostID           string
	from, to         int64
	limit            int32
	includeFollowUps bool
	includeEvents    bool
}

// alertHandler builds a handler wired to a stub GetAlerts, capturing what the
// HTTP layer forwarded. A nil fn leaves the hook unset, which is how "cloud not
// wired" is expressed.
func alertHandler(t *testing.T, fn func(*alertCall) (*cloud.AlertResult, error)) (*handler, *alertCall) {
	t.Helper()
	got := &alertCall{}
	h := &handler{config: &Config{}}
	if fn != nil {
		h.config.Cloud.GetAlerts = func(ctx context.Context, containerIDs []string, hostID string, fromNs, toNs int64, limit int32, includeFollowUps, includeEvents bool) (*cloud.AlertResult, error) {
			*got = alertCall{containerIDs, hostID, fromNs, toNs, limit, includeFollowUps, includeEvents}
			return fn(got)
		}
	}
	return h, got
}

func doAlerts(h *handler, query string) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	h.cloudAlerts(rr, httptest.NewRequest(http.MethodGet, "/api/cloud/alerts?"+query, nil))
	return rr
}

func okResult(*alertCall) (*cloud.AlertResult, error) {
	return &cloud.AlertResult{Hits: []cloud.AlertHit{{AlertID: "6teOh3RH", ContainerID: "abc"}}}, nil
}

func TestCloudAlerts_UnwiredIs503(t *testing.T) {
	h, _ := alertHandler(t, nil)
	rr := doAlerts(h, "containerIds=abc&from=1&to=2")
	require.Equal(t, http.StatusServiceUnavailable, rr.Code)
}

// Alerts must NOT be gated on the streamLogs opt-in the way log search is:
// they live in Cloud's database rather than the log store. hostService is nil
// here, so if the handler ever reached for CloudConfig() this would panic.
func TestCloudAlerts_DoesNotRequireStreamLogs(t *testing.T) {
	h, _ := alertHandler(t, okResult)
	rr := doAlerts(h, "containerIds=abc&from=1&to=2")
	require.Equal(t, http.StatusOK, rr.Code)
}

func TestCloudAlerts_ForwardsParams(t *testing.T) {
	h, got := alertHandler(t, okResult)
	rr := doAlerts(h, "containerIds=abc,%20def&hostId=h1&from=100&to=200&limit=7&followUps=1")

	require.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, []string{"abc", "def"}, got.containerIDs, "ids should be split and trimmed")
	require.Equal(t, "h1", got.hostID)
	require.Equal(t, int64(100), got.from)
	require.Equal(t, int64(200), got.to)
	require.Equal(t, int32(7), got.limit)
	require.True(t, got.includeFollowUps)
}

// Events are opt-in: they are far more numerous than alerts, so a caller that
// only wants incidents must not pay for them.
func TestCloudAlerts_EventsAreOptIn(t *testing.T) {
	h, got := alertHandler(t, okResult)

	require.Equal(t, http.StatusOK, doAlerts(h, "containerIds=abc&from=1&to=2").Code)
	require.False(t, got.includeEvents)

	require.Equal(t, http.StatusOK, doAlerts(h, "containerIds=abc&from=1&to=2&events=1").Code)
	require.True(t, got.includeEvents)
}

// An empty container list is a no-op, not an error: the caller's merge path
// stays uniform and cloud is never asked a pointless question.
func TestCloudAlerts_EmptyContainersReturnsEmpty(t *testing.T) {
	called := false
	h, _ := alertHandler(t, func(*alertCall) (*cloud.AlertResult, error) {
		called = true
		return &cloud.AlertResult{}, nil
	})

	rr := doAlerts(h, "containerIds=&from=1&to=2")
	require.Equal(t, http.StatusOK, rr.Code)
	require.False(t, called, "cloud should not be queried for an empty container list")

	var result cloud.AlertResult
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &result))
	require.Empty(t, result.Hits)
}

func TestCloudAlerts_RejectsBadWindow(t *testing.T) {
	for name, query := range map[string]string{
		"missing from":   "containerIds=abc&to=2",
		"missing to":     "containerIds=abc&from=1",
		"to before from": "containerIds=abc&from=5&to=2",
		"to equals from": "containerIds=abc&from=5&to=5",
		"unparseable":    "containerIds=abc&from=nope&to=2",
	} {
		t.Run(name, func(t *testing.T) {
			h, _ := alertHandler(t, okResult)
			require.Equal(t, http.StatusBadRequest, doAlerts(h, query).Code)
		})
	}
}

func TestCloudAlerts_CapsLimitAndContainers(t *testing.T) {
	h, got := alertHandler(t, okResult)
	require.Equal(t, http.StatusOK, doAlerts(h, "containerIds=abc&from=1&to=2&limit=9999").Code)
	require.Equal(t, int32(200), got.limit, "limit should be capped, mirroring cloud")

	var ids strings.Builder
	for i := 0; i <= maxAlertContainers; i++ {
		ids.WriteString("c,")
	}
	require.Equal(t, http.StatusBadRequest, doAlerts(h, "containerIds="+ids.String()+"&from=1&to=2").Code)
}

func TestCloudAlerts_ErrorMapping(t *testing.T) {
	for name, tc := range map[string]struct {
		err  error
		want int
	}{
		"not configured": {cloud.ErrNotConfigured, http.StatusServiceUnavailable},
		"deadline":       {status.Error(codes.DeadlineExceeded, "too slow"), http.StatusGatewayTimeout},
		"ctx deadline":   {context.DeadlineExceeded, http.StatusGatewayTimeout},
		"other":          {status.Error(codes.Internal, "boom"), http.StatusBadGateway},
	} {
		t.Run(name, func(t *testing.T) {
			h, _ := alertHandler(t, func(*alertCall) (*cloud.AlertResult, error) { return nil, tc.err })
			require.Equal(t, tc.want, doAlerts(h, "containerIds=abc&from=1&to=2").Code)
		})
	}
}

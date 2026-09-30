package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/amir20/dozzle/internal/cloud"
	"github.com/amir20/dozzle/internal/notification"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func streaming(on bool) *notification.CloudConfig {
	return &notification.CloudConfig{APIKey: "key", StreamLogs: &on}
}

// patternHandler wires the patterns hook and a linked cloud config over the
// restricted two-container fixture, capturing what reached the hook.
func patternHandler(t *testing.T, cc *notification.CloudConfig, fn func([]cloud.PatternLine) ([]cloud.PatternContext, error)) (*handler, *[]cloud.PatternLine) {
	t.Helper()
	h := restrictedHandler(t)
	h.hostService = &cloudLinkedService{HostService: h.hostService, cc: cc}
	got := &[]cloud.PatternLine{}
	h.config.Cloud.GetPatternContext = func(ctx context.Context, lines []cloud.PatternLine, fromNs, toNs int64) ([]cloud.PatternContext, error) {
		*got = lines
		return fn(lines)
	}
	return h, got
}

func doPatterns(h *handler, r *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	h.cloudPatterns(rr, r)
	return rr
}

func patternReq(body string) *http.Request {
	return httptest.NewRequest(http.MethodPost, "/api/cloud/patterns", strings.NewReader(body))
}

const twoLines = `{"lines":[{"containerId":"dev123","logId":1},{"containerId":"prod456","logId":2}],"from":1,"to":2}`

func TestCloudPatterns_UnwiredIs503(t *testing.T) {
	h := &handler{config: &Config{}}
	require.Equal(t, http.StatusServiceUnavailable, doPatterns(h, patternReq(twoLines)).Code)
}

// The lines are looked up in what was streamed; without the opt-in there is
// nothing to find, and Cloud is never asked.
func TestCloudPatterns_NeedsStreamLogs(t *testing.T) {
	called := false
	h, _ := patternHandler(t, streaming(false), func([]cloud.PatternLine) ([]cloud.PatternContext, error) {
		called = true
		return nil, nil
	})
	require.Equal(t, http.StatusNoContent, doPatterns(h, patternReq(twoLines)).Code)
	assert.False(t, called)
}

func TestCloudPatterns_DropsOutOfScopeLines(t *testing.T) {
	h, got := patternHandler(t, streaming(true), func([]cloud.PatternLine) ([]cloud.PatternContext, error) {
		return []cloud.PatternContext{{ContainerID: "dev123", LogIDs: []uint32{1}, Status: "new"}}, nil
	})
	rr := doPatterns(h, asRestrictedUser(patternReq(twoLines)))
	require.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, []cloud.PatternLine{{ContainerID: "dev123", LogID: 1}}, *got)

	var body struct {
		Patterns []cloud.PatternContext `json:"patterns"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
	require.Len(t, body.Patterns, 1)
	assert.Equal(t, "new", body.Patterns[0].Status)
}

func TestCloudPatterns_RejectsBadInput(t *testing.T) {
	h, _ := patternHandler(t, streaming(true), func([]cloud.PatternLine) ([]cloud.PatternContext, error) { return nil, nil })
	for name, body := range map[string]string{
		"not json":  `nope`,
		"no window": `{"lines":[{"containerId":"dev123","logId":1}]}`,
		"backwards": `{"lines":[{"containerId":"dev123","logId":1}],"from":5,"to":2}`,
		"too many":  `{"lines":[` + strings.TrimSuffix(strings.Repeat(`{"containerId":"dev123","logId":1},`, maxPatternLines+1), ",") + `],"from":1,"to":2}`,
	} {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, http.StatusBadRequest, doPatterns(h, patternReq(body)).Code)
		})
	}
}

func TestCloudPatterns_ErrorMapping(t *testing.T) {
	for name, tc := range map[string]struct {
		err  error
		want int
	}{
		"not configured": {cloud.ErrNotConfigured, http.StatusServiceUnavailable},
		// A Cloud that predates the RPC shows nothing, quietly.
		"unimplemented": {status.Error(codes.Unimplemented, "unknown method"), http.StatusNoContent},
		"deadline":      {status.Error(codes.DeadlineExceeded, "slow"), http.StatusGatewayTimeout},
		"other":         {status.Error(codes.Internal, "boom"), http.StatusBadGateway},
	} {
		t.Run(name, func(t *testing.T) {
			h, _ := patternHandler(t, streaming(true), func([]cloud.PatternLine) ([]cloud.PatternContext, error) { return nil, tc.err })
			require.Equal(t, tc.want, doPatterns(h, patternReq(twoLines)).Code)
		})
	}
}

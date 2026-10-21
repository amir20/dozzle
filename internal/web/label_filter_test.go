package web

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/auth"
	"github.com/amir20/dozzle/internal/cloud"
	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/container/docker"
	"github.com/amir20/dozzle/internal/hostservice"
	"github.com/amir20/dozzle/internal/notification"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	devContainer  = container.Container{ID: "dev123", Host: "localhost", Name: "dev-allowed", State: "running", Labels: map[string]string{"env": "dev"}}
	prodContainer = container.Container{ID: "prod456", Host: "localhost", Name: "prod-secret", State: "running", Labels: map[string]string{"env": "prod"}}
	devLabels     = container.ContainerLabels{"env": []string{"dev"}}
)

// restrictedHandler wires a handler backed by two containers where the caller
// may only see the dev one.
func restrictedHandler(t *testing.T) *handler {
	t.Helper()
	client := new(MockedClient)
	client.On("Host").Return(container.Host{ID: "localhost"})
	client.On("ContainerEvents", mock.Anything, mock.AnythingOfType("chan<- container.ContainerEvent")).Return(nil)
	client.On("ListContainers", mock.Anything, devLabels).Return([]container.Container{devContainer}, nil)
	client.On("ListContainers", mock.Anything, mock.Anything).Return([]container.Container{devContainer, prodContainer}, nil)
	client.On("FindContainer", mock.Anything, "dev123").Return(devContainer, nil)
	client.On("FindContainer", mock.Anything, "prod456").Return(prodContainer, nil)
	client.On("ContainerLogsBetweenDates", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(io.NopCloser(strings.NewReader("")), nil)
	client.On("ContainerLogs", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(io.NopCloser(strings.NewReader("")), io.EOF)

	manager := hostservice.NewRetriableClientManager(nil, nil, 3*time.Second, tls.Certificate{}, docker.NewService(client, container.ContainerLabels{}))
	return &handler{
		hostService: hostservice.NewMultiHostService(manager, 3*time.Second),
		config:      &Config{Base: "/", Authorization: Authorization{Provider: SIMPLE}},
	}
}

// cloudLinkedService pretends cloud is linked without touching the persister.
type cloudLinkedService struct {
	HostService
	cc *notification.CloudConfig
}

func (s *cloudLinkedService) CloudConfig() *notification.CloudConfig { return s.cc }

func asRestrictedUser(r *http.Request) *http.Request {
	return r.WithContext(auth.WithUser(context.Background(), auth.User{ContainerLabels: devLabels}))
}

func Test_debugStore_respects_user_labels(t *testing.T) {
	h := restrictedHandler(t)
	rr := httptest.NewRecorder()

	h.debugStore(rr, asRestrictedUser(httptest.NewRequest(http.MethodGet, "/api/debug/store", nil)))

	require.Equal(t, http.StatusOK, rr.Code)
	var body struct {
		Containers []container.Container `json:"containers"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
	require.Len(t, body.Containers, 1)
	assert.Equal(t, "dev-allowed", body.Containers[0].Name)
}

func Test_cloudSearchLogs_withholds_volume_from_restricted_users(t *testing.T) {
	h := restrictedHandler(t)
	var asked bool
	h.config.Cloud.SearchLogs = func(ctx context.Context, query string, limit int32, hostID, containerID string, before int64, volume, regex bool) (*cloud.SearchLogResult, error) {
		asked = volume
		return &cloud.SearchLogResult{
			Volume: []cloud.SearchVolumeBucket{{TimestampNs: 1, Level: "info", Count: 9}},
			StepNs: 1,
			Total:  new(int64(9)),
		}, nil
	}
	h.hostService = &cloudLinkedService{HostService: h.hostService, cc: &notification.CloudConfig{APIKey: "key"}}

	rr := httptest.NewRecorder()
	h.cloudSearchLogs(rr, asRestrictedUser(httptest.NewRequest(http.MethodGet, "/api/cloud/search/logs?q=x&volume=1", nil)))

	require.Equal(t, http.StatusOK, rr.Code)
	assert.False(t, asked, "a restricted caller's search must not ask Cloud for instance-wide counts")
	var result cloud.SearchLogResult
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &result))
	assert.Empty(t, result.Volume)
	assert.Nil(t, result.Total, "the total counts the whole instance too")
}

func Test_cloudSearchLogs_forwards_regex_intent(t *testing.T) {
	h := restrictedHandler(t)
	var got []bool
	h.config.Cloud.SearchLogs = func(ctx context.Context, query string, limit int32, hostID, containerID string, before int64, volume, regex bool) (*cloud.SearchLogResult, error) {
		got = append(got, regex)
		return &cloud.SearchLogResult{}, nil
	}
	h.hostService = &cloudLinkedService{HostService: h.hostService, cc: &notification.CloudConfig{APIKey: "key"}}

	for _, url := range []string{"/api/cloud/search/logs?q=x", "/api/cloud/search/logs?q=x&regex=1"} {
		rr := httptest.NewRecorder()
		h.cloudSearchLogs(rr, asRestrictedUser(httptest.NewRequest(http.MethodGet, url, nil)))
		require.Equal(t, http.StatusOK, rr.Code)
	}
	assert.Equal(t, []bool{false, true}, got)
}

func Test_cloudSearchLogs_drops_out_of_scope_hits(t *testing.T) {
	h := restrictedHandler(t)
	h.config.Cloud.SearchLogs = func(ctx context.Context, query string, limit int32, hostID, containerID string, before int64, volume, regex bool) (*cloud.SearchLogResult, error) {
		return &cloud.SearchLogResult{Hits: []cloud.SearchLogHit{
			{ContainerID: "dev123", ContainerName: "dev-allowed", Message: "hello"},
			{ContainerID: "prod456", ContainerName: "prod-secret", Message: "password=hunter2"},
		}}, nil
	}
	h.hostService = &cloudLinkedService{HostService: h.hostService, cc: &notification.CloudConfig{APIKey: "key"}}

	rr := httptest.NewRecorder()
	h.cloudSearchLogs(rr, asRestrictedUser(httptest.NewRequest(http.MethodGet, "/api/cloud/search/logs?q=password", nil)))

	require.Equal(t, http.StatusOK, rr.Code)
	var result cloud.SearchLogResult
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &result))
	require.Len(t, result.Hits, 1, "out-of-scope log hits should be dropped")
	assert.Equal(t, "dev123", result.Hits[0].ContainerID)
}

func Test_cloudAlerts_filters_requested_containers(t *testing.T) {
	h := restrictedHandler(t)
	var asked []string
	h.config.Cloud.GetAlerts = func(ctx context.Context, containerIDs []string, hostID string, fromNs, toNs int64, limit int32, includeFollowUps, includeEvents bool) (*cloud.AlertResult, error) {
		asked = containerIDs
		return &cloud.AlertResult{Hits: []cloud.AlertHit{}}, nil
	}

	rr := httptest.NewRecorder()
	h.cloudAlerts(rr, asRestrictedUser(httptest.NewRequest(http.MethodGet, "/api/cloud/alerts?containerIds=dev123,prod456&from=1&to=2", nil)))

	require.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, []string{"dev123"}, asked, "only in-scope container ids should reach cloud")
}

func Test_cloudAlerts_all_out_of_scope_returns_empty(t *testing.T) {
	h := restrictedHandler(t)
	called := false
	h.config.Cloud.GetAlerts = func(ctx context.Context, containerIDs []string, hostID string, fromNs, toNs int64, limit int32, includeFollowUps, includeEvents bool) (*cloud.AlertResult, error) {
		called = true
		return &cloud.AlertResult{Hits: []cloud.AlertHit{{AlertID: "leak"}}}, nil
	}

	rr := httptest.NewRecorder()
	h.cloudAlerts(rr, asRestrictedUser(httptest.NewRequest(http.MethodGet, "/api/cloud/alerts?containerIds=prod456&from=1&to=2", nil)))

	require.Equal(t, http.StatusOK, rr.Code)
	assert.False(t, called, "cloud should not be queried for out-of-scope containers")
	assert.NotContains(t, rr.Body.String(), "leak")
}

// A stream names its containers itself, so one outside the user's labels must not
// leak its update record (name, image, digests) through a container-update event.
func Test_streamContainerLogs_no_update_marker_out_of_scope(t *testing.T) {
	prev := container.Updates
	container.Updates = container.NewUpdateRecords()
	t.Cleanup(func() { container.Updates = prev })
	at := time.Now()
	container.Updates.Add(container.UpdateRecord{Host: "localhost", Name: "dev-allowed", NewID: "dev123", At: at})
	container.Updates.Add(container.UpdateRecord{Host: "localhost", Name: "prod-secret", NewID: "prod456", At: at})

	for _, tc := range []struct {
		query  string
		marker bool
	}{
		{query: "container=localhost/prod456", marker: false},
		{query: "container=localhost/dev123&container=localhost/prod456", marker: true},
	} {
		t.Run(tc.query, func(t *testing.T) {
			h := restrictedHandler(t)
			ctx, cancel := context.WithCancel(context.Background())
			req := httptest.NewRequest(http.MethodGet, "/api/containers/logs/stream?stdout=1&stderr=1&"+tc.query, nil).
				WithContext(auth.WithUser(ctx, auth.User{ContainerLabels: devLabels}))
			q := req.URL.Query()
			addAllLogLevels(q)
			req.URL.RawQuery = q.Encode()

			rr := httptest.NewRecorder()
			done := make(chan struct{})
			go func() {
				defer close(done)
				h.streamContainerLogs(rr, req)
			}()
			time.Sleep(300 * time.Millisecond)
			cancel()
			<-done

			body := rr.Body.String()
			assert.NotContains(t, body, "prod-secret")
			assert.NotContains(t, body, "prod456")
			assert.Equal(t, tc.marker, strings.Contains(body, "container-update"))
		})
	}
}

// The megaphone's update feed rides the events stream, so it is gated like the
// containers: an update to a container outside the user's labels never reaches
// them, and an older update of a visible container is shown by its name once
// the container it created is gone.
func Test_streamEvents_update_feed_out_of_scope(t *testing.T) {
	prev := container.Updates
	container.Updates = container.NewUpdateRecords()
	t.Cleanup(func() { container.Updates = prev })
	at := time.Now()
	container.Updates.Add(container.UpdateRecord{Host: "localhost", Name: "dev-allowed", NewID: "olddev", ImageRef: "dev:first", At: at.Add(-time.Hour)})
	container.Updates.Add(container.UpdateRecord{Host: "localhost", Name: "dev-allowed", NewID: "dev123", ImageRef: "dev:second", At: at})
	container.Updates.Add(container.UpdateRecord{Host: "localhost", Name: "prod-secret", NewID: "prod456", At: at})
	container.Updates.Add(container.UpdateRecord{Host: "localhost", Name: "dev-allowed", NewID: "ancient", ImageRef: "dev:ancient", At: at.Add(-8 * 24 * time.Hour)})

	h := restrictedHandler(t)
	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodGet, "/api/events/stream", nil).
		WithContext(auth.WithUser(ctx, auth.User{ContainerLabels: devLabels}))

	rr := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		defer close(done)
		h.streamEvents(rr, req)
	}()
	time.Sleep(300 * time.Millisecond)
	cancel()
	<-done

	body := rr.Body.String()
	assert.Contains(t, body, "dev:first")
	assert.Contains(t, body, "dev:second")
	assert.NotContains(t, body, "dev:ancient", "an update older than the feed window is not sent")
	assert.NotContains(t, body, "prod-secret")
	assert.NotContains(t, body, "prod456")
}

func Test_recentUpdates_caps_newest(t *testing.T) {
	now := time.Now()
	var records []container.UpdateRecord
	for i := range 30 {
		records = append(records, container.UpdateRecord{Name: fmt.Sprint(i), At: now.Add(time.Duration(i-30) * time.Minute)})
	}
	got := recentUpdates(records, now, func(container.UpdateRecord) bool { return true })
	require.Len(t, got, updateFeedLimit)
	assert.Equal(t, "10", got[0].Name)
	assert.Equal(t, "29", got[len(got)-1].Name)
}

func Test_requireNotificationsRole(t *testing.T) {
	h := restrictedHandler(t)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusTeapot) })

	serve := func(user auth.User) int {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/notifications/rules", nil).
			WithContext(auth.WithUser(context.Background(), user))
		h.requireNotificationsRole(next).ServeHTTP(rr, req)
		return rr.Code
	}

	assert.Equal(t, http.StatusForbidden, serve(auth.User{Roles: auth.Download}))
	assert.Equal(t, http.StatusTeapot, serve(auth.User{Roles: auth.Notifications}))
	// Unset roles parse to All, which is what a user with only a filter gets.
	assert.Equal(t, http.StatusTeapot, serve(auth.User{Roles: auth.All, ContainerLabels: devLabels}))
}

func Test_requireCloudRole(t *testing.T) {
	h := restrictedHandler(t)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusTeapot) })

	serve := func(ctx context.Context) int {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/cloud/callback?token=abc", nil).WithContext(ctx)
		h.requireCloudRole(next).ServeHTTP(rr, req)
		return rr.Code
	}
	withUser := func(user auth.User) context.Context {
		return auth.WithUser(context.Background(), user)
	}

	// A filtered user without the role must not be able to link the instance.
	assert.Equal(t, http.StatusForbidden, serve(withUser(auth.User{Roles: auth.Download, ContainerLabels: devLabels})))
	assert.Equal(t, http.StatusForbidden, serve(withUser(auth.User{Roles: auth.None})))
	// No user at all (auth enabled) fails closed.
	assert.Equal(t, http.StatusForbidden, serve(context.Background()))

	assert.Equal(t, http.StatusTeapot, serve(withUser(auth.User{Roles: auth.Cloud})))
	// Unset roles parse to All, so existing users keep cloud access.
	assert.Equal(t, http.StatusTeapot, serve(withUser(auth.User{Roles: auth.All, ContainerLabels: devLabels})))
}

// Cloud words InvalidArgument and ResourceExhausted for the person who typed
// the query, so those messages reach the UI verbatim — unwrapped from the
// client's "cloud: search: %w". Anything else stays generic.
func Test_cloudSearchLogs_passes_query_errors_through(t *testing.T) {
	for name, tc := range map[string]struct {
		err      error
		wantCode int
		wantMsg  string
	}{
		"invalid":   {status.Error(codes.InvalidArgument, "Wrap it in double quotes."), http.StatusBadRequest, "Wrap it in double quotes."},
		"too broad": {status.Error(codes.ResourceExhausted, "Narrow it to a container."), http.StatusUnprocessableEntity, "Narrow it to a container."},
		"internal":  {status.Error(codes.Internal, "vl exploded"), http.StatusBadGateway, "cloud search failed"},
	} {
		t.Run(name, func(t *testing.T) {
			h := restrictedHandler(t)
			h.config.Cloud.SearchLogs = func(context.Context, string, int32, string, string, int64, bool, bool) (*cloud.SearchLogResult, error) {
				return nil, fmt.Errorf("cloud: search: %w", tc.err)
			}
			h.hostService = &cloudLinkedService{HostService: h.hostService, cc: &notification.CloudConfig{APIKey: "key"}}

			rr := httptest.NewRecorder()
			h.cloudSearchLogs(rr, httptest.NewRequest(http.MethodGet, "/api/cloud/search/logs?q=x", nil))

			require.Equal(t, tc.wantCode, rr.Code)
			var body map[string]string
			require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
			assert.Equal(t, tc.wantMsg, body["error"])
		})
	}
}

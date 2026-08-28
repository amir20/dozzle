package web

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/auth"
	"github.com/amir20/dozzle/internal/cloud"
	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/notification"
	docker_support "github.com/amir20/dozzle/internal/support/docker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var (
	devContainer  = container.Container{ID: "dev123", Name: "dev-allowed", State: "running", Labels: map[string]string{"env": "dev"}}
	prodContainer = container.Container{ID: "prod456", Name: "prod-secret", State: "running", Labels: map[string]string{"env": "prod"}}
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

	manager := docker_support.NewRetriableClientManager(nil, 3*time.Second, tls.Certificate{}, docker_support.NewDockerClientService(client, container.ContainerLabels{}))
	return &handler{
		hostService: docker_support.NewMultiHostService(manager, 3*time.Second),
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

func Test_cloudSearchLogs_drops_out_of_scope_hits(t *testing.T) {
	h := restrictedHandler(t)
	h.config.Cloud.SearchLogs = func(ctx context.Context, query string, limit int32, hostID, containerID string, before int64) (*cloud.SearchLogResult, error) {
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

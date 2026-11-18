package web

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/amir20/dozzle/internal/auth"
	"github.com/amir20/dozzle/internal/cloud"
	"github.com/amir20/dozzle/internal/container"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type findingsCall struct {
	names   []string
	status  string
	ignored bool
	limit   int32
	calls   int
}

func stubFindings(h *handler, result *cloud.FindingsResult, err error) *findingsCall {
	got := &findingsCall{}
	h.config.Cloud.GetFindings = func(_ context.Context, names []string, status string, ignored bool, limit int32) (*cloud.FindingsResult, error) {
		got.names, got.status, got.ignored, got.limit = names, status, ignored, limit
		got.calls++
		return result, err
	}
	return got
}

func findingsRequest(t *testing.T, h *handler, r *http.Request) (int, cloud.FindingsResult) {
	t.Helper()
	rr := httptest.NewRecorder()
	h.cloudFindings(rr, r)
	var body cloud.FindingsResult
	if rr.Code == http.StatusOK {
		require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
		require.NotNil(t, body.Findings, "findings must encode as [] not null")
	}
	return rr.Code, body
}

// asUnfilteredUser signs the request in as a user with no filter. Simple auth
// always puts a user on the request, and resolveLabels expects one.
func asUnfilteredUser(r *http.Request) *http.Request {
	return r.WithContext(auth.WithUser(context.Background(), auth.User{Roles: auth.All}))
}

func findingsURL(query string) string {
	return "/api/cloud/findings?" + query
}

// A filtered user gets no findings and never sends a name: a finding is keyed
// by names that can belong to containers their filter hides. Cloud is still
// asked, with no names, so they get the instance's review schedule.
func Test_cloudFindings_filtered_user_gets_schedule_only(t *testing.T) {
	h := restrictedHandler(t)
	got := stubFindings(h, &cloud.FindingsResult{Findings: []cloud.Finding{
		{ID: "k3Xa9", ContainerNames: []string{"dev-allowed"}},
	}, LastScanAt: 100, NextScanAt: 200}, nil)

	q := url.Values{"container": {"localhost/dev123"}}
	code, body := findingsRequest(t, h, asRestrictedUser(httptest.NewRequest(http.MethodGet, findingsURL(q.Encode()), nil)))

	require.Equal(t, http.StatusOK, code)
	assert.Empty(t, body.Findings, "even a finding Cloud returns is not shown")
	assert.False(t, body.CanIgnore)
	assert.Equal(t, 1, got.calls)
	assert.Empty(t, got.names)
	assert.Equal(t, int64(100), body.LastScanAt)
	assert.Equal(t, int64(200), body.NextScanAt)
}

// Only containers that resolve are asked about, and the defaults reach Cloud.
func Test_cloudFindings_resolves_requested_containers(t *testing.T) {
	h := restrictedHandler(t)
	got := stubFindings(h, &cloud.FindingsResult{Findings: []cloud.Finding{
		{ID: "k3Xa9", Severity: "high", Status: "ongoing", Headline: "worker OOM", ContainerNames: []string{"dev-allowed"}, URL: "https://cloud.dozzle.dev/findings/k3Xa9"},
	}, FixLocked: true, LastScanAt: 1_700_000_000_000_000_000, NextScanAt: 1_700_003_600_000_000_000}, nil)

	q := url.Values{"container": {"localhost/dev123", "localhost/missing", "nohost"}}
	code, body := findingsRequest(t, h, asUnfilteredUser(httptest.NewRequest(http.MethodGet, findingsURL(q.Encode()), nil)))

	require.Equal(t, http.StatusOK, code)
	assert.Equal(t, []string{"dev-allowed"}, got.names)
	assert.Equal(t, "open", got.status, "the default view is open and ongoing")
	assert.False(t, got.ignored)
	assert.Equal(t, int32(20), got.limit)
	assert.True(t, body.FixLocked)
	require.Len(t, body.Findings, 1)
	assert.Equal(t, []cloud.FindingContainer{{Host: "localhost", ID: "dev123", Name: "dev-allowed"}}, body.Findings[0].Containers)
	assert.Equal(t, "https://cloud.dozzle.dev/findings/k3Xa9", body.Findings[0].URL)
	assert.Equal(t, int64(1_700_000_000_000_000_000), body.LastScanAt)
	assert.Equal(t, int64(1_700_003_600_000_000_000), body.NextScanAt)
}

// Nothing resolved still asks Cloud, with no names, so the home page gets the
// review schedule.
func Test_cloudFindings_no_containers_still_asks_cloud(t *testing.T) {
	h := restrictedHandler(t)
	got := stubFindings(h, &cloud.FindingsResult{NextScanAt: 300}, nil)

	code, body := findingsRequest(t, h, asUnfilteredUser(httptest.NewRequest(http.MethodGet, findingsURL(url.Values{"container": {"localhost/missing"}}.Encode()), nil)))
	require.Equal(t, http.StatusOK, code)
	assert.Empty(t, body.Findings)
	assert.Equal(t, 1, got.calls)
	assert.Empty(t, got.names)
	assert.Equal(t, int64(300), body.NextScanAt)

	code, body = findingsRequest(t, h, asUnfilteredUser(httptest.NewRequest(http.MethodGet, findingsURL(""), nil)))
	require.Equal(t, http.StatusOK, code)
	assert.Equal(t, 2, got.calls)
	assert.Empty(t, got.names)
	assert.Equal(t, int64(300), body.NextScanAt)
}

// sameNameHosts resolves one container name on two hosts, the case Cloud cannot
// tell apart because a finding records names only.
type sameNameHosts struct {
	HostService
	containers map[string]container.Container
}

func (s *sameNameHosts) FindContainer(host, id string, labels container.ContainerLabels) (*container.ContainerService, error) {
	c, ok := s.containers[host+"/"+id]
	if !ok {
		return nil, errors.New("not found")
	}
	return container.NewContainerService(nil, c), nil
}

// A name on two hosts tags both containers, the name goes to Cloud once, and a
// finding about a name nobody asked for is dropped.
func Test_cloudFindings_maps_names_back_to_every_container(t *testing.T) {
	h := restrictedHandler(t)
	h.hostService = &sameNameHosts{HostService: h.hostService, containers: map[string]container.Container{
		"a/1": {ID: "1", Host: "a", Name: "web"},
		"b/2": {ID: "2", Host: "b", Name: "web"},
		"a/3": {ID: "3", Host: "a", Name: "db"},
	}}
	got := stubFindings(h, &cloud.FindingsResult{Findings: []cloud.Finding{
		{ID: "f1", ContainerNames: []string{"web", "web"}},
		{ID: "f2", ContainerNames: []string{"db", "web"}},
		{ID: "f3", ContainerNames: []string{"elsewhere"}},
	}, Truncated: true}, nil)

	q := url.Values{"container": {"a/1", "b/2", "a/3", "a/1"}, "filter": {"all"}, "limit": {"500"}}
	code, body := findingsRequest(t, h, asUnfilteredUser(httptest.NewRequest(http.MethodGet, findingsURL(q.Encode()), nil)))

	require.Equal(t, http.StatusOK, code)
	assert.Equal(t, []string{"web", "db"}, got.names, "an unfiltered user keeps a name on two hosts")
	assert.Equal(t, int32(50), got.limit, "limit is capped")
	assert.True(t, body.Truncated)
	assert.True(t, body.CanIgnore)
	require.Len(t, body.Findings, 2)
	web1 := cloud.FindingContainer{Host: "a", ID: "1", Name: "web"}
	web2 := cloud.FindingContainer{Host: "b", ID: "2", Name: "web"}
	db := cloud.FindingContainer{Host: "a", ID: "3", Name: "db"}
	assert.Equal(t, []cloud.FindingContainer{web1, web2}, body.Findings[0].Containers)
	assert.Equal(t, []cloud.FindingContainer{db, web1, web2}, body.Findings[1].Containers)
}

func Test_cloudFindings_filters(t *testing.T) {
	for filter, want := range map[string]struct {
		status  string
		ignored bool
	}{
		"":        {"open", false},
		"new":     {"open", false},
		"all":     {"", false},
		"ignored": {"", true},
	} {
		t.Run(filter, func(t *testing.T) {
			h := restrictedHandler(t)
			got := stubFindings(h, &cloud.FindingsResult{}, nil)
			q := url.Values{"container": {"localhost/dev123"}, "filter": {filter}}
			code, _ := findingsRequest(t, h, asUnfilteredUser(httptest.NewRequest(http.MethodGet, findingsURL(q.Encode()), nil)))
			require.Equal(t, http.StatusOK, code)
			assert.Equal(t, want.status, got.status)
			assert.Equal(t, want.ignored, got.ignored)
		})
	}

	h := restrictedHandler(t)
	got := stubFindings(h, &cloud.FindingsResult{}, nil)
	q := url.Values{"container": {"localhost/dev123"}, "filter": {"resolved"}}
	code, _ := findingsRequest(t, h, asUnfilteredUser(httptest.NewRequest(http.MethodGet, findingsURL(q.Encode()), nil)))
	assert.Equal(t, http.StatusBadRequest, code)
	assert.Zero(t, got.calls)
}

// Past the cap the first containers, in request order, are looked up and the
// answer says it is incomplete.
func Test_cloudFindings_too_many_containers(t *testing.T) {
	h := restrictedHandler(t)
	containers := map[string]container.Container{}
	q := url.Values{}
	var want []string
	for i := range maxAlertContainers + 5 {
		name := fmt.Sprintf("c%d", i)
		containers["h/"+name] = container.Container{ID: name, Host: "h", Name: name}
		q.Add("container", "h/"+name)
		if i < maxAlertContainers {
			want = append(want, name)
		}
	}
	h.hostService = &sameNameHosts{HostService: h.hostService, containers: containers}
	got := stubFindings(h, &cloud.FindingsResult{}, nil)

	code, body := findingsRequest(t, h, asUnfilteredUser(httptest.NewRequest(http.MethodGet, findingsURL(q.Encode()), nil)))
	require.Equal(t, http.StatusOK, code)
	assert.Equal(t, want, got.names)
	assert.True(t, body.Truncated)

	// Under the cap, Cloud's own flag is all that sets it.
	q = url.Values{"container": {"h/c0"}}
	code, body = findingsRequest(t, h, asUnfilteredUser(httptest.NewRequest(http.MethodGet, findingsURL(q.Encode()), nil)))
	require.Equal(t, http.StatusOK, code)
	assert.False(t, body.Truncated)
}

// canIgnore is the ignore route's own checks, answered up front: the cloud role
// and no filter.
func Test_cloudFindings_canIgnore(t *testing.T) {
	q := url.Values{"container": {"localhost/dev123"}}
	for name, tc := range map[string]struct {
		provider AuthProvider
		user     *auth.User
		want     bool
	}{
		"no auth":              {provider: NONE, want: true},
		"cloud role":           {provider: SIMPLE, user: &auth.User{Roles: auth.Cloud}, want: true},
		"no cloud role":        {provider: SIMPLE, user: &auth.User{Roles: auth.Actions}},
		"filtered":             {provider: SIMPLE, user: &auth.User{ContainerLabels: devLabels}},
		"filtered, every role": {provider: SIMPLE, user: &auth.User{Roles: auth.All, ContainerLabels: devLabels}},
	} {
		t.Run(name, func(t *testing.T) {
			h := restrictedHandler(t)
			h.config.Authorization.Provider = tc.provider
			stubFindings(h, &cloud.FindingsResult{}, nil)
			r := httptest.NewRequest(http.MethodGet, findingsURL(q.Encode()), nil)
			if tc.user != nil {
				r = r.WithContext(auth.WithUser(context.Background(), *tc.user))
			}
			rr := httptest.NewRecorder()
			h.cloudFindings(rr, r)
			require.Equal(t, http.StatusOK, rr.Code)
			var body map[string]any
			require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
			assert.Equal(t, tc.want, body["canIgnore"])
		})
	}
}

func Test_cloudFindings_errors(t *testing.T) {
	q := url.Values{"container": {"localhost/dev123"}}
	req := func() *http.Request {
		return asUnfilteredUser(httptest.NewRequest(http.MethodGet, findingsURL(q.Encode()), nil))
	}

	h := restrictedHandler(t)
	code, _ := findingsRequest(t, h, req())
	assert.Equal(t, http.StatusServiceUnavailable, code, "no hook wired")

	for name, tc := range map[string]struct {
		err  error
		want int
	}{
		"older cloud":    {fmt.Errorf("cloud: findings: %w", status.Error(codes.Unimplemented, "unknown method")), http.StatusOK},
		"not configured": {cloud.ErrNotConfigured, http.StatusServiceUnavailable},
		"deadline":       {context.DeadlineExceeded, http.StatusGatewayTimeout},
		"grpc deadline":  {status.Error(codes.DeadlineExceeded, "slow"), http.StatusGatewayTimeout},
		"other":          {errors.New("boom"), http.StatusBadGateway},
	} {
		t.Run(name, func(t *testing.T) {
			h := restrictedHandler(t)
			stubFindings(h, nil, tc.err)
			code, body := findingsRequest(t, h, req())
			assert.Equal(t, tc.want, code)
			if code == http.StatusOK {
				assert.Empty(t, body.Findings, "an older Cloud is an empty list")
			}
		})
	}
}

func ignoreRouter(h *handler) http.Handler {
	r := chi.NewRouter()
	r.Post("/api/cloud/findings/{id}/ignore", h.ignoreCloudFinding)
	r.Delete("/api/cloud/findings/{id}/ignore", h.ignoreCloudFinding)
	return r
}

func Test_ignoreCloudFinding(t *testing.T) {
	h := restrictedHandler(t)
	var gotID string
	var gotUndo bool
	calls := 0
	h.config.Cloud.IgnoreFinding = func(_ context.Context, id string, undo bool) (*cloud.Finding, error) {
		gotID, gotUndo = id, undo
		calls++
		return &cloud.Finding{ID: id, Severity: "low", Status: "open", IgnoredAt: 42}, nil
	}

	serve := func(r *http.Request) *httptest.ResponseRecorder {
		rr := httptest.NewRecorder()
		ignoreRouter(h).ServeHTTP(rr, r)
		return rr
	}

	rr := serve(httptest.NewRequest(http.MethodPost, "/api/cloud/findings/k3Xa9/ignore", nil))
	require.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "k3Xa9", gotID)
	assert.False(t, gotUndo)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
	assert.Equal(t, "k3Xa9", body["id"])
	assert.Equal(t, []any{}, body["containers"])
	assert.EqualValues(t, 42, body["ignoredAt"])

	rr = serve(httptest.NewRequest(http.MethodDelete, "/api/cloud/findings/k3Xa9/ignore", nil))
	require.Equal(t, http.StatusOK, rr.Code)
	assert.True(t, gotUndo, "DELETE takes the ignore back")

	// A filtered user could silence a finding about containers they cannot see.
	calls = 0
	rr = serve(asRestrictedUser(httptest.NewRequest(http.MethodPost, "/api/cloud/findings/k3Xa9/ignore", nil)))
	assert.Equal(t, http.StatusForbidden, rr.Code)
	// Every role does not make a filtered user an admin.
	rr = serve(httptest.NewRequest(http.MethodPost, "/api/cloud/findings/k3Xa9/ignore", nil).
		WithContext(auth.WithUser(context.Background(), auth.User{Roles: auth.All, ContainerLabels: devLabels})))
	assert.Equal(t, http.StatusForbidden, rr.Code)
	assert.Zero(t, calls)
}

func Test_ignoreCloudFinding_errors(t *testing.T) {
	serve := func(h *handler) int {
		rr := httptest.NewRecorder()
		ignoreRouter(h).ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/cloud/findings/k3Xa9/ignore", nil))
		return rr.Code
	}

	h := restrictedHandler(t)
	assert.Equal(t, http.StatusServiceUnavailable, serve(h), "no hook wired")

	for name, tc := range map[string]struct {
		err  error
		want int
	}{
		"not found":      {fmt.Errorf("cloud: ignore finding: %w", status.Error(codes.NotFound, "no")), http.StatusNotFound},
		"bad id":         {status.Error(codes.InvalidArgument, "bad id"), http.StatusBadRequest},
		"older cloud":    {status.Error(codes.Unimplemented, "unknown method"), http.StatusNotImplemented},
		"not configured": {cloud.ErrNotConfigured, http.StatusServiceUnavailable},
		"deadline":       {context.DeadlineExceeded, http.StatusGatewayTimeout},
		"other":          {errors.New("boom"), http.StatusBadGateway},
	} {
		t.Run(name, func(t *testing.T) {
			h := restrictedHandler(t)
			h.config.Cloud.IgnoreFinding = func(context.Context, string, bool) (*cloud.Finding, error) {
				return nil, tc.err
			}
			assert.Equal(t, tc.want, serve(h))
		})
	}
}

package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/config"
	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/imagecheck"
	"github.com/amir20/dozzle/internal/updatepolicy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeUpdatePolicies(t *testing.T, mode string, choices map[string]string) {
	t.Helper()
	require.NoError(t, config.Update(setupConfigPath, func(f *config.File) {
		if mode != "" {
			f.UpdateContainers = &mode
		}
		f.ContainerUpdates = choices
	}))
}

func names(containers []container.Container) []string {
	var out []string
	for _, c := range containers {
		out = append(out, c.Name)
	}
	return out
}

func policyFleet() []container.Container {
	return []container.Container{
		{ID: "a00000000001", Name: "web", EngineName: "web", Host: "nas", State: "running"},
		{ID: "a00000000002", Name: "Postgres", EngineName: "db-1", Host: "nas", State: "running"},
		{ID: "a00000000003", Name: "labelled", EngineName: "labelled", Host: "nas", State: "running", Labels: map[string]string{updatepolicy.Label: "auto"}},
		{ID: "a00000000004", Name: "legacy", EngineName: "legacy", Host: "nas", State: "running", Labels: map[string]string{AutoUpdateLabel: "true"}},
		{ID: "a00000000005", Name: "frozen", EngineName: "frozen", Host: "nas", State: "running", Labels: map[string]string{updatepolicy.Label: "off"}},
		{ID: "a00000000006", Name: "sick", EngineName: "sick", Host: "nas", State: "running", Health: "unhealthy", Labels: map[string]string{updatepolicy.Label: "auto"}},
		// Dozzle itself, matched by setupSelfID.
		{ID: testSelfID[:12], Name: "dozzle", EngineName: "dozzle", Host: "nas", State: "running", Labels: map[string]string{updatepolicy.Label: "auto"}},
	}
}

// An install that upgrades with nothing new in dozzle.yml updates exactly what
// it did before: the containers labelled for it, and nothing else.
func TestScheduledContainers_UpgradeChangesNothing(t *testing.T) {
	setupTestEnv(t, true)
	got := scheduledContainers(policyFleet(), loadUpdatePolicies(), "")
	assert.Equal(t, []string{"labelled", "legacy"}, names(got), "unhealthy, off, unlabelled and Dozzle itself are left out")
}

func TestScheduledContainers_ChoicesAndModes(t *testing.T) {
	setupTestEnv(t, true)

	// A choice saved from the UI, keyed by the engine's name.
	writeUpdatePolicies(t, "", map[string]string{"nas/db-1": "auto", "nas/labelled": "off"})
	got := scheduledContainers(policyFleet(), loadUpdatePolicies(), "")
	assert.Equal(t, []string{"Postgres", "labelled", "legacy"}, names(got), "the label wins over the UI's off")

	// Everything: anything nobody chose for, except what is manual or off.
	writeUpdatePolicies(t, "all", map[string]string{"nas/web": "manual"})
	got = scheduledContainers(policyFleet(), loadUpdatePolicies(), "")
	assert.Equal(t, []string{"Postgres", "labelled", "legacy"}, names(got))

	// Dozzle only holds the UI's picks back, but a label still wins.
	writeUpdatePolicies(t, "dozzle", map[string]string{"nas/db-1": "auto"})
	got = scheduledContainers(policyFleet(), loadUpdatePolicies(), "")
	assert.Equal(t, []string{"labelled", "legacy"}, names(got))
}

// Everything leaves a stopped container alone unless someone chose auto for
// it, by label or from the UI.
func TestScheduledContainers_EverythingSkipsStopped(t *testing.T) {
	setupTestEnv(t, true)
	fleet := append(policyFleet(),
		container.Container{ID: "a00000000007", Name: "migrate", EngineName: "migrate", Host: "nas", State: "exited"},
		container.Container{ID: "a00000000008", Name: "fresh", EngineName: "fresh", Host: "nas", State: "created"},
		container.Container{ID: "a00000000009", Name: "parked", EngineName: "parked", Host: "nas", State: "exited", Labels: map[string]string{updatepolicy.Label: "auto"}},
		container.Container{ID: "a00000000010", Name: "picked", EngineName: "picked", Host: "nas", State: "exited"},
	)
	writeUpdatePolicies(t, "all", map[string]string{"nas/picked": "auto"})
	got := scheduledContainers(fleet, loadUpdatePolicies(), "")
	assert.Equal(t, []string{"web", "Postgres", "labelled", "legacy", "parked", "picked"}, names(got))
}

// The scheduler never checks a container that is not on the schedule, and only
// updates the ones the registry says are outdated.
func TestAutoUpdate_OutdatedFollowsPolicy(t *testing.T) {
	setupTestEnv(t, true)
	writeUpdatePolicies(t, "all", map[string]string{"nas/web": "off"})
	checked := map[string]imagecheck.Result{
		"a00000000002": {Status: imagecheck.StatusUpdateAvailable, RemoteDigest: "sha256:db"},
		"a00000000003": {Status: imagecheck.StatusPinned},
		"a00000000004": {Status: imagecheck.StatusAuthRequired},
	}
	client := &countingCheckService{results: checked}
	s := &autoUpdateScheduler{config: &serverActions, hostService: &labelledHosts{containers: policyFleet(), client: client}}

	outdated, _ := s.outdatedAutoContainers(context.Background())
	require.Len(t, outdated, 1)
	assert.Equal(t, "Postgres", outdated[0].Container.Name)
	assert.ElementsMatch(t, []string{"a00000000002", "a00000000003", "a00000000004"}, client.ids(), "off, unhealthy and Dozzle are never checked")
}

type countingCheckService struct {
	checkingClientService
	mu      sync.Mutex
	checked []string
}

func (s *countingCheckService) CheckImageUpdate(ctx context.Context, c container.Container, force bool) (imagecheck.Result, error) {
	s.mu.Lock()
	s.checked = append(s.checked, c.ID)
	s.mu.Unlock()
	return s.checkingClientService.CheckImageUpdate(ctx, c, force)
}

func (s *countingCheckService) ids() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.checked)
}

func TestAutoUpdate_SelfLabelOffSkipsDozzle(t *testing.T) {
	setupTestEnv(t, true)
	rec := stubAutoUpdate(t, imagecheck.StatusUpdateAvailable)
	writeSchedule(t, "daily", "03:00")
	selfUpdateInspect = func(context.Context, HostService, string) (selfImage, error) {
		return selfImage{Ref: "amir20/dozzle:latest", Labels: map[string]string{updatepolicy.Label: "off"}}, nil
	}
	newTestScheduler(serverActions).tick(context.Background(), at(14, "03:00"))
	_, starts := rec.counts()
	assert.Equal(t, 0, starts)
}

func TestRecordAutoUpdateResults(t *testing.T) {
	now := time.Now()
	c := container.Container{ID: "x", Name: "Pretty", EngineName: "app", Host: "nas"}
	job := &bulkUpdateJob{Trigger: "schedule", FinishedAt: &now, Items: []*bulkUpdateItem{
		{Status: bulkRolledBack, Error: "unhealthy", service: container.NewContainerService(nil, c)},
	}}
	recordAutoUpdateResults(job)
	last := lastAutoUpdateFor(container.Container{ID: "y", EngineName: "app", Host: "nas"})
	require.NotNil(t, last, "found again under the replacement's id")
	assert.Equal(t, bulkRolledBack, last.Status)
	assert.Equal(t, "unhealthy", last.Error)

	// A manual run is not an auto-update.
	job.Trigger = "manual"
	job.Items[0].Status = bulkDone
	recordAutoUpdateResults(job)
	assert.Equal(t, bulkRolledBack, lastAutoUpdateFor(c).Status)
}

func TestNamedVolumes(t *testing.T) {
	c := container.Container{Mounts: []container.Mount{
		{Type: "volume", Source: "/var/lib/docker/volumes/pgdata/_data", Destination: "/var/lib/postgresql/data"},
		{Type: "volume", Source: "/var/lib/docker/volumes/" + strings.Repeat("ab", 32) + "/_data", Destination: "/tmp"},
		{Type: "bind", Source: "/srv/config", Destination: "/config"},
	}}
	assert.Equal(t, []string{"pgdata"}, namedVolumes(c))
}

func policyHandler(cfg Config) *handler {
	cfg.Base = "/"
	cfg.Mode = "server"
	return &handler{config: &cfg, hostService: &labelledHosts{containers: policyFleet()}}
}

func doPolicy(h *handler, method, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "/api/updates/policy", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	if method == http.MethodGet {
		h.getUpdatePolicies(rr, req)
	} else {
		h.setUpdatePolicy(rr, req)
	}
	return rr
}

func TestUpdatePolicyAPI_List(t *testing.T) {
	setupTestEnv(t, true)
	writeUpdatePolicies(t, "dozzle", map[string]string{"nas/db-1": "auto", "nas/web": "off"})
	h := policyHandler(Config{EnableActions: true, Authorization: Authorization{Provider: NONE}})

	rr := doPolicy(h, http.MethodGet, "")
	require.Equal(t, http.StatusOK, rr.Code)
	var resp updatePoliciesResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, updatepolicy.ModeDozzle, resp.Mode)
	assert.True(t, resp.Persisted)
	assert.True(t, resp.CanChoose)

	byName := map[string]containerUpdatePolicy{}
	for _, c := range resp.Containers {
		byName[c.Name] = c
	}
	assert.Equal(t, containerUpdatePolicy{Host: "nas", ID: "a00000000002", Name: "Postgres", State: "running", Policy: "manual", Source: "default", Choice: "auto"}, byName["Postgres"], "Dozzle only holds the pick back")
	assert.Equal(t, updatepolicy.Off, byName["web"].Policy)
	assert.Equal(t, updatepolicy.SourceChoice, byName["web"].Source)
	assert.Equal(t, updatepolicy.SourceLabel, byName["frozen"].Source)
	assert.True(t, byName["dozzle"].Self)
	var order []string
	for _, c := range resp.Containers {
		order = append(order, c.Name)
	}
	assert.Equal(t, []string{"dozzle", "frozen", "labelled", "legacy", "Postgres", "sick", "web"}, order, "sorted by name, ignoring case")
}

func TestUpdatePolicyAPI_Set(t *testing.T) {
	setupTestEnv(t, true)
	writeUpdatePolicies(t, "dozzle", nil)
	h := policyHandler(Config{EnableActions: true, Authorization: Authorization{Provider: NONE}})

	rr := doPolicy(h, http.MethodPost, `{"containers":[{"host":"nas","id":"a00000000002"}],"policy":"auto"}`)
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	var resp setUpdatePolicyResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, setUpdatePolicyResponse{Mode: updatepolicy.ModePicked, Saved: 1}, resp, "picking a container leaves Dozzle only")

	file, err := config.Load(setupConfigPath)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"nas/db-1": "auto"}, file.ContainerUpdates, "keyed by the engine's name")
	assert.Equal(t, "picked", *file.UpdateContainers)

	// "Keep these manual": several at once, and a labelled one is left alone.
	rr = doPolicy(h, http.MethodPost, `{"containers":[{"host":"nas","id":"a00000000001"},{"host":"nas","id":"a00000000005"}],"policy":"manual"}`)
	require.Equal(t, http.StatusOK, rr.Code)
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, 1, resp.Saved)
	assert.Equal(t, 1, resp.Labelled)

	// Only a label decides it: refused, so the UI can say why.
	rr = doPolicy(h, http.MethodPost, `{"containers":[{"host":"nas","id":"a00000000005"}],"policy":"auto"}`)
	assert.Equal(t, http.StatusConflict, rr.Code)
	assert.Equal(t, "set-by-label", rr.Header().Get(agentErrorHeader))

	// Empty forgets the choice; the last one leaves no map behind.
	for _, id := range []string{"a00000000001", "a00000000002"} {
		require.Equal(t, http.StatusOK, doPolicy(h, http.MethodPost, `{"containers":[{"host":"nas","id":"`+id+`"}],"policy":""}`).Code)
	}
	file, err = config.Load(setupConfigPath)
	require.NoError(t, err)
	assert.Nil(t, file.ContainerUpdates)

	assert.Equal(t, http.StatusBadRequest, doPolicy(h, http.MethodPost, `{"containers":[{"host":"nas","id":"a00000000002"}],"policy":"true"}`).Code)
	assert.Equal(t, http.StatusNotFound, doPolicy(h, http.MethodPost, `{"containers":[{"host":"nas","id":"gone"}],"policy":"off"}`).Code)
}

func TestUpdatePolicyAPI_RefusedWithoutVolume(t *testing.T) {
	setupTestEnv(t, false)
	h := policyHandler(Config{EnableActions: true, Authorization: Authorization{Provider: NONE}})
	rr := doPolicy(h, http.MethodPost, `{"containers":[{"host":"nas","id":"a00000000002"}],"policy":"auto"}`)
	assert.Equal(t, http.StatusPreconditionFailed, rr.Code)
	assert.Equal(t, "not-persisted", rr.Header().Get(agentErrorHeader))
}

// Without actions, choosing is for whoever may change settings: with no login,
// only the setup window.
func TestUpdatePolicyAPI_Permissions(t *testing.T) {
	setupTestEnv(t, true)
	closed := policyHandler(Config{Authorization: Authorization{Provider: NONE}, Setup: SetupConfig{StartedAt: time.Now().Add(-time.Hour)}})
	assert.Equal(t, http.StatusForbidden, doPolicy(closed, http.MethodPost, `{"containers":[{"host":"nas","id":"a00000000002"}],"policy":"auto"}`).Code)

	open := policyHandler(Config{Authorization: Authorization{Provider: NONE}, Setup: SetupConfig{StartedAt: time.Now()}})
	assert.Equal(t, http.StatusOK, doPolicy(open, http.MethodPost, `{"containers":[{"host":"nas","id":"a00000000002"}],"policy":"auto"}`).Code)
}

func TestSetup_UpdateContainersMode(t *testing.T) {
	setupTestEnv(t, true)
	h := setupNoneHandler(time.Now(), SetupConfig{})
	assert.Equal(t, updatepolicy.ModePicked, getSetupState(t, h).AutoUpdate.Containers, "unset reads as today's behaviour")

	require.Equal(t, http.StatusNoContent, doSetup(h, "PATCH", "/api/setup/config", `{"updateContainers":"all"}`).Code)
	assert.Equal(t, updatepolicy.ModeAll, getSetupState(t, h).AutoUpdate.Containers)
	assert.Equal(t, http.StatusBadRequest, doSetup(h, "PATCH", "/api/setup/config", `{"updateContainers":"everything"}`).Code)
	// Not tied to the schedule's env lock: it has no env var.
	locked := setupNoneHandler(time.Now(), SetupConfig{LockedAutoUpdate: true})
	assert.Equal(t, http.StatusNoContent, doSetup(locked, "PATCH", "/api/setup/config", `{"updateContainers":"dozzle"}`).Code)
}

// Off from the UI stops the check too, without reaching any registry.
func TestCheckAllImageUpdates_SkipsOff(t *testing.T) {
	setupTestEnv(t, true)
	writeUpdatePolicies(t, "", map[string]string{"nas/web": "off"})
	client := &countingCheckService{results: map[string]imagecheck.Result{}}
	h := &handler{config: &Config{Mode: "server", ImageCheckMode: imagecheck.ModeAutomatic, Authorization: Authorization{Provider: NONE}}, hostService: &labelledHosts{containers: policyFleet()[:2], client: client}}

	rr := httptest.NewRecorder()
	h.checkAllImageUpdates(rr, httptest.NewRequest(http.MethodGet, "/api/image/check", nil))
	var results []containerImageCheck
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &results))
	require.Len(t, results, 2)
	assert.Equal(t, []string{"a00000000002"}, client.ids())
	for _, r := range results {
		if r.ID == "a00000000001" {
			assert.Equal(t, imagecheck.StatusSkipped, r.Result.Status)
		}
	}
}

// The cloud client's decider reads dozzle.yml when it is made, and keeps that
// answer for the whole batch.
func TestUpdatePolicies_ReadsOncePerBatch(t *testing.T) {
	setupTestEnv(t, true)
	writeUpdatePolicies(t, "all", map[string]string{"nas/web": "off"})
	policy := UpdatePolicies()
	fleet := policyFleet()
	assert.Equal(t, updatepolicy.Off, policy(fleet[0]))
	assert.Equal(t, updatepolicy.Auto, policy(fleet[1]))

	writeUpdatePolicies(t, "dozzle", nil)
	assert.Equal(t, updatepolicy.Auto, policy(fleet[1]), "a batch keeps the settings it started with")
	assert.Equal(t, updatepolicy.Manual, UpdatePolicies()(fleet[1]))
}

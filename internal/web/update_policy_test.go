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

// checkingClientService answers image checks with a fixed result per container.
type checkingClientService struct {
	container.ClientService
	results map[string]imagecheck.Result
}

func (s *checkingClientService) CheckImageUpdate(_ context.Context, c container.Container, _ bool) (imagecheck.Result, error) {
	return s.results[c.ID], nil
}

// countingCheckService is a checkingClientService that remembers what it was
// asked about.
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

// fleetHosts lists containers and finds them on one client service.
type fleetHosts struct {
	HostService
	containers []container.Container
	client     container.ClientService
}

func (h *fleetHosts) ListAllContainers(container.ContainerLabels) ([]container.Container, []error) {
	return h.containers, nil
}

func (h *fleetHosts) FindContainer(_ string, id string, _ container.ContainerLabels) (*container.ContainerService, error) {
	for _, c := range h.containers {
		if c.ID == id {
			return container.NewContainerService(h.client, c), nil
		}
	}
	return nil, container.ErrContainerNotFound
}

func writeUpdateMode(t *testing.T, mode string) {
	t.Helper()
	require.NoError(t, config.Update(setupConfigPath, func(f *config.File) {
		f.UpdateContainers = &mode
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
		{ID: "a00000000001", Name: "web", Host: "nas", State: "running"},
		{ID: "a00000000002", Name: "Postgres", Host: "nas", State: "running", Mounts: []container.Mount{
			{Type: "volume", Source: "/var/lib/docker/volumes/pgdata/_data", Destination: "/var/lib/postgresql/data"},
		}},
		{ID: "a00000000003", Name: "labelled", Host: "nas", State: "running", Labels: map[string]string{updatepolicy.Label: "auto"}},
		{ID: "a00000000004", Name: "legacy", Host: "nas", State: "running", Labels: map[string]string{updatepolicy.LegacyAutoLabel: "true"}},
		{ID: "a00000000005", Name: "frozen", Host: "nas", State: "running", Labels: map[string]string{updatepolicy.Label: "off"}},
		{ID: "a00000000006", Name: "sick", Host: "nas", State: "running", Health: "unhealthy", Labels: map[string]string{updatepolicy.Label: "auto"}},
		// Dozzle itself, matched by setupSelfID.
		{ID: testSelfID[:12], Name: "dozzle", Host: "nas", State: "running", Labels: map[string]string{updatepolicy.Label: "auto"}},
	}
}

// An install that upgrades with nothing new in dozzle.yml updates exactly what
// it did before: the containers labelled for it, and nothing else.
func TestScheduledContainers_UpgradeChangesNothing(t *testing.T) {
	setupTestEnv(t, true)
	got := scheduledContainers(policyFleet(), loadUpdateMode(SetupConfig{}), "")
	assert.Equal(t, []string{"labelled", "legacy"}, names(got), "unhealthy, off, unlabelled and Dozzle itself are left out")
}

func TestScheduledContainers_Modes(t *testing.T) {
	setupTestEnv(t, true)

	writeUpdateMode(t, "all")
	got := scheduledContainers(policyFleet(), loadUpdateMode(SetupConfig{}), "")
	assert.Equal(t, []string{"web", "Postgres", "labelled", "legacy"}, names(got), "everything but off, unhealthy and Dozzle")

	writeUpdateMode(t, "off")
	got = scheduledContainers(policyFleet(), loadUpdateMode(SetupConfig{}), "")
	assert.Empty(t, got, "off moves no container, labelled or not")

	writeUpdateMode(t, "labelled")
	got = scheduledContainers(policyFleet(), loadUpdateMode(SetupConfig{}), "")
	assert.Equal(t, []string{"labelled", "legacy"}, names(got))

	env := "all"
	got = scheduledContainers(policyFleet(), loadUpdateMode(SetupConfig{UpdateContainers: &env}), "")
	assert.Equal(t, []string{"web", "Postgres", "labelled", "legacy"}, names(got), "the env var wins over the file")
}

// All leaves a stopped container alone unless its label asks for updates.
// A stopped container is never updated, whatever the mode or its label says.
func TestScheduledContainers_SkipsStopped(t *testing.T) {
	fleet := append(policyFleet(),
		container.Container{ID: "a00000000007", Name: "migrate", Host: "nas", State: "exited"},
		container.Container{ID: "a00000000008", Name: "fresh", Host: "nas", State: "created"},
		container.Container{ID: "a00000000009", Name: "parked", Host: "nas", State: "exited", Labels: map[string]string{updatepolicy.Label: "auto"}},
	)
	setupTestEnv(t, true)
	got := scheduledContainers(fleet, updatepolicy.ModeAll, "")
	assert.Equal(t, []string{"web", "Postgres", "labelled", "legacy"}, names(got))
	got = scheduledContainers(fleet, updatepolicy.ModeLabelled, "")
	assert.NotContains(t, names(got), "parked")
}

// The scheduler never checks a container that is not on the schedule, and only
// updates the ones the registry says are outdated.
func TestAutoUpdate_OutdatedFollowsPolicy(t *testing.T) {
	setupTestEnv(t, true)
	writeUpdateMode(t, "all")
	checked := map[string]imagecheck.Result{
		"a00000000001": {Status: imagecheck.StatusUpToDate},
		"a00000000002": {Status: imagecheck.StatusUpdateAvailable, RemoteDigest: "sha256:db"},
		"a00000000003": {Status: imagecheck.StatusPinned},
		"a00000000004": {Status: imagecheck.StatusAuthRequired},
	}
	client := &countingCheckService{results: checked}
	s := &autoUpdateScheduler{config: &serverActions, hostService: &fleetHosts{containers: policyFleet(), client: client}}

	outdated, _ := s.outdatedScheduledContainers(context.Background())
	require.Len(t, outdated, 1)
	assert.Equal(t, "Postgres", outdated[0].Container.Name)
	assert.ElementsMatch(t, []string{"a00000000001", "a00000000002", "a00000000003", "a00000000004"}, client.ids(), "off, unhealthy and Dozzle are never checked")
}

// A container someone rolled back keeps being offered the image its tag still
// names. The schedule leaves it until a newer image is pushed.
func TestAutoUpdate_SkipsImageRolledBackFrom(t *testing.T) {
	setupTestEnv(t, true)
	immich := container.Container{ID: "aaa", Name: "immich", Host: "nas", Image: "immich:release", State: "running", Labels: map[string]string{
		updatepolicy.Label:            "auto",
		container.RolledBackFromLabel: "immich@sha256:broken",
	}}
	redis := container.Container{ID: "bbb", Name: "redis", Host: "nas", Image: "redis:7", State: "running", Labels: map[string]string{updatepolicy.Label: "auto"}}
	client := &checkingClientService{results: map[string]imagecheck.Result{
		"aaa": {Status: imagecheck.StatusUpdateAvailable, RemoteDigest: "sha256:broken"},
		"bbb": {Status: imagecheck.StatusUpdateAvailable, RemoteDigest: "sha256:redis"},
	}}
	s := &autoUpdateScheduler{config: &serverActions, hostService: &fleetHosts{containers: []container.Container{immich, redis}, client: client}}
	outdatedNames := func() []string {
		outdated, _ := s.outdatedScheduledContainers(context.Background())
		var out []string
		for _, cs := range outdated {
			out = append(out, cs.Container.Name)
		}
		return out
	}
	assert.Equal(t, []string{"redis"}, outdatedNames(), "rolled back from sha256:broken, which immich's tag still names")

	client.results["aaa"] = imagecheck.Result{Status: imagecheck.StatusUpdateAvailable, RemoteDigest: "sha256:fixed"}
	assert.ElementsMatch(t, []string{"immich", "redis"}, outdatedNames(), "a newer image is applied")
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

func TestNamedVolumes(t *testing.T) {
	c := container.Container{Mounts: []container.Mount{
		{Type: "volume", Source: "/var/lib/docker/volumes/pgdata/_data", Destination: "/var/lib/postgresql/data"},
		{Type: "volume", Source: "/var/lib/docker/volumes/" + strings.Repeat("ab", 32) + "/_data", Destination: "/tmp"},
		{Type: "bind", Source: "/srv/config", Destination: "/config"},
	}}
	assert.Equal(t, []string{"pgdata"}, namedVolumes(c))
}

func TestUpdatePolicyAPI_List(t *testing.T) {
	setupTestEnv(t, true)
	writeUpdateMode(t, "all")
	h := &handler{config: &Config{Base: "/", Mode: "server", Authorization: Authorization{Provider: NONE}}, hostService: &fleetHosts{containers: policyFleet()}}

	rr := httptest.NewRecorder()
	h.getUpdatePolicies(rr, httptest.NewRequest(http.MethodGet, "/api/updates/policy", nil))
	require.Equal(t, http.StatusOK, rr.Code)
	var resp updatePoliciesResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, updatepolicy.ModeAll, resp.Mode)

	byName := map[string]containerUpdatePolicy{}
	var order []string
	for _, c := range resp.Containers {
		byName[c.Name] = c
		order = append(order, c.Name)
	}
	assert.Equal(t, []string{"frozen", "labelled", "legacy", "Postgres", "sick", "web"}, order, "sorted by name ignoring case, and Dozzle left out")
	assert.Equal(t, containerUpdatePolicy{Host: "nas", ID: "a00000000002", Name: "Postgres", State: "running", Volumes: []string{"pgdata"}}, byName["Postgres"])
	assert.Equal(t, updatepolicy.Off, byName["frozen"].Label)
	assert.Equal(t, updatepolicy.Auto, byName["legacy"].Label, "the old label still reads as auto")
}

func TestSetup_UpdateContainersMode(t *testing.T) {
	setupTestEnv(t, true)
	h := setupNoneHandler(time.Now(), SetupConfig{})
	assert.Equal(t, updatepolicy.ModeLabelled, getSetupState(t, h).AutoUpdate.Containers, "unset reads as today's behaviour")

	require.Equal(t, http.StatusNoContent, doSetup(h, "PATCH", "/api/setup/config", `{"updateContainers":"all"}`).Code)
	assert.Equal(t, updatepolicy.ModeAll, getSetupState(t, h).AutoUpdate.Containers)
	assert.Equal(t, http.StatusBadRequest, doSetup(h, "PATCH", "/api/setup/config", `{"updateContainers":"everything"}`).Code)
	// Not tied to the schedule's env lock: it has no env var.
	locked := setupNoneHandler(time.Now(), SetupConfig{LockedAutoUpdate: true})
	assert.Equal(t, http.StatusNoContent, doSetup(locked, "PATCH", "/api/setup/config", `{"updateContainers":"off"}`).Code)
}

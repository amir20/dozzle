package cloud

import (
	"context"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/container"
	container_support "github.com/amir20/dozzle/internal/support/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// The read path collapses the find-then-act round-trip: when a name matches
// several same-service replicas/corpses, a read-only tool resolves to the
// single most-relevant container (running first, then newest active) in one
// shot instead of erroring and forcing the LLM to call find_containers and
// re-issue. The write path stays strict — these tests lock both halves in.

// --- resolver unit tests: read mode (resolveContainerRefRead) ---

func TestResolveRead_AllStopped_PicksMostRecentlyActiveCorpse(t *testing.T) {
	// The crash-loop case: every "svc.1.*" task has exited. There is no live
	// container, but a read-only inspect/logs call still has an obvious target —
	// the corpse that died most recently, which is what the investigation is
	// about. Resolve to it rather than erroring.
	t0 := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	deps := resolverDeps([]container.Container{
		{ID: "old", Name: "svc.1.aaa", Host: "local", State: "exited", StartedAt: t0, FinishedAt: t0.Add(1 * time.Minute)},
		{ID: "newest", Name: "svc.1.bbb", Host: "local", State: "exited", StartedAt: t0.Add(5 * time.Minute), FinishedAt: t0.Add(6 * time.Minute)},
		{ID: "mid", Name: "svc.1.ccc", Host: "local", State: "exited", StartedAt: t0.Add(2 * time.Minute), FinishedAt: t0.Add(3 * time.Minute)},
	})

	host, id, note, err := resolveContainerRefRead("svc.1", "", deps)
	assert.NoError(t, err)
	assert.Equal(t, "local", host)
	assert.Equal(t, "newest", id)
	// The pick is transparent: the note names the chosen task and that siblings
	// exist, so the model can re-call with an explicit id if it wants another.
	assert.NotEmpty(t, note)
	assert.Contains(t, note, "svc.1.bbb")
}

func TestResolveRead_MultipleRunning_PicksNewestRunningWithNote(t *testing.T) {
	// Several live replicas of one service. For a read (logs/inspect) the
	// replicas are interchangeable, so resolve to the newest running one and say
	// so — never force a round-trip.
	t0 := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	deps := resolverDeps([]container.Container{
		{ID: "r-old", Name: "svc.1.aaa", Host: "local", State: "running", StartedAt: t0},
		{ID: "r-new", Name: "svc.1.bbb", Host: "local", State: "running", StartedAt: t0.Add(10 * time.Minute)},
		{ID: "dead", Name: "svc.1.ccc", Host: "local", State: "exited", StartedAt: t0.Add(20 * time.Minute)},
	})

	host, id, note, err := resolveContainerRefRead("svc.1", "", deps)
	assert.NoError(t, err)
	assert.Equal(t, "local", host)
	// Running beats a newer-but-dead container; among running, newest StartedAt.
	assert.Equal(t, "r-new", id)
	assert.NotEmpty(t, note)
	assert.Contains(t, note, "svc.1.bbb")
	assert.Contains(t, note, "running")
}

func TestResolveRead_SingleRunning_NoNote(t *testing.T) {
	// Exactly one running among corpses — same outcome as the write path, and
	// since the choice is unambiguous there is nothing to disclose: no note.
	deps := resolverDeps([]container.Container{
		{ID: "dead", Name: "svc.1.aaa", Host: "local", State: "exited"},
		{ID: "live", Name: "svc.1.bbb", Host: "local", State: "running"},
	})

	host, id, note, err := resolveContainerRefRead("svc.1", "", deps)
	assert.NoError(t, err)
	assert.Equal(t, "local", host)
	assert.Equal(t, "live", id)
	assert.Empty(t, note)
}

func TestResolveRead_UniqueName_NoNote(t *testing.T) {
	// A plain unique match resolves exactly as before with no note — the read
	// relaxation only adds behavior to the previously-erroring ambiguous case.
	deps := resolverDeps([]container.Container{
		{ID: "abc123def456", Name: "nginx", Host: "local", State: "running"},
		{ID: "fff999", Name: "redis", Host: "local", State: "running"},
	})

	host, id, note, err := resolveContainerRefRead("nginx", "", deps)
	assert.NoError(t, err)
	assert.Equal(t, "local", host)
	assert.Equal(t, "abc123def456", id)
	assert.Empty(t, note)
}

func TestResolveRead_AmbiguousAcrossHosts_PicksNewestRunning(t *testing.T) {
	// Same name on two hosts. The write path refuses (host_id needed). The read
	// path is allowed to pick the newest running one across hosts — reads are
	// safe and the note discloses the cross-host siblings.
	t0 := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	deps := resolverDeps([]container.Container{
		{ID: "a", Name: "nginx", Host: "host-a", State: "running", StartedAt: t0},
		{ID: "b", Name: "nginx", Host: "host-b", State: "running", StartedAt: t0.Add(1 * time.Minute)},
	})

	host, id, note, err := resolveContainerRefRead("nginx", "", deps)
	assert.NoError(t, err)
	assert.Equal(t, "host-b", host)
	assert.Equal(t, "b", id)
	assert.NotEmpty(t, note)
}

func TestResolveRead_NotFound_StillErrors(t *testing.T) {
	// No match at all is still an error — the relaxation never invents a target.
	deps := resolverDeps([]container.Container{
		{ID: "id1", Name: "nginx", Host: "local"},
	})

	_, _, _, err := resolveContainerRefRead("does-not-exist", "", deps)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no container matching")
}

func TestResolveRead_HostScoped_StillRespectsHost(t *testing.T) {
	// An explicit host still scopes the search even on the read path.
	deps := resolverDeps([]container.Container{
		{ID: "id1", Name: "nginx", Host: "host-a", State: "running"},
		{ID: "id2", Name: "nginx", Host: "host-b", State: "running"},
	})

	host, id, _, err := resolveContainerRefRead("nginx", "host-b", deps)
	assert.NoError(t, err)
	assert.Equal(t, "host-b", host)
	assert.Equal(t, "id2", id)
}

// --- write path stays strict: the relaxation must NOT leak into writes ---

func TestResolveWrite_AllStopped_StillRefuses(t *testing.T) {
	// The read path now resolves this; the write path must still refuse, because
	// picking a destructive target among corpses is exactly the ambiguity write
	// tools must never guess at.
	deps := resolverDeps([]container.Container{
		{ID: "s1", Name: "svc.1.aaa", Host: "local", State: "exited"},
		{ID: "s2", Name: "svc.1.bbb", Host: "local", State: "exited"},
	})

	_, _, err := resolveContainerRef("svc.1", "", deps)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "matches multiple containers")
}

func TestResolveWrite_MultipleRunning_StillRefuses(t *testing.T) {
	deps := resolverDeps([]container.Container{
		{ID: "r1", Name: "svc.1.aaa", Host: "local", State: "running"},
		{ID: "r2", Name: "svc.1.bbb", Host: "local", State: "running"},
	})

	_, _, err := resolveContainerRef("svc.1", "", deps)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "matches multiple containers")
}

// --- end-to-end through ExecuteTool: read tools resolve in one shot ---

// closedLogsClientService is a MockClientService whose LogsBetweenDates returns
// an already-closed channel, so executeFetchContainerLogs's range loop drains
// immediately instead of blocking on the bare mock's nil channel. Embeds
// MockClientService so it satisfies the full ClientService interface unchanged.
type closedLogsClientService struct {
	MockClientService
}

func (c *closedLogsClientService) LogsBetweenDates(_ context.Context, _ container.Container, _ time.Time, _ time.Time, _ container.StdType) (<-chan *container.LogEvent, error) {
	ch := make(chan *container.LogEvent)
	close(ch)
	return ch, nil
}

func TestExecuteTool_FetchLogs_AllStopped_ResolvesNewestCorpseWithNote(t *testing.T) {
	// fetch_container_logs on a crash-looped "svc.1" (every task exited) must
	// succeed against the most-recently-active corpse and surface the pick in
	// the result, not fail and force a find_containers round-trip.
	t0 := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	mockHost := &MockHostService{}
	withResolver(mockHost,
		container.Container{ID: "old", Name: "svc.1.aaa", Host: "local", State: "exited", StartedAt: t0},
		container.Container{ID: "newest", Name: "svc.1.bbb", Host: "local", State: "exited", StartedAt: t0.Add(5 * time.Minute)},
	)
	newestC := container.Container{ID: "newest", Name: "svc.1.bbb", Host: "local", State: "exited", StartedAt: t0.Add(5 * time.Minute)}
	cs := container_support.NewContainerService(&closedLogsClientService{}, newestC)
	mockHost.On("FindContainer", "local", "newest", container.ContainerLabels(nil)).Return(cs, nil)

	resp := ExecuteTool(context.Background(), "fetch_container_logs", `{"container_id":"svc.1"}`, ToolDeps{HostService: mockHost})
	assert.True(t, resp.Success)
	// The returned container_name discloses the resolution so the model gets the
	// sibling context for free in the same turn.
	name := resp.GetFetchLogs().ContainerName
	assert.Contains(t, name, "svc.1.bbb")
	assert.Contains(t, name, "svc.1") // mentions the ambiguous ref it resolved
	mockHost.AssertCalled(t, "FindContainer", "local", "newest", container.ContainerLabels(nil))
}

func TestExecuteTool_InspectContainer_AllStopped_ResolvesNewestCorpse(t *testing.T) {
	// inspect_container on the same crash-loop resolves to the newest corpse in
	// one shot. Inspect returns the concrete id/name/state (itself
	// disambiguating), so we don't mangle its structured fields with a note.
	t0 := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	mockHost := &MockHostService{}
	withResolver(mockHost,
		container.Container{ID: "old", Name: "svc.1.aaa", Host: "local", State: "exited", StartedAt: t0},
		container.Container{ID: "newest", Name: "svc.1.bbb", Host: "local", State: "exited", StartedAt: t0.Add(5 * time.Minute)},
	)
	newestC := container.Container{ID: "newest", Name: "svc.1.bbb", Host: "local", State: "exited", StartedAt: t0.Add(5 * time.Minute)}
	cs := container_support.NewContainerService(&MockClientService{}, newestC)
	mockHost.On("FindContainer", "local", "newest", container.ContainerLabels(nil)).Return(cs, nil)

	resp := ExecuteTool(context.Background(), "inspect_container", `{"container_id":"svc.1"}`, ToolDeps{HostService: mockHost})
	assert.True(t, resp.Success)
	assert.Equal(t, "newest", resp.GetInspectContainer().Id)
	assert.Equal(t, "svc.1.bbb", resp.GetInspectContainer().Name)
}

func TestExecuteTool_RestartContainer_AllStopped_StillRefuses(t *testing.T) {
	// The destructive counterpart of the fetch_logs test above: the SAME
	// ambiguous all-stopped ref that reads now resolve must still refuse for a
	// write tool, and must never reach FindContainer / ContainerAction.
	mockClient := &MockClientService{}
	mockHost := &MockHostService{}
	withResolver(mockHost,
		container.Container{ID: "s1", Name: "svc.1.aaa", Host: "local", State: "exited"},
		container.Container{ID: "s2", Name: "svc.1.bbb", Host: "local", State: "exited"},
	)

	resp := ExecuteTool(context.Background(), "restart_container", `{"container_id":"svc.1"}`, ToolDeps{HostService: mockHost, EnableActions: true})
	assert.False(t, resp.Success)
	assert.Contains(t, resp.Error, "matches multiple containers")
	mockHost.AssertNotCalled(t, "FindContainer", mock.Anything, mock.Anything, mock.Anything)
	mockClient.AssertNotCalled(t, "ContainerAction", mock.Anything, mock.Anything, mock.Anything)
}

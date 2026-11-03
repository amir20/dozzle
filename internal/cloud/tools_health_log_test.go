package cloud

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeHealthLog struct {
	probes []container.HealthProbe
	err    error
}

func (f fakeHealthLog) HealthLog(context.Context) ([]container.HealthProbe, error) {
	return f.probes, f.err
}

func TestRecentHealthProbesKeepsTheNewestThreeClipped(t *testing.T) {
	start := time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)
	var probes []container.HealthProbe
	for i := range 5 {
		at := start.Add(time.Duration(i) * 30 * time.Second)
		probes = append(probes, container.HealthProbe{Start: at, End: at.Add(time.Second), ExitCode: i, Output: "probe " + string(rune('a'+i)) + "\n"})
	}
	probes[4].Output = strings.Repeat("é", 400) // 800 bytes

	got := recentHealthProbes(context.Background(), fakeHealthLog{probes: probes})

	require.Len(t, got, 3)
	assert.Equal(t, "probe c", got[0].Output, "oldest first, trailing newline trimmed")
	assert.Equal(t, "2026-11-02T10:01:00Z", got[0].Start)
	assert.Equal(t, "2026-11-02T10:01:01Z", got[0].End)
	assert.EqualValues(t, 2, got[0].ExitCode)
	assert.EqualValues(t, 4, got[2].ExitCode)
	assert.Len(t, got[2].Output, healthLogOutputBytes, "clipped on a rune boundary")
}

func TestRecentHealthProbesLeavesThemOutOnError(t *testing.T) {
	assert.Empty(t, recentHealthProbes(context.Background(), fakeHealthLog{err: errors.New("engine down")}))
	assert.Empty(t, recentHealthProbes(context.Background(), fakeHealthLog{err: errors.ErrUnsupported}))
	assert.Empty(t, recentHealthProbes(context.Background(), fakeHealthLog{}))
}

// A host that cannot read healthcheck runs still answers the inspect.
func TestInspectContainerWithoutHealthLogSupport(t *testing.T) {
	hosts := &fakeHostService{
		containers: []container.Container{{ID: "web", Name: "web", Host: "h", State: "running", Health: "unhealthy"}},
		clients:    map[string]*fakeClientService{"h": newFakeClientService("h")},
	}

	resp := ExecuteTool(context.Background(), toolInspectContainer, `{"container_id":"web","host":"h"}`, ToolDeps{HostService: hosts, Principal: Principal{Kind: PrincipalInstance}})

	require.True(t, resp.Success, resp.Error)
	assert.Equal(t, "unhealthy", resp.GetInspectContainer().Health)
	assert.Empty(t, resp.GetInspectContainer().HealthLog)
}

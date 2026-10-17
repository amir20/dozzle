package cloud

import (
	"context"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// slowHosts never answers FindContainer for the ids in block, to exercise the
// deadline.
type slowHosts struct {
	*fakeHostService
	block   map[string]bool
	release chan struct{}
}

func (s *slowHosts) FindContainer(host, id string, labels container.ContainerLabels) (*container.ContainerService, error) {
	if s.block[id] {
		<-s.release
	}
	return s.fakeHostService.FindContainer(host, id, labels)
}

func TestInspectContainersAnswersEveryContainerInOneCall(t *testing.T) {
	hosts := &fakeHostService{
		containers: []container.Container{
			{ID: "vpn", Name: "gluetun", Host: "h", State: "running", RestartCount: 2},
			{ID: "qb", Name: "qbittorrent", Host: "h", State: "running", NetworkMode: "container:vpn", RestartPolicy: "unless-stopped"},
			{ID: "oom", Name: "worker", Host: "h", State: "exited", OOMKilled: true, ExitCode: 137, MemoryLimit: 512 << 20},
		},
		clients: map[string]*fakeClientService{"h": newFakeClientService("h")},
	}

	resp := ExecuteTool(context.Background(), toolInspectContainers, "", ToolDeps{HostService: hosts, Principal: Principal{Kind: PrincipalInstance}})
	require.True(t, resp.Success, resp.Error)
	res := resp.GetInspectContainers()
	require.NotNil(t, res)
	assert.False(t, res.DeadlineExceeded)
	assert.False(t, res.Truncated)
	require.Len(t, res.Containers, 3)

	byID := map[string]int{}
	for i, c := range res.Containers {
		byID[c.Id] = i
	}
	qb := res.Containers[byID["qb"]]
	assert.Equal(t, "container:vpn", qb.NetworkMode)
	assert.Equal(t, "unless-stopped", qb.RestartPolicy)
	assert.Equal(t, "h", qb.HostName)
	oom := res.Containers[byID["oom"]]
	assert.True(t, oom.OomKilled)
	assert.EqualValues(t, 137, oom.ExitCode)
	assert.EqualValues(t, 512<<20, oom.MemoryLimit)
	assert.EqualValues(t, 2, res.Containers[byID["vpn"]].RestartCount)
}

func TestInspectContainersLeavesOutWhatItCannotFind(t *testing.T) {
	hosts := &fakeHostService{
		containers: []container.Container{
			{ID: "a", Name: "a", Host: "h"},
			{ID: "gone", Name: "gone", Host: "missing-host"},
		},
		clients: map[string]*fakeClientService{"h": newFakeClientService("h")},
	}
	res := ExecuteTool(context.Background(), toolInspectContainers, "", ToolDeps{HostService: hosts}).GetInspectContainers()
	require.Len(t, res.Containers, 1)
	assert.Equal(t, "a", res.Containers[0].Id)
}

func TestInspectContainersReturnsAtTheDeadline(t *testing.T) {
	hosts := &slowHosts{
		fakeHostService: &fakeHostService{
			containers: []container.Container{{ID: "fast", Host: "h"}, {ID: "stuck", Host: "h"}},
			clients:    map[string]*fakeClientService{"h": newFakeClientService("h")},
		},
		block:   map[string]bool{"stuck": true},
		release: make(chan struct{}),
	}
	defer close(hosts.release)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	start := time.Now()
	res := ExecuteTool(ctx, toolInspectContainers, "", ToolDeps{HostService: hosts}).GetInspectContainers()
	assert.Less(t, time.Since(start), 2*time.Second, "a stuck read must not hold the response")
	assert.True(t, res.DeadlineExceeded)
	require.Len(t, res.Containers, 1)
	assert.Equal(t, "fast", res.Containers[0].Id)
}

func TestInspectContainersIsNeverOfferedToAModel(t *testing.T) {
	for _, d := range AvailableTools(ToolDeps{}) {
		if d.Name == toolInspectContainers {
			assert.True(t, d.Internal)
			assert.True(t, d.ReadOnly)
			return
		}
	}
	t.Fatal("inspect_containers must be listed")
}

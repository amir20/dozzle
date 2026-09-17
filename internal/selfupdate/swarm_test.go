package selfupdate

import (
	"context"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/moby/moby/api/types/image"
	"github.com/moby/moby/api/types/swarm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const serviceID = "svc123"

// swarmFake is newFake with Dozzle running as slot 1 of a replicated service
// whose task swarm pinned to a digest, on a manager node.
func swarmFake() *fakeDocker {
	f := newFake()
	self := f.containers[selfID]
	self.Config.Image = "amir20/dozzle:latest@sha256:1111111111111111111111111111111111111111111111111111111111111111"
	self.Config.Labels = map[string]string{
		swarmLabel:          "dozzle_dozzle",
		swarmServiceIDLabel: serviceID,
		swarmTaskNameLabel:  "dozzle_dozzle.1.y04ozihyqwcgbnywww2s2q15y",
	}
	f.containers[selfID] = self
	f.service = &swarm.Service{
		ID:      serviceID,
		Version: swarm.Version{Index: 42}, UpdatedAt: time.Now().Add(-24 * time.Hour),
		Spec: swarm.ServiceSpec{TaskTemplate: swarm.TaskSpec{
			ContainerSpec: &swarm.ContainerSpec{Image: "amir20/dozzle:latest@sha256:1111111111111111111111111111111111111111111111111111111111111111"},
			ForceUpdate:   7,
		}},
	}
	return f
}

func TestTaskImageRef(t *testing.T) {
	assert.Equal(t, "amir20/dozzle:master", taskImageRef("amir20/dozzle:master@sha256:abc"))
	assert.Equal(t, "amir20/dozzle:master", taskImageRef("amir20/dozzle:master"))
	assert.Equal(t, "localhost:5000/dozzle:latest", taskImageRef("localhost:5000/dozzle:latest@sha256:abc"))
}

func TestSwarmPrimary(t *testing.T) {
	assert.True(t, SwarmPrimary(map[string]string{swarmTaskNameLabel: "dozzle_dozzle.1.y04ozihyqwcgbnywww2s2q15y"}))
	assert.False(t, SwarmPrimary(map[string]string{swarmTaskNameLabel: "dozzle_dozzle.2.abcdefabcdefabcdefabcdefa"}))
	assert.True(t, SwarmPrimary(map[string]string{swarmTaskNameLabel: "dozzle_dozzle.mv7alucl9n54o6rdub8tvlgjf.abcdef"}), "a global task has no slot and goes ahead")
	assert.True(t, SwarmPrimary(map[string]string{}), "no task name is not a replica")
}

func TestSupportSwarm(t *testing.T) {
	f := swarmFake()
	ok, reason, img := support(context.Background(), f, selfID)
	assert.True(t, ok, reason)
	assert.Equal(t, "amir20/dozzle:latest", img, "the digest swarm pinned the task to is dropped, so the tag does not read as pinned")

	f.service = nil
	ok, reason, _ = support(context.Background(), f, selfID)
	assert.False(t, ok)
	assert.Equal(t, ReasonSwarmWorker, reason)

	f = swarmFake()
	self := f.containers[selfID]
	self.Config.Image = "amir20/dozzle:v8.12.0@sha256:abc"
	f.containers[selfID] = self
	_, reason, _ = support(context.Background(), f, selfID)
	assert.Equal(t, ReasonPinnedTag, reason, "a version tag is still pinned once the digest is dropped")
}

func TestStartSwarmUpdatesTheService(t *testing.T) {
	f := swarmFake()
	var got []string
	updated, err := start(context.Background(), f, selfID, func(p container.UpdateProgress) { got = append(got, p.Status) })
	require.NoError(t, err)
	assert.True(t, updated)
	assert.Equal(t, []string{"pulling", "recreating", "done"}, got)
	assert.Equal(t, []string{"pull amir20/dozzle:latest", "service update " + serviceID}, f.calls, "no helper container for a swarm task")

	require.Len(t, f.serviceUpdates, 1)
	opts := f.serviceUpdates[0]
	assert.Equal(t, uint64(42), opts.Version.Index)
	assert.Equal(t, "amir20/dozzle:latest", opts.Spec.TaskTemplate.ContainerSpec.Image, "the tag, so the manager resolves the new digest")
	assert.Equal(t, uint64(8), opts.Spec.TaskTemplate.ForceUpdate)
}

func TestStartSwarmUpToDate(t *testing.T) {
	f := swarmFake()
	f.images["amir20/dozzle:latest"] = image.InspectResponse{ID: oldImgID}
	updated, err := start(context.Background(), f, selfID, func(container.UpdateProgress) {})
	require.NoError(t, err)
	assert.False(t, updated)
	assert.Empty(t, f.serviceUpdates)
}

func TestStartSwarmRefusesWhileUpdating(t *testing.T) {
	f := swarmFake()
	f.service.UpdateStatus = &swarm.UpdateStatus{State: swarm.UpdateStateUpdating}
	_, err := start(context.Background(), f, selfID, func(container.UpdateProgress) {})
	require.ErrorContains(t, err, "already in progress")
	assert.Empty(t, f.serviceUpdates)
}

func TestStartSwarmSkipsARolloutThatJustHappened(t *testing.T) {
	f := swarmFake()
	f.service.Spec.TaskTemplate.ContainerSpec.Image = "amir20/dozzle:latest"
	f.service.UpdatedAt = time.Now().Add(-time.Minute)
	updated, err := start(context.Background(), f, selfID, func(container.UpdateProgress) {})
	require.NoError(t, err)
	assert.True(t, updated, "another replica already rolled the service onto this tag")
	assert.Empty(t, f.serviceUpdates)
}

func TestStartSwarmOnWorkerFails(t *testing.T) {
	f := swarmFake()
	f.service = nil
	var last container.UpdateProgress
	_, err := start(context.Background(), f, selfID, func(p container.UpdateProgress) { last = p })
	require.Error(t, err)
	assert.Equal(t, "error", last.Status)
	assert.Empty(t, f.serviceUpdates)
}

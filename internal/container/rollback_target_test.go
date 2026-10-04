package container

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	rtCurrent = "sha256:cccc"
	rtPrev    = "sha256:bbbb"
	rtOlder   = "sha256:aaaa"
	rtID      = "0123456789ab"
)

func dozzleUpdated() map[string]string {
	return map[string]string{
		PreviousImageLabel: rtOlder,
		PreviousRefLabel:   "app@sha256:older",
		UpdateSourceLabel:  UpdateSourceDozzle,
	}
}

func TestRollbackTargetOfLabels(t *testing.T) {
	target, err := RollbackTargetOf(rtID+"ffff", rtCurrent, dozzleUpdated(), nil)
	require.NoError(t, err)
	assert.Equal(t, RollbackTarget{ImageID: rtOlder, Ref: "app@sha256:older"}, target)

	_, err = RollbackTargetOf(rtID, rtCurrent, nil, nil)
	require.ErrorIs(t, err, ErrNoRollbackTarget, "never updated")

	_, err = RollbackTargetOf(rtID, rtOlder, dozzleUpdated(), nil)
	require.ErrorIs(t, err, ErrNoRollbackTarget, "already runs it")

	labels := dozzleUpdated()
	labels[UpdateSourceLabel] = UpdateSourceRollback
	_, err = RollbackTargetOf(rtID, rtCurrent, labels, nil)
	require.ErrorIs(t, err, ErrNoRollbackTarget, "a rollback has no target of its own")

	labels = dozzleUpdated()
	labels["com.docker.swarm.service.name"] = "web"
	_, err = RollbackTargetOf(rtID, rtCurrent, labels, nil)
	require.ErrorIs(t, err, ErrRollbackUnsupported)
}

// Watchtower copies Dozzle's labels onto the container it recreates. Its event
// is what says where the container came from.
func TestRollbackTargetOfEventsWinOverCopiedLabels(t *testing.T) {
	watchtower := ContainerUpdateEvent{NewID: rtID, FromImageID: rtPrev, FromDigest: "app@sha256:prev", Source: UpdateSourceWatchtower}
	unrelated := ContainerUpdateEvent{NewID: "ffffffffffff", FromImageID: rtOlder, Source: UpdateSourceDozzle}

	target, err := RollbackTargetOf(rtID, rtCurrent, dozzleUpdated(), []ContainerUpdateEvent{watchtower, unrelated})
	require.NoError(t, err)
	assert.Equal(t, RollbackTarget{ImageID: rtPrev, Ref: "app@sha256:prev"}, target, "not the stale label's older image")

	// After a rollback, Watchtower copies update-source=rollback too.
	labels := dozzleUpdated()
	labels[UpdateSourceLabel] = UpdateSourceRollback
	target, err = RollbackTargetOf(rtID, rtCurrent, labels, []ContainerUpdateEvent{watchtower})
	require.NoError(t, err)
	assert.Equal(t, rtPrev, target.ImageID, "a container Watchtower updated after a rollback can be rolled back")

	// With no labels at all, as after an update by compose.
	target, err = RollbackTargetOf(rtID, rtCurrent, nil, []ContainerUpdateEvent{watchtower})
	require.NoError(t, err)
	assert.Equal(t, rtPrev, target.ImageID)

	// The event of a rollback says the container has no target.
	rollback := ContainerUpdateEvent{NewID: rtID, FromImageID: rtPrev, Source: UpdateSourceRollback}
	_, err = RollbackTargetOf(rtID, rtCurrent, dozzleUpdated(), []ContainerUpdateEvent{watchtower, rollback})
	require.ErrorIs(t, err, ErrNoRollbackTarget)

	// A rolled back swap put this container back; the update that created
	// it is the one that counts.
	undone := ContainerUpdateEvent{NewID: rtID, FromImageID: rtCurrent, ToImageID: "sha256:dddd", RolledBack: true}
	target, err = RollbackTargetOf(rtID, rtCurrent, nil, []ContainerUpdateEvent{watchtower, undone})
	require.NoError(t, err)
	assert.Equal(t, rtPrev, target.ImageID)
}

type historyService struct {
	ClientService
	events []ContainerUpdateEvent
}

func (h historyService) RecentUpdates() []ContainerUpdateEvent { return h.events }

func (historyService) SubscribeUpdates(context.Context, chan<- ContainerUpdateEvent) {}

// The container service reads the host's events from any service that keeps
// them, which is how an agent host gets the same target as a local one.
func TestContainerServiceRollbackTarget(t *testing.T) {
	c := Container{ID: rtID, ImageID: rtCurrent, Labels: dozzleUpdated()}
	watchtower := ContainerUpdateEvent{NewID: rtID, FromImageID: rtPrev, Source: UpdateSourceWatchtower}

	target, err := NewContainerService(historyService{events: []ContainerUpdateEvent{watchtower}}, c).RollbackTarget()
	require.NoError(t, err)
	assert.Equal(t, rtPrev, target.ImageID)

	// A service without history falls back to the labels.
	target, err = NewContainerService(struct{ ClientService }{}, c).RollbackTarget()
	require.NoError(t, err)
	assert.Equal(t, rtOlder, target.ImageID)
}

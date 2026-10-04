package cloud

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/notification"
	pb "github.com/amir20/dozzle/proto/cloud"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeUpdateHosts is an UpdateStreamHostService over a fixed set of
// containers on one host. hidden ids are what the instance's label filter
// keeps from Cloud.
type fakeUpdateHosts struct {
	containers []container.Container
	// inspected stands in for an inspect, which knows what a list entry may not.
	inspected map[string]container.Container
	recent    []container.ContainerUpdateEvent
	subs      chan chan<- container.ContainerUpdateEvent
}

func (f *fakeUpdateHosts) ListAllContainers(_ container.ContainerLabels) ([]container.Container, []error) {
	return f.containers, nil
}

func (f *fakeUpdateHosts) FindContainer(host, id string, _ container.ContainerLabels) (*container.ContainerService, error) {
	if c, ok := f.inspected[id]; ok && c.Host == host {
		return container.NewContainerService(nil, c), nil
	}
	for _, c := range f.containers {
		if c.Host == host && c.ID == id {
			return container.NewContainerService(nil, c), nil
		}
	}
	return nil, container.ErrContainerNotFound
}

func (f *fakeUpdateHosts) Hosts() []container.Host {
	return []container.Host{{ID: "nas-id", Name: "nas"}}
}

func (f *fakeUpdateHosts) RecentUpdates() []container.ContainerUpdateEvent { return f.recent }

func (f *fakeUpdateHosts) SubscribeUpdates(_ context.Context, ch chan<- container.ContainerUpdateEvent) {
	if f.subs != nil {
		f.subs <- ch
	}
}

type capturedPushes struct {
	mu        sync.Mutex
	updates   []*pb.ContainerUpdate
	snapshots []*pb.ImageSnapshot
	fail      bool
	attempts  int
}

func (c *capturedPushes) send(resp *pb.ToolResponse) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.attempts++
	if c.fail {
		return errors.New("stream closed")
	}
	if u := resp.GetContainerUpdate(); u != nil {
		c.updates = append(c.updates, u)
	}
	if s := resp.GetImageSnapshot(); s != nil {
		c.snapshots = append(c.snapshots, s)
	}
	return nil
}

func (c *capturedPushes) sentUpdates() []*pb.ContainerUpdate {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]*pb.ContainerUpdate(nil), c.updates...)
}

func rule(t *testing.T, sub *notification.Subscription) *notification.Subscription {
	t.Helper()
	sub.Enabled = true
	require.NoError(t, sub.CompileExpressions())
	return sub
}

// lifecycleRule is what the welcome modal creates: every container, die events,
// notifying Dozzle Cloud.
func lifecycleRule(t *testing.T) *notification.Subscription {
	return rule(t, &notification.Subscription{
		DispatcherID:        cloudDispatcherID,
		ContainerExpression: `name == "immich"`,
		EventExpression:     `name == "die"`,
	})
}

var (
	updateAt     = time.Date(2026, 10, 3, 3, 4, 0, 0, time.UTC)
	oldStartedAt = time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
)

func immichUpdate(source string) container.ContainerUpdateEvent {
	return container.ContainerUpdateEvent{
		Host:         "nas-id",
		Name:         "immich",
		OldID:        "old000000000",
		NewID:        "new000000000",
		FromRef:      "ghcr.io/immich-app/immich-server:release",
		ToRef:        "ghcr.io/immich-app/immich-server:release",
		FromDigest:   "ghcr.io/immich-app/immich-server@sha256:aaa",
		ToDigest:     "ghcr.io/immich-app/immich-server@sha256:bbb",
		FromImageID:  "sha256:111",
		ToImageID:    "sha256:222",
		OldStartedAt: oldStartedAt,
		At:           updateAt,
		Source:       source,
		RunID:        "run1",
	}
}

func immichContainer(labels map[string]string) container.Container {
	return container.Container{
		ID:          "new000000000",
		Name:        "immich",
		Host:        "nas-id",
		Image:       "ghcr.io/immich-app/immich-server:release",
		ImageID:     "sha256:222",
		ImageDigest: "ghcr.io/immich-app/immich-server@sha256:bbb",
		State:       "running",
		StartedAt:   updateAt,
		Labels:      labels,
	}
}

func newTestPusher(hosts *fakeUpdateHosts, deps ToolDeps) (*updatePusher, *capturedPushes) {
	sent := &capturedPushes{}
	return newUpdatePusher(hosts, deps, newUpdateLedger(), sent.send), sent
}

func TestUpdatePusher_Consent(t *testing.T) {
	watchedAll := func(container.ContainerUpdateEvent) bool { return true }

	tests := []struct {
		name    string
		source  string
		rules   []*notification.Subscription
		watched func(container.ContainerUpdateEvent) bool
		consent string // empty: not sent
	}{
		{name: "the schedule is its own opt-in", source: container.UpdateSourceSchedule, consent: consentSchedule},
		{name: "a manual update with nothing agreed is not sent", source: container.UpdateSourceDozzle},
		{name: "Watchtower with nothing agreed is not sent", source: container.UpdateSourceWatchtower},
		{name: "the watch checkbox", source: container.UpdateSourceDozzle, watched: watchedAll, consent: consentCheckbox},
		{name: "a lifecycle rule that notifies Cloud", source: container.UpdateSourceExternal, rules: []*notification.Subscription{lifecycleRule(t)}, consent: consentRule},
		{name: "the schedule wins over a rule", source: container.UpdateSourceSchedule, rules: []*notification.Subscription{lifecycleRule(t)}, consent: consentSchedule},
		{
			name:   "a rule for a webhook is not consent",
			source: container.UpdateSourceExternal,
			rules: []*notification.Subscription{rule(t, &notification.Subscription{
				DispatcherID: 3, ContainerExpression: "true", EventExpression: `name == "die"`,
			})},
		},
		{
			name:   "a log rule is not a lifecycle rule",
			source: container.UpdateSourceExternal,
			rules: []*notification.Subscription{rule(t, &notification.Subscription{
				DispatcherID: cloudDispatcherID, ContainerExpression: "true", LogExpression: `level == "error"`,
			})},
		},
		{
			name:   "a disabled rule is not consent",
			source: container.UpdateSourceExternal,
			rules: func() []*notification.Subscription {
				r := lifecycleRule(t)
				r.Enabled = false
				return []*notification.Subscription{r}
			}(),
		},
		{
			name:   "a rule for another container is not consent",
			source: container.UpdateSourceExternal,
			rules: []*notification.Subscription{rule(t, &notification.Subscription{
				DispatcherID: cloudDispatcherID, ContainerExpression: `name == "postgres"`, EventExpression: `name == "die"`,
			})},
		},
		{
			name:   "a rule can match on the host name",
			source: container.UpdateSourceExternal,
			rules: []*notification.Subscription{rule(t, &notification.Subscription{
				DispatcherID: cloudDispatcherID, ContainerExpression: `hostName == "nas"`, EventExpression: `name == "die"`,
			})},
			consent: consentRule,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hosts := &fakeUpdateHosts{containers: []container.Container{immichContainer(nil)}}
			p, sent := newTestPusher(hosts, ToolDeps{
				NotificationService: &fakeNotificationService{subs: tt.rules},
				UpdateWatched:       tt.watched,
			})
			p.push(immichUpdate(tt.source))

			updates := sent.sentUpdates()
			if tt.consent == "" {
				assert.Empty(t, updates)
				return
			}
			require.Len(t, updates, 1)
			assert.Equal(t, tt.consent, updates[0].Consent)
		})
	}
}

func TestUpdatePusher_SendsTheUpdate(t *testing.T) {
	hosts := &fakeUpdateHosts{containers: []container.Container{immichContainer(nil)}}
	p, sent := newTestPusher(hosts, ToolDeps{})
	p.push(immichUpdate(container.UpdateSourceSchedule))

	updates := sent.sentUpdates()
	require.Len(t, updates, 1)
	u := updates[0]
	assert.Equal(t, "nas-id", u.Host)
	assert.Equal(t, "immich", u.Name)
	assert.Equal(t, "old000000000", u.OldContainerId)
	assert.Equal(t, "new000000000", u.NewContainerId)
	assert.Equal(t, "ghcr.io/immich-app/immich-server:release", u.FromRef)
	// cloud.proto carries the bare digest; the repository is in the refs.
	assert.Equal(t, "sha256:aaa", u.FromDigest)
	assert.Equal(t, "sha256:bbb", u.ToDigest)
	assert.Equal(t, "sha256:111", u.FromImageId)
	assert.Equal(t, "sha256:222", u.ToImageId)
	assert.Equal(t, oldStartedAt.UnixNano(), u.OldStartedAt)
	assert.Equal(t, updateAt.UnixNano(), u.At)
	assert.Equal(t, container.UpdateSourceSchedule, u.Source)
	assert.Equal(t, "run1", u.RunId)
	assert.False(t, u.RolledBack)
}

// A swap that rolled itself back is pushed too, flagged: Cloud records the
// verdict and needs no judge.
func TestUpdatePusher_RolledBackSwap(t *testing.T) {
	old := immichContainer(nil)
	old.ID = "old000000000"
	old.ImageID = "sha256:111"
	hosts := &fakeUpdateHosts{containers: []container.Container{old}}
	p, sent := newTestPusher(hosts, ToolDeps{})

	event := immichUpdate(container.UpdateSourceSchedule)
	event.NewID = "old000000000"
	event.RolledBack = true
	p.push(event)

	updates := sent.sentUpdates()
	require.Len(t, updates, 1)
	assert.True(t, updates[0].RolledBack)
	assert.Equal(t, "old000000000", updates[0].NewContainerId)
}

// Rolling back an update Cloud was told about ends that update, so it goes
// under the same consent. A rollback of one Cloud never heard of does not.
func TestUpdatePusher_RollbackFollowsItsUpdate(t *testing.T) {
	hosts := &fakeUpdateHosts{containers: []container.Container{immichContainer(nil)}}
	p, sent := newTestPusher(hosts, ToolDeps{})

	rollback := immichUpdate(container.UpdateSourceRollback)
	rollback.OldID, rollback.NewID = "new000000000", "back00000000"
	rollback.At = updateAt.Add(time.Hour)
	back := immichContainer(nil)
	back.ID = "back00000000"
	hosts.containers = append(hosts.containers, back)

	p.push(rollback)
	assert.Empty(t, sent.sentUpdates(), "nothing agreed for this container yet")

	p.push(immichUpdate(container.UpdateSourceSchedule))
	rollback.At = updateAt.Add(2 * time.Hour)
	p.push(rollback)

	updates := sent.sentUpdates()
	require.Len(t, updates, 2)
	assert.Equal(t, container.UpdateSourceRollback, updates[1].Source)
	assert.Equal(t, consentSchedule, updates[1].Consent)
}

// The instance's label filter keeps a container from Cloud, its updates too.
func TestUpdatePusher_SkipsHiddenContainers(t *testing.T) {
	p, sent := newTestPusher(&fakeUpdateHosts{}, ToolDeps{})
	p.push(immichUpdate(container.UpdateSourceSchedule))
	assert.Empty(t, sent.sentUpdates())
}

// What failed to send goes out on the next connection, and what was sent does
// not go out again.
func TestUpdatePusher_ReplaysRecentUpdatesOnce(t *testing.T) {
	event := immichUpdate(container.UpdateSourceSchedule)
	hosts := &fakeUpdateHosts{containers: []container.Container{immichContainer(nil)}, recent: []container.ContainerUpdateEvent{event}}
	ledger := newUpdateLedger()

	connect := func(fail bool) *capturedPushes {
		sent := &capturedPushes{fail: fail}
		hosts.subs = make(chan chan<- container.ContainerUpdateEvent, 1)
		p := newUpdatePusher(hosts, ToolDeps{}, ledger, sent.send)
		ctx, cancel := context.WithCancel(t.Context())
		done := make(chan struct{})
		go func() {
			p.run(ctx)
			close(done)
		}()
		<-hosts.subs
		// The replay runs after the subscription; wait for it to try, or for
		// the ledger to say there is nothing left to try.
		assert.Eventually(t, func() bool {
			sent.mu.Lock()
			defer sent.mu.Unlock()
			return sent.attempts > 0 || ledger.hasSent(event)
		}, time.Second, time.Millisecond)
		cancel()
		<-done
		return sent
	}

	assert.Empty(t, connect(true).sentUpdates())
	assert.Len(t, connect(false).sentUpdates(), 1)
	assert.Empty(t, connect(false).sentUpdates())
}

// Live updates arrive on the subscription once the snapshot and the replay
// are out.
func TestUpdatePusher_PushesLiveUpdates(t *testing.T) {
	hosts := &fakeUpdateHosts{
		containers: []container.Container{immichContainer(nil)},
		subs:       make(chan chan<- container.ContainerUpdateEvent, 1),
	}
	p, sent := newTestPusher(hosts, ToolDeps{})
	go p.run(t.Context())

	ch := <-hosts.subs
	ch <- immichUpdate(container.UpdateSourceSchedule)
	assert.Eventually(t, func() bool { return len(sent.sentUpdates()) == 1 }, time.Second, time.Millisecond)
}

func TestUpdatePusher_Snapshot(t *testing.T) {
	auto := map[string]string{container.AutoUpdateLabel: "true"}
	listed := immichContainer(auto)
	// A list entry that has not been inspected yet.
	listed.ImageDigest = ""
	listed.StartedAt = time.Time{}

	postgres := immichContainer(nil)
	postgres.ID, postgres.Name = "pg0000000000", "postgres"
	redis := immichContainer(nil)
	redis.ID, redis.Name = "rd0000000000", "redis"
	created := immichContainer(auto)
	created.ID, created.Name, created.State = "cr0000000000", "migrate", "created"
	built := immichContainer(auto)
	built.ID, built.Name, built.ImageID = "bl0000000000", "local-build", ""

	hosts := &fakeUpdateHosts{
		containers: []container.Container{listed, postgres, redis, created, built},
		inspected:  map[string]container.Container{listed.ID: immichContainer(auto)},
	}
	p, sent := newTestPusher(hosts, ToolDeps{NotificationService: &fakeNotificationService{subs: []*notification.Subscription{
		rule(t, &notification.Subscription{DispatcherID: cloudDispatcherID, ContainerExpression: `name == "postgres"`, EventExpression: `name == "die"`}),
	}}})
	require.NoError(t, p.sendSnapshot())

	require.Len(t, sent.snapshots, 1)
	entries := sent.snapshots[0].Entries
	require.Len(t, entries, 2, "the labelled container and the one a rule covers")
	byName := map[string]*pb.ImageSnapshotEntry{}
	for _, e := range entries {
		byName[e.Name] = e
	}
	immich := byName["immich"]
	require.NotNil(t, immich)
	assert.Equal(t, "nas-id", immich.Host)
	assert.Equal(t, "new000000000", immich.ContainerId)
	assert.Equal(t, "sha256:222", immich.ImageId)
	assert.Equal(t, "sha256:bbb", immich.Digest, "filled in by an inspect")
	assert.Equal(t, "ghcr.io/immich-app/immich-server:release", immich.Ref)
	assert.Equal(t, updateAt.UnixNano(), immich.StartedAt)
	assert.NotNil(t, byName["postgres"])
}

// Nothing covered, nothing sent: no empty snapshot on every connect.
func TestUpdatePusher_NoSnapshotWithoutConsent(t *testing.T) {
	hosts := &fakeUpdateHosts{containers: []container.Container{immichContainer(nil)}}
	p, sent := newTestPusher(hosts, ToolDeps{})
	require.NoError(t, p.sendSnapshot())
	assert.Empty(t, sent.snapshots)
}

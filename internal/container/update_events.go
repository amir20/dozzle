package container

import (
	"context"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/puzpuzpuz/xsync/v4"
	"github.com/rs/zerolog/log"
)

// ContainerUpdateEvent records that the container under a name now runs a
// different image than the last one seen under that name. The name is the
// engine's (EngineName), never the display name: dev.dozzle.name and other
// labels can give two different containers the same display name.
//
// It is worked out from starts by name, never from what started the update, so
// every way of updating a container reads the same: Watchtower's stop, remove,
// create, start; compose's create under a temporary name and rename; and
// Dozzle's own swap. A restart keeps its image and is not an update.
type ContainerUpdateEvent struct {
	Host string `json:"host"`
	// Name is the display name, for the UI. EngineName is the engine's own
	// name for the container, which is what the event is worked out from.
	Name       string `json:"name"`
	EngineName string `json:"engineName"`
	// OldID is the container that ran before. NewID is the one that runs under
	// the name now: the replacement, or for a rolled back swap the old
	// container put back (a recreation of it for a --rm one).
	OldID string `json:"oldId"`
	NewID string `json:"newId"`
	// FromRef and ToRef are the references the containers were created from,
	// usually the same tag.
	FromRef string `json:"fromRef,omitempty"`
	ToRef   string `json:"toRef,omitempty"`
	// FromDigest and ToDigest are repo@sha256:... Either is empty when the
	// image was built locally or its digest was never seen.
	FromDigest  string `json:"fromDigest,omitempty"`
	ToDigest    string `json:"toDigest,omitempty"`
	FromImageID string `json:"fromImageId,omitempty"`
	ToImageID   string `json:"toImageId,omitempty"`
	// OldStartedAt is when the old container last started: the start of the
	// previous version's startup window, which is the baseline to compare the
	// new version's first minutes against.
	OldStartedAt time.Time `json:"oldStartedAt,omitzero"`
	At           time.Time `json:"at"`
	// Source is one of the UpdateSource* values.
	Source string `json:"source"`
	// RunID groups the updates of one bulk run, such as one scheduled night.
	RunID string `json:"runId,omitempty"`
	// RolledBack is set when Dozzle's swap put the old container back because
	// the replacement failed. The update did not happen: ToImageID is what was
	// tried, and the container still runs FromImageID.
	RolledBack bool `json:"rolledBack,omitempty"`
}

const (
	// maxUpdateEvents is how many update events a host keeps.
	maxUpdateEvents = 20
	// A name whose container is gone is kept this long, so the start of the
	// container that replaces it still finds it. Watchtower removes the old
	// container before it creates the new one.
	goneRecordTTL = time.Hour
	// pruneRecordsAbove is the record count above which gone records are pruned.
	pruneRecordsAbove = 256
	// watchtowerLabelPrefix starts every label Watchtower reads.
	watchtowerLabelPrefix = "com.centurylinklabs.watchtower"
	// restoredRefLabel is set on a --rm container that a rolled back swap had
	// to recreate (swap.ImageRefLabel, which this package cannot import).
	restoredRefLabel = "dev.dozzle.self-update.image"
)

// imageRecord is the last image seen under an engine name.
type imageRecord struct {
	containerID string
	imageID     string
	digest      string
	ref         string
	startedAt   time.Time
	goneAt      time.Time
}

// updateTracker turns starts into update events for one host. It is fed by the
// store's event loop and refreshes, and read from anywhere. A nil tracker (a
// Store built by hand in a test) records nothing.
type updateTracker struct {
	mu          sync.Mutex
	byName      map[string]imageRecord // keyed by EngineName
	events      []ContainerUpdateEvent
	subscribers *xsync.Map[context.Context, chan<- ContainerUpdateEvent]
	now         func() time.Time
}

func newUpdateTracker() *updateTracker {
	return &updateTracker{
		byName:      make(map[string]imageRecord),
		subscribers: xsync.NewMap[context.Context, chan<- ContainerUpdateEvent](),
		now:         time.Now,
	}
}

// imageIdentity is what a container runs: the image id, or for k8s, which has
// none, the digest.
func imageIdentity(c Container) string {
	if c.ImageID != "" {
		return c.ImageID
	}
	return c.ImageDigest
}

// seen records a container a list or an inspect found, without treating it as a
// start. It fills a name nothing was recorded under yet, so the first update
// after Dozzle starts is not missed, and refreshes the record of the container
// already there (an inspect knows the digest a list entry does not). A
// container that was only created never ran, so it says nothing about the name.
func (t *updateTracker) seen(c Container) {
	identity := imageIdentity(c)
	if t == nil || identity == "" || c.EngineName == "" {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	prev, ok := t.byName[c.EngineName]
	switch {
	case !ok && c.State != "created":
		t.byName[c.EngineName] = recordOf(c, imageRecord{})
	case ok && prev.containerID == c.ID:
		t.byName[c.EngineName] = recordOf(c, prev)
	}
}

// recordOf is c as a record. A list entry has no digest or start time, so what
// an inspect of the same container already found is kept.
func recordOf(c Container, prev imageRecord) imageRecord {
	r := imageRecord{
		containerID: c.ID,
		imageID:     imageIdentity(c),
		digest:      c.ImageDigest,
		ref:         c.Image,
		startedAt:   c.StartedAt,
	}
	if prev.containerID == c.ID && prev.imageID == r.imageID {
		if r.digest == "" {
			r.digest = prev.digest
		}
		if r.startedAt.IsZero() {
			r.startedAt = prev.startedAt
		}
	}
	return r
}

// started records a start and returns the update it amounts to, if any.
func (t *updateTracker) started(c Container) (ContainerUpdateEvent, bool) {
	identity := imageIdentity(c)
	if t == nil || identity == "" || c.EngineName == "" {
		return ContainerUpdateEvent{}, false
	}

	t.mu.Lock()
	prev, ok := t.byName[c.EngineName]
	t.byName[c.EngineName] = recordOf(c, prev)
	t.pruneLocked()
	if !ok || prev.containerID == c.ID || prev.imageID == identity || t.restoringLocked(c, prev) {
		t.mu.Unlock()
		return ContainerUpdateEvent{}, false
	}

	event := ContainerUpdateEvent{
		Host:         c.Host,
		Name:         c.Name,
		EngineName:   c.EngineName,
		OldID:        prev.containerID,
		NewID:        c.ID,
		FromRef:      prev.ref,
		ToRef:        c.Image,
		FromDigest:   prev.digest,
		ToDigest:     c.ImageDigest,
		FromImageID:  prev.imageID,
		ToImageID:    identity,
		OldStartedAt: prev.startedAt,
		At:           c.StartedAt,
	}
	if event.At.IsZero() {
		event.At = t.now()
	}
	event.Source, event.RunID = updateSource(c, prev)
	t.appendLocked(event)
	t.mu.Unlock()

	t.broadcast(event)
	return event, true
}

// updateSource works out what made the update from the new container's labels.
// Dozzle's own labels count only when they name the image the old container
// ran: Watchtower copies every label onto the container it recreates, so one
// that recreates a container Dozzle once updated carries Dozzle's labels from
// that earlier update.
func updateSource(c Container, prev imageRecord) (source, runID string) {
	if previous := c.Labels[PreviousImageLabel]; previous != "" && previous == prev.imageID {
		source = c.Labels[UpdateSourceLabel]
		if source == "" {
			source = UpdateSourceDozzle
		}
		return source, c.Labels[UpdateRunLabel]
	}
	for key, value := range c.Labels {
		if strings.HasPrefix(key, watchtowerLabelPrefix) && !(strings.HasSuffix(key, ".enable") && value == "false") {
			return UpdateSourceWatchtower, ""
		}
	}
	return UpdateSourceExternal, ""
}

// restoringLocked reports whether a start is a rolled back swap putting the old
// container back: the replacement was the last update under this name, and
// what starts now is the container it replaced, or for a --rm one, a
// recreation of it on its old image. The swap reports that itself, with
// RolledBack set, so the start is not a second update.
func (t *updateTracker) restoringLocked(c Container, prev imageRecord) bool {
	for _, e := range slices.Backward(t.events) {
		if e.EngineName != c.EngineName || e.RolledBack {
			continue
		}
		return e.NewID == prev.containerID && e.FromImageID == imageIdentity(c) &&
			(e.OldID == c.ID || c.Labels[restoredRefLabel] != "")
	}
	return false
}

// rolledBack records a swap that put the old container back.
func (t *updateTracker) rolledBack(event ContainerUpdateEvent) {
	if t == nil {
		return
	}
	event.RolledBack = true
	if event.At.IsZero() {
		event.At = t.now()
	}
	t.mu.Lock()
	t.appendLocked(event)
	t.mu.Unlock()
	t.broadcast(event)
}

// gone notes that a container was removed. Its record stays a while, for the
// container that replaces it.
func (t *updateTracker) gone(id string) {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	for name, r := range t.byName {
		if r.containerID == id && r.goneAt.IsZero() {
			r.goneAt = t.now()
			t.byName[name] = r
		}
	}
}

func (t *updateTracker) pruneLocked() {
	if len(t.byName) <= pruneRecordsAbove {
		return
	}
	cutoff := t.now().Add(-goneRecordTTL)
	for name, r := range t.byName {
		if !r.goneAt.IsZero() && r.goneAt.Before(cutoff) {
			delete(t.byName, name)
		}
	}
}

func (t *updateTracker) appendLocked(event ContainerUpdateEvent) {
	log.Info().
		Str("container", event.Name).
		Str("from", event.FromImageID).
		Str("to", event.ToImageID).
		Str("source", event.Source).
		Bool("rolledBack", event.RolledBack).
		Msg("container image changed")
	t.events = append(t.events, event)
	if over := len(t.events) - maxUpdateEvents; over > 0 {
		t.events = slices.Delete(t.events, 0, over)
	}
}

// recent is a copy of the kept events, oldest first.
func (t *updateTracker) recent() []ContainerUpdateEvent {
	if t == nil {
		return nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return slices.Clone(t.events)
}

func (t *updateTracker) subscribe(ctx context.Context, ch chan<- ContainerUpdateEvent) {
	if t == nil {
		return
	}
	t.subscribers.Store(ctx, ch)
	go func() {
		<-ctx.Done()
		t.subscribers.Delete(ctx)
	}()
}

// broadcast never blocks the event loop: a subscriber that is not keeping up
// misses the event, which it can still read from recent.
func (t *updateTracker) broadcast(event ContainerUpdateEvent) {
	t.subscribers.Range(func(ctx context.Context, ch chan<- ContainerUpdateEvent) bool {
		select {
		case ch <- event:
		case <-ctx.Done():
		default:
			log.Warn().Str("container", event.Name).Msg("update event subscriber is not keeping up, dropping event")
		}
		return true
	})
}

// RecentUpdates is the host's last update events, oldest first.
func (s *Store) RecentUpdates() []ContainerUpdateEvent {
	return s.updates.recent()
}

// SubscribeUpdates sends every update event on this host to ch until ctx ends.
// A send that would block is dropped, so ch should be buffered.
func (s *Store) SubscribeUpdates(ctx context.Context, ch chan<- ContainerUpdateEvent) {
	s.updates.subscribe(ctx, ch)
}

// RecordRolledBack records a swap that put the old container back. The swap is
// the only one that knows it tried: when the replacement never started there
// is no start to detect, and when it did, the old container coming back is
// not reported as an update of its own.
func (s *Store) RecordRolledBack(event ContainerUpdateEvent) {
	s.updates.rolledBack(event)
}

// UpdateHistory is a client service whose host keeps update events. The Docker
// service keeps its own; an agent's service reads the agent's over gRPC, and
// is empty for an agent too old to keep them.
type UpdateHistory interface {
	RecentUpdates() []ContainerUpdateEvent
	SubscribeUpdates(ctx context.Context, ch chan<- ContainerUpdateEvent)
}

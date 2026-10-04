package container

import (
	"context"
	"slices"
	"sync"
	"time"
)

// Where an update came from. Only updates Dozzle itself makes are recorded.
const (
	// UpdateSourceSchedule is the auto-update schedule.
	UpdateSourceSchedule = "schedule"
	// UpdateSourceDozzle is someone in the Dozzle UI, one container or a bulk run.
	UpdateSourceDozzle = "dozzle"
	// UpdateSourceCloud is Dozzle Cloud's update_container tool.
	UpdateSourceCloud = "cloud"
	// UpdateSourceRollback is a rollback to the image the container ran
	// before its last update. Only Dozzle Cloud starts one.
	UpdateSourceRollback = "rollback"
)

// UpdateRecord is one update or rollback Dozzle made: the image under a name
// changed. It is recorded once the update finishes, by the process that asked
// for it, so an update an agent ran is recorded by the server like its own.
// The log view marks it, and Dozzle Cloud hears about it.
type UpdateRecord struct {
	Host string `json:"host"`
	Name string `json:"name"`
	// OldID is the container that ran before, NewID the one the update left
	// running: the replacement, or the old one put back when RolledBack.
	OldID string `json:"oldId"`
	NewID string `json:"newId"`
	// ImageRef is the image reference (repo:tag). An update keeps it and
	// moves what it resolves to, so it is the same before and after.
	ImageRef    string `json:"imageRef,omitempty"`
	FromDigest  string `json:"fromDigest,omitempty"`
	ToDigest    string `json:"toDigest,omitempty"`
	FromImageID string `json:"fromImageId,omitempty"`
	ToImageID   string `json:"toImageId,omitempty"`
	// OldStartedAt is when the old container last started.
	OldStartedAt time.Time `json:"-"`
	// At is when the update finished.
	At time.Time `json:"at"`
	// Source is one of the UpdateSource* values.
	Source string `json:"source"`
	// RolledBack means the new container did not stay up and Dozzle put the
	// old one back.
	RolledBack bool `json:"rolledBack,omitempty"`
}

// updateRecordsPerHost bounds what is kept. It only has to cover the log view
// of a container updated recently and a Dozzle Cloud link that was down for a
// while; a restart forgets everything anyway.
const updateRecordsPerHost = 50

// UpdateRecords keeps the last updates of every host in memory and tells
// subscribers about each new one.
type UpdateRecords struct {
	mu     sync.Mutex
	byHost map[string][]UpdateRecord
	subs   map[chan<- UpdateRecord]struct{}
}

func NewUpdateRecords() *UpdateRecords {
	return &UpdateRecords{byHost: make(map[string][]UpdateRecord), subs: make(map[chan<- UpdateRecord]struct{})}
}

// Updates is this process's record of the updates it made.
var Updates = NewUpdateRecords()

// Add keeps r and sends it to every subscriber. A send that would block is
// dropped, so a slow subscriber never holds up an update.
func (u *UpdateRecords) Add(r UpdateRecord) {
	u.mu.Lock()
	defer u.mu.Unlock()
	kept := append(u.byHost[r.Host], r)
	if len(kept) > updateRecordsPerHost {
		kept = slices.Clone(kept[len(kept)-updateRecordsPerHost:])
	}
	u.byHost[r.Host] = kept
	for ch := range u.subs {
		select {
		case ch <- r:
		default:
		}
	}
}

// Recent is every kept record, oldest first.
func (u *UpdateRecords) Recent() []UpdateRecord {
	u.mu.Lock()
	defer u.mu.Unlock()
	var all []UpdateRecord
	for _, records := range u.byHost {
		all = append(all, records...)
	}
	slices.SortStableFunc(all, func(a, b UpdateRecord) int { return a.At.Compare(b.At) })
	return all
}

// Latest is the newest record that left container id running on host.
func (u *UpdateRecords) Latest(host, id string) (UpdateRecord, bool) {
	if len(id) > 12 {
		id = id[:12]
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	records := u.byHost[host]
	for _, record := range slices.Backward(records) {
		if record.NewID == id {
			return record, true
		}
	}
	return UpdateRecord{}, false
}

// Subscribe sends every new record to ch until ctx ends. Sends never block,
// so ch should be buffered.
func (u *UpdateRecords) Subscribe(ctx context.Context, ch chan<- UpdateRecord) {
	u.mu.Lock()
	u.subs[ch] = struct{}{}
	u.mu.Unlock()
	context.AfterFunc(ctx, func() {
		u.mu.Lock()
		delete(u.subs, ch)
		u.mu.Unlock()
	})
}

// newUpdateRecord is the record of an update or rollback of c that ran and
// reported result.
func newUpdateRecord(c Container, source string, result UpdateResult, at time.Time) UpdateRecord {
	return UpdateRecord{
		Host:         c.Host,
		Name:         c.Name,
		OldID:        result.OldID,
		NewID:        result.NewID,
		ImageRef:     c.Image,
		FromDigest:   result.FromDigest,
		ToDigest:     result.ToDigest,
		FromImageID:  result.FromImageID,
		ToImageID:    result.ToImageID,
		OldStartedAt: result.OldStartedAt,
		At:           at.UTC(),
		Source:       source,
		RolledBack:   result.RolledBack,
	}
}

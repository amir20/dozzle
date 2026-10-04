package cloud

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/amir20/dozzle/internal/container"
	pb "github.com/amir20/dozzle/proto/cloud"
	"github.com/rs/zerolog/log"
)

// Dozzle Cloud judges container updates: it watches what the new image does
// and offers to roll back one that broke something. For that it hears about
// the updates the user agreed to share, and nothing else:
//
//   - an update the auto-update schedule made. Putting a container on the
//     schedule (its dev.dozzle.update label, or the instance updating all
//     containers) is the user's opt-in, to the update and to Cloud watching
//     it. One the swap undid is sent too: it is how that update ended.
//   - a rollback. Only Dozzle Cloud starts one, so it is the end of something
//     Cloud already knows about.
//
// Updates made from the Dozzle UI or by Cloud's update_container tool are not
// sent. Only the update record goes, never log content.
const consentSchedule = "schedule"

const (
	// updateChanBuf absorbs a scheduled night finishing several containers
	// while the pusher is busy sending one.
	updateChanBuf = 64
	// updateLedgerMax bounds the ledger. Past it the ledger starts over, which
	// at worst sends an update twice: Cloud keys updates on (host, name,
	// new_container_id), so a repeat is recorded once.
	updateLedgerMax = 1024
)

// updateLedger remembers what was pushed across reconnects, so the replay of
// recent updates on each connect sends only what Cloud has not had.
type updateLedger struct {
	mu   sync.Mutex
	sent map[string]struct{}
}

func newUpdateLedger() *updateLedger {
	return &updateLedger{sent: make(map[string]struct{})}
}

func updateKey(r container.UpdateRecord) string {
	return r.Host + "|" + r.Name + "|" + r.NewID + "|" + strconv.FormatBool(r.RolledBack) + "|" + strconv.FormatInt(r.At.UnixNano(), 10)
}

func (l *updateLedger) hasSent(r container.UpdateRecord) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	_, ok := l.sent[updateKey(r)]
	return ok
}

func (l *updateLedger) record(r container.UpdateRecord) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.sent) >= updateLedgerMax {
		l.sent = make(map[string]struct{})
	}
	l.sent[updateKey(r)] = struct{}{}
}

// shared reports whether r may be sent to Dozzle Cloud.
func shared(r container.UpdateRecord) bool {
	return r.Source == container.UpdateSourceSchedule || r.Source == container.UpdateSourceRollback
}

// pushUpdates sends every shared update recorded before this connection that
// Cloud has not had yet, then each new one as it is recorded, until ctx ends.
func pushUpdates(ctx context.Context, records *container.UpdateRecords, ledger *updateLedger, send func(*pb.ToolResponse) error) {
	// Subscribed before the replay, so an update recorded while it runs is
	// not missed. One that arrives both ways is sent once, by the ledger.
	updates := make(chan container.UpdateRecord, updateChanBuf)
	records.Subscribe(ctx, updates)

	// Updates kept from before this connection: made while the link was down,
	// or sent on a connection that broke before Cloud had them.
	for _, r := range records.Recent() {
		if ctx.Err() != nil {
			return
		}
		pushUpdate(r, ledger, send)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case r := <-updates:
			pushUpdate(r, ledger, send)
		}
	}
}

func pushUpdate(r container.UpdateRecord, ledger *updateLedger, send func(*pb.ToolResponse) error) {
	if !shared(r) || ledger.hasSent(r) {
		return
	}
	resp := &pb.ToolResponse{Type: &pb.ToolResponse_ContainerUpdate{ContainerUpdate: containerUpdateProto(r)}}
	if err := send(resp); err != nil {
		// The connection is going; the next one replays it.
		log.Debug().Err(err).Str("container", r.Name).Msg("update pusher: send failed")
		return
	}
	ledger.record(r)
	log.Debug().Str("container", r.Name).Str("source", r.Source).Bool("rolledBack", r.RolledBack).Msg("update pusher: sent update")
}

func containerUpdateProto(r container.UpdateRecord) *pb.ContainerUpdate {
	return &pb.ContainerUpdate{
		Host:           r.Host,
		Name:           r.Name,
		OldContainerId: r.OldID,
		NewContainerId: r.NewID,
		FromRef:        r.FromRef,
		ToRef:          r.ToRef,
		FromDigest:     digestOnly(r.FromDigest),
		ToDigest:       digestOnly(r.ToDigest),
		FromImageId:    r.FromImageID,
		ToImageId:      r.ToImageID,
		OldStartedAt:   unixNano(r.OldStartedAt),
		At:             unixNano(r.At),
		Source:         r.Source,
		// The schedule is the only consent this build sends updates under. A
		// rollback carries it too: it ends an update Cloud already judged.
		Consent:    consentSchedule,
		RolledBack: r.RolledBack,
	}
}

// digestOnly is the sha256:... part of a repo@sha256:... reference, which is
// what cloud.proto carries. The repository is in the ref fields.
func digestOnly(ref string) string {
	if ref == "" {
		return ""
	}
	return container.DigestOf(ref)
}

func unixNano(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.UnixNano()
}

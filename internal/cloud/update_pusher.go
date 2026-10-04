package cloud

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/notification"
	pb "github.com/amir20/dozzle/proto/cloud"
	"github.com/rs/zerolog/log"
)

// Dozzle Cloud judges container updates: it watches what the new image does
// and asks before rolling anything back. For that it needs to hear about each
// update, but only the ones the user agreed to share. An update is pushed when
// one of these holds, and the reason travels with it as ContainerUpdate.consent:
//
//   - schedule: the auto-update schedule made it. The dev.dozzle.auto-update
//     label is the user's opt-in, to the update and to Cloud watching it.
//   - checkbox: the user started it from the Dozzle UI with "Have Dozzle Cloud
//     watch this update" ticked.
//   - rule: an enabled lifecycle (event) rule that notifies Dozzle Cloud
//     matches the container. That rule already sends Cloud this container's
//     starts and stops; the update is the same kind of fact.
//
// Nothing here reaches webhook, Slack or ntfy rules. Only the update record is
// sent, never log content.
const (
	consentSchedule = "schedule"
	consentCheckbox = "checkbox"
	consentRule     = "rule"
)

// UpdateStreamHostService is the subset of the host service the update pusher
// needs. It is type-asserted at connect time; a host service without update
// history (k8s) simply pushes nothing.
type UpdateStreamHostService interface {
	ToolHostService
	// RecentUpdates is every host's last update events, oldest first.
	RecentUpdates() []container.ContainerUpdateEvent
	// SubscribeUpdates sends every host's update events to ch until ctx ends.
	// A send that would block is dropped, so ch should be buffered.
	SubscribeUpdates(ctx context.Context, ch chan<- container.ContainerUpdateEvent)
}

const (
	// updateChanBuf absorbs a scheduled night finishing several containers
	// while the pusher is busy resolving one.
	updateChanBuf = 64
	// updateLedgerMax bounds the ledger. Past it the ledger starts over, which
	// at worst sends an update twice: Cloud keys updates on (host, name,
	// to_digest), so a repeat is recorded once.
	updateLedgerMax = 1024
)

// updateLedger remembers what was pushed across reconnects, so the replay of
// recent updates on each connect sends only what the cloud has not had.
type updateLedger struct {
	mu   sync.Mutex
	sent map[string]struct{}
	// consent is the consent of the last update pushed per host/name. A
	// rollback of that update is pushed under the same consent: it is the
	// end of the update the user already agreed to share.
	consent map[string]string
}

func newUpdateLedger() *updateLedger {
	return &updateLedger{sent: make(map[string]struct{}), consent: make(map[string]string)}
}

func updateKey(e container.ContainerUpdateEvent) string {
	return e.Host + "|" + e.Name + "|" + e.NewID + "|" + e.ToImageID + "|" + strconv.FormatBool(e.RolledBack) + "|" + strconv.FormatInt(e.At.UnixNano(), 10)
}

func (l *updateLedger) hasSent(e container.ContainerUpdateEvent) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	_, ok := l.sent[updateKey(e)]
	return ok
}

func (l *updateLedger) record(e container.ContainerUpdateEvent, consent string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.sent) >= updateLedgerMax {
		l.sent = make(map[string]struct{})
	}
	if len(l.consent) >= updateLedgerMax {
		l.consent = make(map[string]string)
	}
	l.sent[updateKey(e)] = struct{}{}
	l.consent[e.Host+"/"+e.Name] = consent
}

func (l *updateLedger) lastConsent(host, name string) string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.consent[host+"/"+name]
}

// updatePusher sends one ImageSnapshot when the connection opens, then one
// ContainerUpdate for every update the user agreed to share. It lives for one
// connection; the ledger outlives it.
type updatePusher struct {
	hosts   UpdateStreamHostService
	labels  container.ContainerLabels
	rules   NotificationService
	watched func(container.ContainerUpdateEvent) bool
	ledger  *updateLedger
	send    func(resp *pb.ToolResponse) error
}

func newUpdatePusher(hosts UpdateStreamHostService, deps ToolDeps, ledger *updateLedger, send func(resp *pb.ToolResponse) error) *updatePusher {
	return &updatePusher{
		hosts:   hosts,
		labels:  deps.Principal.Labels,
		rules:   deps.NotificationService,
		watched: deps.UpdateWatched,
		ledger:  ledger,
		send:    send,
	}
}

// run blocks until ctx ends.
func (p *updatePusher) run(ctx context.Context) {
	// Subscribed before the snapshot and the replay, so an update that lands
	// while they are sent is not missed. One that arrives both ways is sent
	// once, by the ledger.
	updates := make(chan container.ContainerUpdateEvent, updateChanBuf)
	p.hosts.SubscribeUpdates(ctx, updates)

	if err := p.sendSnapshot(); err != nil {
		log.Debug().Err(err).Msg("update pusher: snapshot not sent")
	}
	// Updates kept from before this connection: made while the link was down,
	// or sent on a connection that broke before Cloud had them.
	for _, e := range p.hosts.RecentUpdates() {
		if ctx.Err() != nil {
			return
		}
		p.push(e)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case e := <-updates:
			p.push(e)
		}
	}
}

// push sends e if the user agreed to share it and it was not sent before.
func (p *updatePusher) push(e container.ContainerUpdateEvent) {
	if p.ledger.hasSent(e) {
		return
	}
	c, ok := p.resolve(e)
	if !ok {
		// Gone already, or hidden by the instance's label filter, which keeps
		// it from Cloud like every other tool does.
		log.Debug().Str("container", e.Name).Msg("update pusher: container not visible, not sending update")
		return
	}
	consent, ok := p.consent(e, c, p.hostNames())
	if !ok {
		return
	}
	resp := &pb.ToolResponse{Type: &pb.ToolResponse_ContainerUpdate{ContainerUpdate: containerUpdateProto(e, consent)}}
	if err := p.send(resp); err != nil {
		// The connection is going; the next one replays it from RecentUpdates.
		log.Debug().Err(err).Str("container", e.Name).Msg("update pusher: send failed")
		return
	}
	p.ledger.record(e, consent)
	log.Debug().Str("container", e.Name).Str("consent", consent).Bool("rolledBack", e.RolledBack).Msg("update pusher: sent update")
}

// resolve finds the container that holds the name now, under the instance's
// label filter.
func (p *updatePusher) resolve(e container.ContainerUpdateEvent) (container.Container, bool) {
	if e.NewID != "" {
		if service, err := p.hosts.FindContainer(e.Host, e.NewID, p.labels); err == nil && service != nil {
			return service.Container, true
		}
	}
	containers, _ := p.hosts.ListAllContainers(p.labels)
	for _, c := range containers {
		if c.Host == e.Host && c.Name == e.Name && c.State != "deleted" {
			return c, true
		}
	}
	return container.Container{}, false
}

// consent is why e may be sent, or false when it may not.
func (p *updatePusher) consent(e container.ContainerUpdateEvent, c container.Container, hostNames map[string]string) (string, bool) {
	if e.Source == container.UpdateSourceSchedule {
		return consentSchedule, true
	}
	if p.watched != nil && p.watched(e) {
		return consentCheckbox, true
	}
	if e.Source == container.UpdateSourceRollback {
		if consent := p.ledger.lastConsent(e.Host, e.Name); consent != "" {
			return consent, true
		}
	}
	if p.ruleMatches(c, hostNames) {
		return consentRule, true
	}
	return "", false
}

// ruleMatches reports whether an enabled lifecycle rule that notifies Dozzle
// Cloud covers c.
func (p *updatePusher) ruleMatches(c container.Container, hostNames map[string]string) bool {
	if p.rules == nil {
		return false
	}
	nc := notification.FromContainerModel(c, container.Host{ID: c.Host, Name: hostNames[c.Host]})
	for _, sub := range p.rules.Subscriptions() {
		if sub.Enabled && sub.DispatcherID == cloudDispatcherID && sub.IsEventAlert() && sub.MatchesContainer(nc) {
			return true
		}
	}
	return false
}

func (p *updatePusher) hostNames() map[string]string {
	names := make(map[string]string)
	for _, h := range p.hosts.Hosts() {
		names[h.ID] = h.Name
	}
	return names
}

// sendSnapshot sends what every covered container runs now: one labelled for
// the auto-update schedule, or matched by a lifecycle rule that notifies
// Cloud. Cloud diffs it against what it recorded, which is how it learns of an
// update Dozzle made while the link was down. Containers the user never agreed
// to share are left out, and nothing is sent when that leaves none.
func (p *updatePusher) sendSnapshot() error {
	containers, errs := p.hosts.ListAllContainers(p.labels)
	for _, err := range errs {
		log.Debug().Err(err).Msg("update pusher: host unavailable for the image snapshot")
	}
	hostNames := p.hostNames()

	snapshot := &pb.ImageSnapshot{}
	for _, c := range containers {
		// A container that was only created never ran, and the update tracker
		// ignores it for the same reason.
		if c.ImageID == "" || c.Name == "" || c.State == "created" || c.State == "deleted" {
			continue
		}
		if !container.AutoUpdateEnabled(c.Labels) && !p.ruleMatches(c, hostNames) {
			continue
		}
		// A list entry may not know the digest or start time yet; an inspect does.
		if c.ImageDigest == "" || c.StartedAt.IsZero() {
			if service, err := p.hosts.FindContainer(c.Host, c.ID, p.labels); err == nil && service != nil {
				c = service.Container
			}
		}
		snapshot.Entries = append(snapshot.Entries, &pb.ImageSnapshotEntry{
			Host:        c.Host,
			Name:        c.Name,
			ContainerId: c.ID,
			ImageId:     c.ImageID,
			Digest:      digestOnly(c.ImageDigest),
			Ref:         c.Image,
			StartedAt:   unixNano(c.StartedAt),
		})
	}
	if len(snapshot.Entries) == 0 {
		return nil
	}
	return p.send(&pb.ToolResponse{Type: &pb.ToolResponse_ImageSnapshot{ImageSnapshot: snapshot}})
}

func containerUpdateProto(e container.ContainerUpdateEvent, consent string) *pb.ContainerUpdate {
	return &pb.ContainerUpdate{
		Host:           e.Host,
		Name:           e.Name,
		OldContainerId: e.OldID,
		NewContainerId: e.NewID,
		FromRef:        e.FromRef,
		ToRef:          e.ToRef,
		FromDigest:     digestOnly(e.FromDigest),
		ToDigest:       digestOnly(e.ToDigest),
		FromImageId:    e.FromImageID,
		ToImageId:      e.ToImageID,
		OldStartedAt:   unixNano(e.OldStartedAt),
		At:             unixNano(e.At),
		Source:         e.Source,
		RunId:          e.RunID,
		Consent:        consent,
		RolledBack:     e.RolledBack,
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

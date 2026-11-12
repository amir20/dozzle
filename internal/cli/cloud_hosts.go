package cli

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/amir20/dozzle/internal/cloud"
	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/hostservice"
	"github.com/amir20/dozzle/internal/web"
	"github.com/rs/zerolog/log"
)

// cloudHostService is the view of the fleet the cloud client is given.
//
// In swarm mode every replica discovers every peer, so a replica reporting the
// whole fleet would multiply hosts — and duplicate log and stat ingestion — by
// replica count on the cloud side. There the view is scoped to this process's
// own docker daemon.
//
// Everywhere else there is exactly one Dozzle holding the fleet: --remote-host
// and --remote-agent endpoints are configured on it and on nothing else, so
// scoping them away simply hid them from the cloud. --remote-host survived only
// because it happens to be a *docker.Service; agents did not appear at all.
//
// services is a func, not a slice, because an agent that is unreachable at boot
// joins later. Reading it per call means such an agent shows up as soon as it
// connects rather than at the next restart. Its retry argument asks for
// unreachable agents to be re-dialed, which costs a connection attempt each —
// only the periodic fan-out calls pay it.
type cloudHostService struct {
	services func(retry bool) []container.ClientService
	// hs is the underlying host service, used only to learn when a previously
	// unreachable host becomes available so log and stat subscriptions can be
	// extended to it.
	hs web.HostService

	mu      sync.Mutex
	hostIDs map[container.ClientService]string
}

func newCloudHostService(mode string, hs web.HostService) cloud.LogStreamHostService {
	services := func(bool) []container.ClientService { return hs.LocalClientServices() }
	if mode != "swarm" {
		if mhs, ok := hs.(*hostservice.MultiHostService); ok {
			services = mhs.ClientServices
		}
	}
	if len(services(false)) == 0 {
		// k8s has no docker client services but its HostService already
		// exposes only what this process can see, so use it directly.
		return hs
	}
	return &cloudHostService{
		services: services,
		hs:       hs,
		hostIDs:  make(map[container.ClientService]string),
	}
}

// newConfigHostService is the hosts whose container configs this Dozzle sends
// to the cloud: its own docker daemons (local and --remote-host), never its
// --remote-agent agents, which each send their own. Two senders would hash the
// same labels with two different keys, and every start would read as a
// change. Nil on k8s, where a pod's name changes with every rollout, so "the
// same container" needs another key.
func newConfigHostService(mode string, hs web.HostService) cloud.ConfigStreamHostService {
	if mode == "k8s" || len(hs.LocalClientServices()) == 0 {
		return nil
	}
	return &cloudHostService{
		services: func(bool) []container.ClientService { return hs.LocalClientServices() },
		hs:       hs,
		hostIDs:  make(map[container.ClientService]string),
	}
}

// reattachInterval is how often log and stat subscriptions re-scan the fleet.
// A var, not a const, so tests need not wait a real minute.
var reattachInterval = time.Minute

func (l *cloudHostService) hostTimeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 5*time.Second)
}

// hostID caches a service's host ID. A host ID is stable for the lifetime of
// the client that serves it, so one lookup per service is enough.
//
// An agent answers Host over the wire and can fail transiently, but it returns
// its last known host alongside the error, and a service only enters the fleet
// after one successful Host call. So an id is taken whenever there is one,
// error or not; only a service that has never identified itself yields "".
//
// The lock is dropped across the lookup on purpose. Two callers racing a cold
// cache both dial and both store, which costs one redundant call and writes the
// same id twice; holding the lock instead would serialise every caller behind a
// network round trip, including callers asking about other hosts.
func (l *cloudHostService) hostID(s container.ClientService) string {
	l.mu.Lock()
	id, ok := l.hostIDs[s]
	l.mu.Unlock()
	if ok {
		return id
	}

	ctx, cancel := l.hostTimeout()
	h, _ := s.Host(ctx)
	cancel()
	if h.ID == "" {
		return ""
	}

	l.mu.Lock()
	l.hostIDs[s] = h.ID
	l.mu.Unlock()
	return h.ID
}

func (l *cloudHostService) Hosts() []container.Host {
	services := l.services(true)
	hosts := make([]container.Host, 0, len(services))
	for _, s := range services {
		ctx, cancel := l.hostTimeout()
		h, err := s.Host(ctx)
		cancel()
		if err != nil {
			continue
		}
		h.Available = true
		hosts = append(hosts, h)
	}
	return hosts
}

func (l *cloudHostService) ListAllContainers(labels container.ContainerLabels) ([]container.Container, []error) {
	var all []container.Container
	var errs []error
	for _, s := range l.services(true) {
		ctx, cancel := l.hostTimeout()
		list, err := s.ListContainers(ctx, labels)
		cancel()
		if err != nil {
			errs = append(errs, err)
			continue
		}
		all = append(all, list...)
	}
	return all, errs
}

func (l *cloudHostService) FindContainer(host string, id string, labels container.ContainerLabels) (*container.ContainerService, error) {
	// No retry: this runs once per log reader, and a run of them against an
	// unreachable agent would each wait out the dial timeout.
	for _, s := range l.services(false) {
		if l.hostID(s) != host {
			continue
		}
		ctx, cancel := l.hostTimeout()
		cont, err := s.FindContainer(ctx, id, labels)
		cancel()
		if err != nil {
			return nil, err
		}
		return container.NewContainerService(s, cont), nil
	}
	return nil, fmt.Errorf("host %s is not served by this Dozzle instance", host)
}

// watchNewServices calls attach again whenever a host that was unreachable at
// startup joins, so a late agent gets subscribed without waiting for a restart.
//
// The ticker is the backstop. SubscribeAvailableHosts only fires on the
// unreachable-to-reachable edge, so anything attach skipped for another reason
// — a host whose id could not be resolved at the time — would otherwise stay
// skipped for the life of the connection. A re-attach that finds nothing new is
// a map lookup per service, so this is cheap enough to run unconditionally.
//
// attach is only ever called from this one goroutine, after the caller's
// initial synchronous call has returned, so it needs no locking of its own.
func (l *cloudHostService) watchNewServices(ctx context.Context, attach func()) {
	hosts := make(chan container.Host, 8)
	l.hs.SubscribeAvailableHosts(ctx, hosts)
	ticker := time.NewTicker(reattachInterval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-hosts:
				attach()
			case <-ticker.C:
				attach()
			}
		}
	}()
}

// SubscribeStats fans stats in from every client service, stamping the
// originating host on each sample. container.ContainerStat has no host field,
// so without this the cloud side could not tell two same-named containers on
// different hosts apart.
func (l *cloudHostService) SubscribeStats(ctx context.Context, samples chan<- cloud.StatSample) {
	// One inbound channel + forwarder goroutine per service, matching
	// SubscribeContainersStarted: a burst on one service must not stall the others.
	var dropWarn sync.Once
	subs := serviceSubscriptions{}
	attach := func() {
		subs.sync(ctx, l.services(false), func(ctx context.Context, s container.ClientService) bool {
			hostID := l.hostID(s)
			if hostID == "" {
				// Unstamped samples can't be told apart on the cloud side.
				// Leave the service unsubscribed; watchNewServices re-attaches
				// on a timer, so this resolves itself once the host answers.
				return false
			}
			ch := make(chan container.ContainerStat, 64)
			s.SubscribeStats(ctx, ch)
			go func() {
				for {
					select {
					case <-ctx.Done():
						return
					case stat := <-ch:
						// Non-blocking on purpose. The stats collector dispatches to
						// each subscriber with a blocking send, so if cloud ingest
						// ever wedged, backpressure would travel all the way up and
						// starve the live UI's stats subscriber on this host. Losing
						// a sample only nudges a 30s average; stalling the UI is not
						// an acceptable trade for that.
						select {
						case samples <- cloud.StatSample{Stat: stat, HostID: hostID}:
						default:
							dropWarn.Do(func() {
								log.Warn().Msg("cloud stats: consumer is not keeping up, dropping samples (further drops are silent)")
							})
						}
					}
				}
			}()
			return true
		})
	}

	attach()
	l.watchNewServices(ctx, attach)
}

func (l *cloudHostService) SubscribeContainersStarted(ctx context.Context, containers chan<- container.Container, filter container.ContainerFilter) {
	// One inbound channel + forwarder goroutine per service so a slow consumer
	// or a burst on one service can't cause the others to drop events.
	subs := serviceSubscriptions{}
	attach := func() {
		subs.sync(ctx, l.services(false), func(ctx context.Context, s container.ClientService) bool {
			ch := make(chan container.Container, 64)
			s.SubscribeContainersStarted(ctx, ch)
			go func() {
				for {
					select {
					case <-ctx.Done():
						return
					case c := <-ch:
						if filter(&c) {
							select {
							case containers <- c:
							case <-ctx.Done():
								return
							}
						}
					}
				}
			}()
			return true
		})
	}

	attach()
	l.watchNewServices(ctx, attach)
}

// serviceSubscriptions is one cloud subscription per client service, each under
// its own context so it can end on its own.
type serviceSubscriptions map[container.ClientService]context.CancelFunc

// sync subscribes the services not yet subscribed, and ends the subscriptions of
// services no longer listed: an agent removed from the UI would otherwise keep
// its forwarder goroutine and channel until the cloud connection ends. start
// returns false to leave a service for the next sync.
func (subs serviceSubscriptions) sync(ctx context.Context, services []container.ClientService, start func(context.Context, container.ClientService) bool) {
	listed := make(map[container.ClientService]bool, len(services))
	for _, s := range services {
		listed[s] = true
		if _, ok := subs[s]; ok {
			continue
		}
		subCtx, cancel := context.WithCancel(ctx)
		if !start(subCtx, s) {
			cancel()
			continue
		}
		subs[s] = cancel
	}
	for s, cancel := range subs {
		if !listed[s] {
			cancel()
			delete(subs, s)
		}
	}
}

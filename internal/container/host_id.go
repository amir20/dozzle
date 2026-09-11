package container

import (
	"github.com/google/uuid"
	"github.com/samber/lo"
)

// EngineIdentity is everything a backend can say about the thing it just
// connected to. Every field is best effort and may be empty: a backend reports
// what it has and does not decide what any of it is worth.
type EngineIdentity struct {
	// Runtime is "docker", "podman" or "k8s".
	Runtime string

	// EngineID is the identity the engine claims for itself. Docker writes one
	// to /var/lib/docker/engine-id at first start and a Kubernetes node reports
	// its machine id, both of which outlive a restart. Podman fills this field
	// with a value that does not, which is the whole reason this package has a
	// say in the matter.
	EngineID string

	// SwarmNodeID is set only on a swarm member. Every other node of the swarm
	// already refers to this one by it.
	SwarmNodeID string

	// Hostname and StorageRoot are the two things Podman reports consistently,
	// and all there is to derive an identity from when EngineID is worthless.
	Hostname    string
	StorageRoot string

	// Fallback is whatever the caller already knew, used when the engine could
	// not be reached and reported nothing at all.
	Fallback string
}

// HostIDResolver decides what id identifies a host.
//
// A host id has to be stable for the life of the machine and unique across the
// fleet, because it is what the hub routes on and what every bookmarked URL
// carries. Backends differ in how much of that they can supply, so the choice
// lives here rather than in each client: a client reports an EngineIdentity and
// is told what it is talking to.
//
// Nothing enforces stability, so treat it as a preference rather than a
// guarantee. RetriableClientManager re-keys a client whose id moves anyway.
type HostIDResolver interface {
	Resolve(EngineIdentity) string
}

// NewHostIDResolver builds the resolver for one host, from configuration read
// once at startup. An empty override means Dozzle works the id out itself.
//
// The result identifies a single host, so it is not shared with clients for
// other engines. Handing one StaticHostID to several remote hosts would give
// them all the same id and collapse them into one.
func NewHostIDResolver(override string) HostIDResolver {
	if override == "" {
		return DerivedHostID{}
	}

	return StaticHostID(override)
}

// StaticHostID is an id the operator set by hand. It is the only remedy when a
// derived id collides, which takes two hosts sharing both a hostname and a
// storage root.
type StaticHostID string

func (s StaticHostID) Resolve(EngineIdentity) string {
	return string(s)
}

// DerivedHostID works the id out from what the engine reported, preferring the
// most durable thing on offer.
type DerivedHostID struct{}

func (DerivedHostID) Resolve(engine EngineIdentity) string {
	return lo.CoalesceOrEmpty(engine.SwarmNodeID, podmanHostID(engine), engine.EngineID, engine.Fallback)
}

// dozzleHostNamespace seeds the derived ids below. It is a random constant
// picked once, not one of the RFC 4122 namespaces, and it exists only so the
// derivation is reproducible across restarts and across Dozzle versions.
// Changing it would re-id every Podman host in the world, so don't.
var dozzleHostNamespace = uuid.MustParse("cb6c32a9-acb9-454b-8427-014fe9bc073c")

// podmanHostID derives an id Podman itself cannot supply, and returns "" for
// anything it has no answer for: every other runtime, and a Podman that
// reported nothing to derive from.
//
// Podman is daemonless and tracks no engine identity at all, so its Docker
// compat /info fills the required ID field with a throwaway uuid.New() on every
// single call. Nothing reads /var/lib/docker/engine-id on the way, which is why
// creating that file has never done anything on Podman. A client reads that
// value once when it connects, so the id it gets survives only as long as the
// process that first asked for it.
//
// The storage root is what keeps two rootless users on one machine apart: they
// share a hostname but never a store.
//
// Declining when Podman reported neither matters. Hashing two empty strings
// would hand every host in the fleet one id, and the duplicate-host check in
// RetriableClientManager would collapse them into a single host. A churning id
// is bad, hosts silently vanishing is worse.
func podmanHostID(engine EngineIdentity) string {
	if engine.Runtime != "podman" {
		return ""
	}

	if engine.Hostname == "" && engine.StorageRoot == "" {
		return ""
	}

	return uuid.NewSHA1(dozzleHostNamespace, []byte(engine.Hostname+"\x00"+engine.StorageRoot)).String()
}

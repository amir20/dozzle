package imagecheck

import (
	"context"
	"errors"
	"slices"
	"sync"
	"time"
)

// Mode controls whether Dozzle contacts registries at all.
type Mode string

const (
	// ModeAutomatic checks in the background when a container is viewed.
	ModeAutomatic Mode = "automatic"
	// ModeManual only checks when a user explicitly asks.
	ModeManual Mode = "manual"
	// ModeOff disables the feature entirely; no route, no egress.
	ModeOff Mode = "off"
)

func ParseMode(input string) (Mode, error) {
	switch mode := Mode(input); mode {
	case ModeAutomatic, ModeManual, ModeOff:
		return mode, nil
	default:
		return "", errors.New("invalid image check mode: " + input + " (expected automatic, manual or off)")
	}
}

// Status is the outcome of an update check for a single container.
type Status string

const (
	StatusUpToDate        Status = "up-to-date"
	StatusUpdateAvailable Status = "update-available"
	// StatusPinned means the container runs a digest-pinned image, which by
	// definition cannot drift.
	StatusPinned Status = "pinned"
	// StatusNotCheckable means the image has no registry digest locally,
	// which is the case for locally built images.
	StatusNotCheckable Status = "not-checkable"
	// StatusAuthRequired means the registry refused an anonymous request.
	StatusAuthRequired Status = "auth-required"
	// StatusSkipped means the container opted out via label.
	StatusSkipped Status = "skipped"
	StatusUnknown Status = "unknown"
)

// Result is returned to the frontend for a single container.
type Result struct {
	Status       Status    `json:"status"`
	Image        string    `json:"image"`
	LocalDigest  string    `json:"localDigest,omitempty"`
	RemoteDigest string    `json:"remoteDigest,omitempty"`
	CheckedAt    time.Time `json:"checkedAt"`
	Reason       string    `json:"reason,omitempty"`
}

// UpdateAvailable is a convenience for callers that only care about the alert.
func (r Result) UpdateAvailable() bool {
	return r.Status == StatusUpdateAvailable
}

// SkipLabel lets an operator silence checks for a container they pin
// deliberately, following the existing dev.dozzle.* label convention.
const SkipLabel = "dev.dozzle.update-check"

// Skipped reports whether container labels opt out of update checks.
func Skipped(labels map[string]string) bool {
	switch labels[SkipLabel] {
	case "false", "off", "no":
		return true
	default:
		return false
	}
}

type digestEntry struct {
	digest    string
	err       error
	fetchedAt time.Time
}

// Checker resolves remote digests with a TTL cache in front of the registry.
// The cache holds only the remote digest, never the verdict, so a container
// that is updated locally reports up-to-date immediately without waiting for
// the entry to expire.
type Checker struct {
	registry *Registry
	ttl      time.Duration

	mu    sync.Mutex
	cache map[string]digestEntry
}

func NewChecker(registry *Registry, ttl time.Duration) *Checker {
	return &Checker{
		registry: registry,
		ttl:      ttl,
		cache:    make(map[string]digestEntry),
	}
}

// Check compares the registry's current digest for image against the digests
// Docker records locally. localDigests is the image's RepoDigests, which may
// hold more than one entry for a tag, so membership rather than equality
// decides whether the container is current.
func (c *Checker) Check(ctx context.Context, image string, localDigests []string, force bool) Result {
	result := Result{Image: image, CheckedAt: time.Now()}

	ref, err := ParseReference(image)
	if err != nil {
		result.Status = StatusUnknown
		result.Reason = err.Error()
		return result
	}

	if ref.Pinned() {
		result.Status = StatusPinned
		result.LocalDigest = ref.Digest
		return result
	}

	if len(localDigests) == 0 {
		result.Status = StatusNotCheckable
		result.Reason = "image has no registry digest locally"
		return result
	}

	remote, err := c.remoteDigest(ctx, image, ref, force)
	if err != nil {
		switch {
		case errors.Is(err, ErrAuthRequired):
			result.Status = StatusAuthRequired
			result.Reason = "registry requires credentials"
		default:
			result.Status = StatusUnknown
			result.Reason = err.Error()
		}
		return result
	}

	result.RemoteDigest = remote
	result.LocalDigest = localDigests[0]

	if slices.Contains(localDigests, remote) {
		result.Status = StatusUpToDate
	} else {
		result.Status = StatusUpdateAvailable
	}

	return result
}

func (c *Checker) remoteDigest(ctx context.Context, image string, ref Reference, force bool) (string, error) {
	if !force {
		c.mu.Lock()
		entry, ok := c.cache[image]
		c.mu.Unlock()
		if ok && time.Since(entry.fetchedAt) < c.ttl {
			return entry.digest, entry.err
		}
	}

	digest, err := c.registry.Digest(ctx, ref)

	// Cache failures too, so an unreachable or private registry is not retried
	// on every page view.
	c.mu.Lock()
	c.cache[image] = digestEntry{digest: digest, err: err, fetchedAt: time.Now()}
	c.mu.Unlock()

	return digest, err
}

var (
	sharedOnce    sync.Once
	sharedChecker *Checker
)

// DefaultTTL is how long a remote digest stays cached. Tags move rarely, and
// a long TTL keeps registry traffic negligible even on hosts running many
// containers.
const DefaultTTL = 6 * time.Hour

// Shared returns the process-wide checker. The cache is deliberately global:
// the same image is often deployed across several hosts, and each distinct
// image reference then costs a single request no matter how many containers
// or hosts run it.
func Shared() *Checker {
	sharedOnce.Do(func() {
		sharedChecker = NewChecker(NewRegistry(10*time.Second), DefaultTTL)
	})
	return sharedChecker
}

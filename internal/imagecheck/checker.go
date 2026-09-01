package imagecheck

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/sync/singleflight"
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
	ttl       time.Duration
}

// errorTTL is how long a failed lookup is remembered. A registry that is
// genuinely unreachable should not be retried on every page view, but a
// transient blip must not silence checks for the whole success TTL.
const errorTTL = 5 * time.Minute

// Checker resolves remote digests with a TTL cache in front of the registry.
// The cache holds only the remote digest, never the verdict, so a container
// that is updated locally reports up-to-date immediately without waiting for
// the entry to expire.
// maxCacheEntries bounds the digest cache. Hosts churn through images over a
// long uptime, and nothing else ever removes an entry.
const maxCacheEntries = 500

type Checker struct {
	registry *Registry
	ttl      time.Duration

	// Collapses concurrent lookups for the same image into one request. The
	// same image is often deployed across several hosts, which otherwise all
	// miss the cache at once and each hit the registry.
	group singleflight.Group

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
		log.Debug().Err(err).Str("image", image).Msg("image update check: unparsable reference")
		result.Status = StatusUnknown
		result.Reason = err.Error()
		return result
	}

	log.Debug().
		Str("image", image).
		Str("registry", ref.Registry).
		Str("repository", ref.Repository).
		Str("tag", ref.Tag).
		Bool("force", force).
		Strs("localDigests", localDigests).
		Msg("image update check: starting")

	if ref.Pinned() {
		log.Debug().Str("image", image).Msg("image update check: reference is digest pinned")
		result.Status = StatusPinned
		result.LocalDigest = ref.Digest
		return result
	}

	if len(localDigests) == 0 {
		log.Debug().Str("image", image).Msg("image update check: no local registry digest, likely built locally")
		result.Status = StatusNotCheckable
		result.Reason = "image has no registry digest locally"
		return result
	}

	remote, err := c.remoteDigest(ctx, image, ref, force)
	if err != nil {
		log.Debug().Err(err).Str("image", image).Msg("image update check: could not resolve remote digest")
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

	// An image can carry RepoDigests for several repositories (it was pulled
	// from one and pushed to another). Only digests for the repository being
	// checked say anything about whether this container is current.
	local := digestsForRepository(localDigests, ref)
	if len(local) == 0 {
		log.Debug().
			Str("image", image).
			Strs("localDigests", localDigests).
			Str("repository", ref.Repository).
			Msg("image update check: no local digest belongs to this repository")
		result.Status = StatusNotCheckable
		result.Reason = "image has no registry digest for " + ref.Repository
		return result
	}

	result.LocalDigest = local[0]

	if slices.Contains(local, remote) {
		result.Status = StatusUpToDate
	} else {
		result.Status = StatusUpdateAvailable
	}

	log.Debug().
		Str("image", image).
		Strs("local", local).
		Str("remote", remote).
		Str("status", string(result.Status)).
		Msg("image update check: compared digests")

	return result
}

// evictLocked drops expired entries once the cache grows past its bound, and
// falls back to clearing it entirely if everything is still live. Callers must
// hold c.mu.
func (c *Checker) evictLocked() {
	if len(c.cache) < maxCacheEntries {
		return
	}

	for key, entry := range c.cache {
		if time.Since(entry.fetchedAt) >= entry.ttl {
			delete(c.cache, key)
		}
	}

	// Nothing had expired, so the cache is genuinely full of live entries.
	// Dropping them costs a few HEAD requests, which is cheaper than growing
	// without limit.
	if len(c.cache) >= maxCacheEntries {
		clear(c.cache)
	}
}

// digestsForRepository keeps only the digests recorded for ref's repository.
// Entries look like "alpine@sha256:..." or "localhost:5000/app@sha256:...",
// so each repository is parsed through the same normalization as the image
// reference itself.
func digestsForRepository(repoDigests []string, ref Reference) []string {
	matched := make([]string, 0, len(repoDigests))

	for _, entry := range repoDigests {
		repo, digest, found := strings.Cut(entry, "@")
		if !found {
			continue
		}

		parsed, err := ParseReference(repo)
		if err != nil {
			continue
		}

		if parsed.Registry == ref.Registry && parsed.Repository == ref.Repository {
			matched = append(matched, digest)
		}
	}

	return matched
}

func (c *Checker) remoteDigest(ctx context.Context, image string, ref Reference, force bool) (string, error) {
	if !force {
		c.mu.Lock()
		entry, ok := c.cache[image]
		c.mu.Unlock()
		if ok && time.Since(entry.fetchedAt) < entry.ttl {
			log.Debug().
				Str("image", image).
				Str("digest", entry.digest).
				Dur("age", time.Since(entry.fetchedAt)).
				Msg("image update check: serving remote digest from cache")
			return entry.digest, entry.err
		}
	}

	fetched, err, shared := c.group.Do(image, func() (any, error) {
		return c.registry.Digest(ctx, ref)
	})
	digest, _ := fetched.(string)

	if shared {
		log.Debug().Str("image", image).Msg("image update check: joined an in-flight lookup")
		return digest, err
	}

	// Cache failures too, so an unreachable or private registry is not retried
	// on every page view.
	// A private registry will keep refusing us, so that answer is worth
	// keeping. Anything else may well be temporary.
	ttl := c.ttl
	if err != nil && !errors.Is(err, ErrAuthRequired) {
		ttl = min(errorTTL, c.ttl)
	}

	c.mu.Lock()
	c.evictLocked()
	c.cache[image] = digestEntry{digest: digest, err: err, fetchedAt: time.Now(), ttl: ttl}
	c.mu.Unlock()

	log.Debug().
		Err(err).
		Str("image", image).
		Str("digest", digest).
		Dur("ttl", ttl).
		Msg("image update check: cached remote digest")

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

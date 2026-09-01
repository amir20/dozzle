package imagecheck

import (
	"fmt"
	"net"
	"strings"
)

const (
	defaultRegistry     = "docker.io"
	defaultRegistryHost = "registry-1.docker.io"
	defaultTag          = "latest"
)

// Reference is a parsed container image reference.
type Reference struct {
	// Registry is the hostname the manifest is served from.
	Registry string
	// Repository is the path portion, normalized so that official Docker Hub
	// images carry their implicit "library/" prefix.
	Repository string
	// Tag is the mutable tag. Empty when the reference is digest-pinned.
	Tag string
	// Digest is set when the reference pins an exact manifest digest.
	Digest string
}

// Pinned reports whether the reference names an immutable digest. Pinned
// references can never go out of date, so there is nothing to check.
func (r Reference) Pinned() bool {
	return r.Digest != ""
}

// host returns the hostname to send registry requests to. Docker Hub is
// addressed as docker.io in references but served from registry-1.docker.io.
func (r Reference) host() string {
	if r.Registry == defaultRegistry {
		return defaultRegistryHost
	}
	return r.Registry
}

// Insecure reports whether this registry may be reached over plain HTTP.
// Docker treats loopback registries as insecure by default, and local
// registries are very commonly run without TLS. A non-loopback host is never
// downgraded: silently falling back to HTTP there would strip transport
// security from a real network request.
func (r Reference) Insecure() bool {
	host, _, err := net.SplitHostPort(r.Registry)
	if err != nil {
		host = r.Registry
	}

	if host == "localhost" {
		return true
	}

	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func (r Reference) scheme() string {
	if r.Insecure() {
		return "http"
	}
	return "https"
}

// manifestURL is the v2 manifest endpoint for this reference.
func (r Reference) manifestURL() string {
	target := r.Tag
	if target == "" {
		target = r.Digest
	}
	return fmt.Sprintf("%s://%s/v2/%s/manifests/%s", r.scheme(), r.host(), r.Repository, target)
}

// ParseReference splits an image reference into its registry, repository and
// tag/digest parts, applying Docker's implicit defaults.
func ParseReference(ref string) (Reference, error) {
	if ref == "" {
		return Reference{}, fmt.Errorf("empty image reference")
	}

	remainder := ref
	var digest string
	// A digest always trails the reference and may follow a tag, as in
	// "nginx:1.25@sha256:abc...".
	if i := strings.Index(remainder, "@"); i != -1 {
		digest = remainder[i+1:]
		remainder = remainder[:i]
		if digest == "" {
			return Reference{}, fmt.Errorf("invalid image reference %q: empty digest", ref)
		}
	}

	registry := defaultRegistry
	// The first path component is a registry only when it looks like a host:
	// it contains a dot or port separator, or is localhost. Otherwise it is a
	// Docker Hub namespace such as "amir20" in "amir20/dozzle".
	if i := strings.Index(remainder, "/"); i != -1 {
		candidate := remainder[:i]
		if candidate == "localhost" || strings.ContainsAny(candidate, ".:") {
			registry = candidate
			remainder = remainder[i+1:]
		}
	}

	tag := ""
	// Only treat a colon as a tag separator when it is not part of a registry
	// port, which has already been stripped above.
	if i := strings.LastIndex(remainder, ":"); i != -1 {
		tag = remainder[i+1:]
		remainder = remainder[:i]
		if tag == "" {
			return Reference{}, fmt.Errorf("invalid image reference %q: empty tag", ref)
		}
	}

	if remainder == "" {
		return Reference{}, fmt.Errorf("invalid image reference %q: empty repository", ref)
	}

	// Official images live under library/ but are referenced without it.
	if registry == defaultRegistry && !strings.Contains(remainder, "/") {
		remainder = "library/" + remainder
	}

	if tag == "" && digest == "" {
		tag = defaultTag
	}

	return Reference{
		Registry:   registry,
		Repository: remainder,
		Tag:        tag,
		Digest:     digest,
	}, nil
}

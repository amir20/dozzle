package imagecheck

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// VersionLabel is the OCI key images use to say which release they are. It is
// read from manifest annotations first and the image config's labels last.
const VersionLabel = "org.opencontainers.image.version"

// fallbackVersionLabel is the plain key some images (Collabora, for one) set
// instead of the OCI one. It is read only from an image config fetched from the
// registry: annotations are always OCI keys, and a container's labels also
// carry whatever its compose file sets, where "version" is a common key.
const fallbackVersionLabel = "version"

// Why a result with an update carries no latest version, so the UI can say so
// rather than look broken.
const (
	VersionUndeclared  = "undeclared"
	VersionRateLimited = "rate-limited"
	VersionAuth        = "auth"
	VersionFailed      = "failed"
)

func labelVersion(labels map[string]string) string {
	if v := labels[VersionLabel]; v != "" {
		return v
	}
	return labels[fallbackVersionLabel]
}

func versionReason(err error) string {
	switch {
	case errors.Is(err, ErrRateLimited), errors.Is(err, ErrLowQuota):
		return VersionRateLimited
	case errors.Is(err, ErrAuthRequired):
		return VersionAuth
	default:
		return VersionFailed
	}
}

// maxDocumentSize caps how much of a manifest or image config is read. Real
// ones are a few kilobytes.
const maxDocumentSize = 4 << 20

var indexTypes = map[string]bool{
	"application/vnd.oci.image.index.v1+json":                   true,
	"application/vnd.docker.distribution.manifest.list.v2+json": true,
}

type manifestDocument struct {
	MediaType   string            `json:"mediaType"`
	Annotations map[string]string `json:"annotations"`
	Manifests   []struct {
		Digest      string            `json:"digest"`
		Annotations map[string]string `json:"annotations"`
		Platform    struct {
			OS           string `json:"os"`
			Architecture string `json:"architecture"`
		} `json:"platform"`
	} `json:"manifests"`
	Config struct {
		Digest string `json:"digest"`
	} `json:"config"`
}

// Version returns the version the image at digest declares, or "" when it
// declares none. fromConfig is true when it was read from the image config's
// labels, the same place a container's labels come from. Unlike Digest these
// are GETs, which Docker Hub counts as pulls, so they are only made for an
// available update, and a digest is only ever asked once. Most images answer
// from the first manifest; the rest take one or two more requests.
func (r *Registry) Version(ctx context.Context, ref Reference, digest string) (version string, fromConfig bool, err error) {
	ref.Tag, ref.Digest = "", digest

	doc, err := r.manifest(ctx, ref)
	if err != nil {
		return "", false, err
	}
	if v := doc.Annotations[VersionLabel]; v != "" {
		return v, false, nil
	}

	if indexTypes[doc.MediaType] || len(doc.Manifests) > 0 {
		// Labels are the same for every platform of a release, so any real
		// image will do. Attestations are listed as unknown/unknown.
		chosen := -1
		for i, m := range doc.Manifests {
			if m.Platform.OS == "unknown" || m.Platform.Architecture == "unknown" {
				continue
			}
			if chosen == -1 || m.Platform.Architecture == runtime.GOARCH {
				chosen = i
			}
		}
		if chosen == -1 {
			return "", false, nil
		}

		entry := doc.Manifests[chosen]
		if v := entry.Annotations[VersionLabel]; v != "" {
			return v, false, nil
		}

		ref.Digest = entry.Digest
		if doc, err = r.manifest(ctx, ref); err != nil {
			return "", false, err
		}
		if v := doc.Annotations[VersionLabel]; v != "" {
			return v, false, nil
		}
	}

	if doc.Config.Digest == "" {
		return "", false, nil
	}

	version, err = r.configVersion(ctx, ref, doc.Config.Digest)
	return version, true, err
}

func (r *Registry) manifest(ctx context.Context, ref Reference) (manifestDocument, error) {
	var doc manifestDocument

	if r.quotaLow(ref) {
		return doc, ErrLowQuota
	}

	log.Debug().Str("url", ref.manifestURL()).Msg("image version: GET manifest")
	resp, err := r.send(ctx, ref, http.MethodGet, ref.manifestURL(), acceptManifests)
	if err != nil {
		return doc, err
	}
	defer resp.Body.Close()

	if err := statusError(resp, ref); err != nil {
		return doc, err
	}

	if err := json.NewDecoder(io.LimitReader(resp.Body, maxDocumentSize)).Decode(&doc); err != nil {
		return doc, fmt.Errorf("invalid manifest for %s: %w", ref.Repository, err)
	}
	// Docker v2 manifest lists may only say what they are in the header.
	if doc.MediaType == "" {
		doc.MediaType, _, _ = strings.Cut(resp.Header.Get("Content-Type"), ";")
	}

	return doc, nil
}

// configVersion reads the version label from an image config blob. Registries
// usually redirect blobs to a CDN; the client drops the token when it follows
// a redirect to another host.
func (r *Registry) configVersion(ctx context.Context, ref Reference, digest string) (string, error) {
	log.Debug().Str("url", ref.blobURL(digest)).Msg("image version: GET config")
	resp, err := r.send(ctx, ref, http.MethodGet, ref.blobURL(digest), "")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if err := statusError(resp, ref); err != nil {
		return "", err
	}

	var config struct {
		Config struct {
			Labels map[string]string `json:"Labels"`
		} `json:"config"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxDocumentSize)).Decode(&config); err != nil {
		return "", fmt.Errorf("invalid image config for %s: %w", ref.Repository, err)
	}

	return labelVersion(config.Config.Labels), nil
}

type versionEntry struct {
	version    string
	fromConfig bool
	err        error
	fetchedAt  time.Time
}

// versionTimeout bounds a version lookup, which runs apart from the caller's
// context because singleflight hands its answer to every caller waiting on it.
const versionTimeout = 15 * time.Second

// WithVersions fills in the versions either side of an available update, and
// returns result unchanged otherwise. A version is a nicety, so a lookup that
// fails leaves that side empty rather than failing the check. labels are the
// container's: Docker merges the image's labels into them, so the running
// version is often known without a request.
func (c *Checker) WithVersions(ctx context.Context, result Result, labels map[string]string) Result {
	if !result.UpdateAvailable() {
		return result
	}

	latest, latestFromConfig, err := c.version(ctx, result.Image, result.RemoteDigest)
	if err != nil {
		log.Debug().Err(err).Str("image", result.Image).Msg("image version: could not read latest")
		result.VersionReason = versionReason(err)
		return result
	}
	if latest == "" {
		log.Debug().Str("image", result.Image).Msg("image version: image declares no version")
		result.VersionReason = VersionUndeclared
		return result
	}
	result.LatestVersion = latest

	// Without the latest there is nothing to compare against, so the running
	// version is only worth a request once the latest is known. The labels are
	// only comparable with a latest read from labels too: an image annotated
	// 1.4.2 can still carry the 24.04 label of the base it was built on.
	if latestFromConfig {
		result.CurrentVersion = labels[VersionLabel]
	}
	if result.CurrentVersion == "" && result.LocalDigest != "" {
		result.CurrentVersion, _, err = c.version(ctx, result.Image, result.LocalDigest)
		if err != nil {
			log.Debug().Err(err).Str("image", result.Image).Msg("image version: could not read current")
		}
	}

	return result
}

// version returns the version declared by image at digest. A digest never
// changes what it points to, so answers, including "no version", are kept
// until the cache fills. Failures are kept briefly, so a registry that refuses
// is not asked again on every check, and a skipped or timed out lookup is not
// kept at all.
func (c *Checker) version(ctx context.Context, image, digest string) (string, bool, error) {
	ref, err := ParseReference(image)
	if err != nil {
		return "", false, err
	}

	key := ref.Registry + "/" + ref.Repository + "@" + digest

	c.mu.Lock()
	entry, ok := c.versions[key]
	c.mu.Unlock()
	if ok && (entry.err == nil || time.Since(entry.fetchedAt) < errorTTL) {
		return entry.version, entry.fromConfig, entry.err
	}

	fetched, err, _ := c.group.Do("version:"+key, func() (any, error) {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), versionTimeout)
		defer cancel()
		version, fromConfig, err := c.registry.Version(ctx, ref, digest)
		return versionEntry{version: version, fromConfig: fromConfig}, err
	})
	found, _ := fetched.(versionEntry)

	if !errors.Is(err, ErrLowQuota) && !errors.Is(err, context.DeadlineExceeded) {
		c.mu.Lock()
		if len(c.versions) >= maxCacheEntries {
			clear(c.versions)
		}
		c.versions[key] = versionEntry{version: found.version, fromConfig: found.fromConfig, err: err, fetchedAt: time.Now()}
		c.mu.Unlock()
	}

	return found.version, found.fromConfig, err
}

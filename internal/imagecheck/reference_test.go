package imagecheck

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseReference(t *testing.T) {
	tests := []struct {
		name     string
		ref      string
		registry string
		repo     string
		tag      string
		digest   string
		url      string
	}{
		{
			name:     "official image without tag",
			ref:      "alpine",
			registry: "docker.io",
			repo:     "library/alpine",
			tag:      "latest",
			url:      "https://registry-1.docker.io/v2/library/alpine/manifests/latest",
		},
		{
			name:     "official image with tag",
			ref:      "postgres:18-alpine",
			registry: "docker.io",
			repo:     "library/postgres",
			tag:      "18-alpine",
			url:      "https://registry-1.docker.io/v2/library/postgres/manifests/18-alpine",
		},
		{
			name:     "docker hub namespace",
			ref:      "amir20/dozzle:latest",
			registry: "docker.io",
			repo:     "amir20/dozzle",
			tag:      "latest",
			url:      "https://registry-1.docker.io/v2/amir20/dozzle/manifests/latest",
		},
		{
			name:     "third party registry",
			ref:      "ghcr.io/home-assistant/home-assistant:stable",
			registry: "ghcr.io",
			repo:     "home-assistant/home-assistant",
			tag:      "stable",
			url:      "https://ghcr.io/v2/home-assistant/home-assistant/manifests/stable",
		},
		{
			name:     "registry with single path segment",
			ref:      "mcr.microsoft.com/playwright:v1.62.1-jammy",
			registry: "mcr.microsoft.com",
			repo:     "playwright",
			tag:      "v1.62.1-jammy",
			url:      "https://mcr.microsoft.com/v2/playwright/manifests/v1.62.1-jammy",
		},
		{
			name:     "registry with port is not mistaken for a tag",
			ref:      "localhost:5000/my/app:dev",
			registry: "localhost:5000",
			repo:     "my/app",
			tag:      "dev",
			url:      "https://localhost:5000/v2/my/app/manifests/dev",
		},
		{
			name:     "digest pinned",
			ref:      "nginx@sha256:abc123",
			registry: "docker.io",
			repo:     "library/nginx",
			digest:   "sha256:abc123",
		},
		{
			name:     "tag and digest together",
			ref:      "nginx:1.25@sha256:abc123",
			registry: "docker.io",
			repo:     "library/nginx",
			tag:      "1.25",
			digest:   "sha256:abc123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref, err := ParseReference(tt.ref)
			require.NoError(t, err)

			assert.Equal(t, tt.registry, ref.Registry)
			assert.Equal(t, tt.repo, ref.Repository)
			assert.Equal(t, tt.tag, ref.Tag)
			assert.Equal(t, tt.digest, ref.Digest)

			if tt.url != "" {
				assert.Equal(t, tt.url, ref.manifestURL())
			}
		})
	}
}

func TestParseReferenceErrors(t *testing.T) {
	for _, ref := range []string{"", "nginx:", "nginx@"} {
		_, err := ParseReference(ref)
		assert.Error(t, err, "expected %q to be rejected", ref)
	}
}

func TestReferencePinned(t *testing.T) {
	pinned, err := ParseReference("nginx@sha256:abc")
	require.NoError(t, err)
	assert.True(t, pinned.Pinned())

	tagged, err := ParseReference("nginx:1.25")
	require.NoError(t, err)
	assert.False(t, tagged.Pinned())
}

func TestParseChallenge(t *testing.T) {
	realm, service := parseChallenge(`Bearer realm="https://auth.docker.io/token",service="registry.docker.io",scope="repository:library/nginx:pull"`)
	assert.Equal(t, "https://auth.docker.io/token", realm)
	assert.Equal(t, "registry.docker.io", service)

	realm, service = parseChallenge("Basic realm=\"something\"")
	assert.Empty(t, realm)
	assert.Empty(t, service)
}

//go:build live

package imagecheck

import (
	"context"
	"strings"
	"testing"
	"time"
)

// TestLiveRegistries exercises the real registry path. It is behind the "live"
// build tag because it needs network access, so it never runs in CI.
//
//	go test -tags live ./internal/imagecheck/ -run TestLiveRegistries -v
func TestLiveRegistries(t *testing.T) {
	checker := NewChecker(NewRegistry(15*time.Second), time.Hour)

	// Local digests are repository qualified, exactly as Docker records them
	// in RepoDigests.
	cases := []struct {
		image  string
		local  []string
		expect Status
	}{
		{
			"grafana/grafana:13.1.3",
			[]string{"grafana/grafana@sha256:ab5cb380e3ff3172d6c8bd2e7cfd31cce977d2881b260e1f5bc089bf0b759b43"},
			StatusUpToDate,
		},
		{
			"mcr.microsoft.com/playwright:v1.62.1-jammy",
			[]string{"mcr.microsoft.com/playwright@sha256:b3251f7ff1a9fa559a28d1c67eaa15fc1a9800f7845e82756caea7842967f615"},
			StatusUpToDate,
		},
		// Docker Hub answers to several hostnames; all of them must line up
		// with the bare "repo@sha256:..." form Docker records locally.
		{
			"index.docker.io/grafana/grafana:13.1.3",
			[]string{"grafana/grafana@sha256:ab5cb380e3ff3172d6c8bd2e7cfd31cce977d2881b260e1f5bc089bf0b759b43"},
			StatusUpToDate,
		},
		{"amir20/dozzle:latest", []string{"amir20/dozzle@sha256:stale"}, StatusUpdateAvailable},
		{
			"ghcr.io/home-assistant/home-assistant:stable",
			[]string{"ghcr.io/home-assistant/home-assistant@sha256:stale"},
			StatusUpdateAvailable,
		},
	}

	for _, tc := range cases {
		result := checker.Check(context.Background(), tc.image, tc.local, false)
		t.Logf("%-46s -> %-18s %s", tc.image, result.Status, result.RemoteDigest)

		if result.Status != tc.expect {
			t.Errorf("%s: got %s (%s), want %s", tc.image, result.Status, result.Reason, tc.expect)
		}
	}
}

// TestLiveVersions reads versions for whatever each tag currently points at.
//
//	go test -tags live ./internal/imagecheck/ -run TestLiveVersions -v
func TestLiveVersions(t *testing.T) {
	checker := NewChecker(NewRegistry(15*time.Second), time.Hour)

	for _, image := range []string{
		"postgres:18-alpine",
		"grafana/grafana:latest",
		"ghcr.io/home-assistant/home-assistant:stable",
		"amir20/dozzle:latest",
	} {
		result := checker.Check(context.Background(), image, []string{strings.Split(image, ":")[0] + "@sha256:stale"}, false)
		if result.RemoteDigest == "" {
			t.Errorf("%s: no remote digest (%s)", image, result.Reason)
			continue
		}

		version, err := checker.version(context.Background(), image, result.RemoteDigest)
		t.Logf("%-46s -> %q", image, version)
		if err != nil {
			t.Errorf("%s: %v", image, err)
		}
	}
}

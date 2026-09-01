//go:build live

package imagecheck

import (
	"context"
	"testing"
	"time"
)

// TestLiveRegistries exercises the real registry path. It is behind the "live"
// build tag because it needs network access, so it never runs in CI.
//
//	go test -tags live ./internal/imagecheck/ -run TestLiveRegistries -v
func TestLiveRegistries(t *testing.T) {
	checker := NewChecker(NewRegistry(15*time.Second), time.Hour)

	cases := []struct {
		image  string
		local  []string
		expect Status
	}{
		{"grafana/grafana:13.1.3", []string{"sha256:ab5cb380e3ff3172d6c8bd2e7cfd31cce977d2881b260e1f5bc089bf0b759b43"}, StatusUpToDate},
		{"mcr.microsoft.com/playwright:v1.62.1-jammy", []string{"sha256:b3251f7ff1a9fa559a28d1c67eaa15fc1a9800f7845e82756caea7842967f615"}, StatusUpToDate},
		{"amir20/dozzle:latest", []string{"sha256:stale"}, StatusUpdateAvailable},
		{"ghcr.io/home-assistant/home-assistant:stable", []string{"sha256:stale"}, StatusUpdateAvailable},
	}

	for _, tc := range cases {
		result := checker.Check(context.Background(), tc.image, tc.local, false)
		t.Logf("%-46s -> %-18s %s", tc.image, result.Status, result.RemoteDigest)

		if result.Status != tc.expect {
			t.Errorf("%s: got %s (%s), want %s", tc.image, result.Status, result.Reason, tc.expect)
		}
	}
}

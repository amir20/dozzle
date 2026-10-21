package cloud

import (
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/stretchr/testify/assert"
)

// What Dozzle pushes comes back the same, except the digests, which Cloud keeps
// bare.
func TestUpdateRecordRoundTrip(t *testing.T) {
	at := time.Date(2026, 10, 21, 4, 0, 0, 123, time.UTC)
	r := container.UpdateRecord{
		Host:         "h1",
		Name:         "web",
		OldID:        "aaaaaaaaaaaa",
		NewID:        "bbbbbbbbbbbb",
		ImageRef:     "nginx:latest",
		FromDigest:   "nginx@sha256:1111",
		ToDigest:     "nginx@sha256:2222",
		FromImageID:  "sha256:from",
		ToImageID:    "sha256:to",
		OldStartedAt: at.Add(-time.Hour),
		At:           at,
		Source:       container.UpdateSourceSchedule,
		RolledBack:   true,
	}

	got := updateRecordFromProto(containerUpdateProto(r))

	want := r
	want.FromDigest = "sha256:1111"
	want.ToDigest = "sha256:2222"
	assert.Equal(t, want, got)
	assert.True(t, updateRecordFromProto(containerUpdateProto(container.UpdateRecord{})).At.IsZero())
}

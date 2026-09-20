package releases

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// A dev build is not a point on the release line, so no release is newer than
// it and nothing may be marked latest.
func TestOnReleaseLine(t *testing.T) {
	for _, version := range []string{"v11.1.0", "11.1.0", "v11.1.0-beta.1"} {
		assert.True(t, OnReleaseLine(version), version)
	}
	for _, version := range []string{"master-1d901bf", "pr-5216-abc1234", "local", "head", "", "v11.1"} {
		assert.False(t, OnReleaseLine(version), version)
	}
}

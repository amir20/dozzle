package web

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The reference must be stable for as long as the secret is, and it must never
// be the username: cloud has no business knowing who signs in to somebody's
// Dozzle.
func Test_userRef(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "data"), 0o700))
	t.Chdir(dir)

	// Reset the process-wide cache so this test owns the secret it creates.
	userRefOnce = sync.Once{}
	userRefSecret = nil

	amir := userRef("amir")
	assert.NotEmpty(t, amir)
	assert.NotContains(t, amir, "amir")
	assert.Len(t, amir, 32)

	// Same user, same reference, or a reopened pane loses its thread.
	assert.Equal(t, amir, userRef("amir"))
	assert.NotEqual(t, amir, userRef("someone-else"))

	// No user is no reference rather than a shared one.
	assert.Empty(t, userRef(""))
}

// A secret regenerated on restart silently changes every reference, so it has
// to survive on disk.
func Test_userRefSecret_persists(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	userRefOnce = sync.Once{}
	userRefSecret = nil

	first := userRef("amir")

	// Simulate a restart: the process cache goes, the file stays.
	userRefOnce = sync.Once{}
	userRefSecret = nil

	assert.Equal(t, first, userRef("amir"))
}

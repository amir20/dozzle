package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFreshDataDir(t *testing.T) {
	t.Run("empty dir", func(t *testing.T) {
		assert.True(t, FreshDataDir(t.TempDir()))
	})

	t.Run("missing dir", func(t *testing.T) {
		assert.True(t, FreshDataDir(filepath.Join(t.TempDir(), "nope")))
	})

	t.Run("only filesystem noise", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.Mkdir(filepath.Join(dir, "lost+found"), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, ".DS_Store"), nil, 0644))
		assert.True(t, FreshDataDir(dir))
	})

	for _, name := range []string{"notifications.yml", "users.yml", "dozzle.yml", "__default__"} {
		t.Run("existing "+name, func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(dir, name), nil, 0644))
			assert.False(t, FreshDataDir(dir))
		})
	}
}

func TestLoadMissingFileIsEmpty(t *testing.T) {
	f, err := Load(filepath.Join(t.TempDir(), "dozzle.yml"))
	require.NoError(t, err)
	assert.Equal(t, File{}, f)
}

// A hand-written dozzle.yml loads, and a key that is absent stays nil.
func TestLoadHandWrittenFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dozzle.yml")
	require.NoError(t, os.WriteFile(path, []byte(`
authProvider: simple
enableActions: false
autoUpdate: daily
autoUpdateTime: "04:30"
remoteAgents:
  - agent1:7007
  - agent2:7007
setupWindowStartedAt: 2026-01-02T03:04:05Z
`), 0644))

	f, err := Load(path)
	require.NoError(t, err)
	require.NotNil(t, f.AuthProvider)
	assert.Equal(t, "simple", *f.AuthProvider)
	require.NotNil(t, f.EnableActions)
	assert.False(t, *f.EnableActions)
	assert.Nil(t, f.EnableShell)
	assert.Equal(t, "daily", *f.AutoUpdate)
	assert.Equal(t, "04:30", *f.AutoUpdateTime)
	assert.Equal(t, []string{"agent1:7007", "agent2:7007"}, f.RemoteAgents)
	require.NotNil(t, f.SetupWindowStartedAt)
	assert.True(t, f.SetupWindowStartedAt.Equal(time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)))
}

func TestLoadInvalidYAMLFails(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dozzle.yml")
	require.NoError(t, os.WriteFile(path, []byte("authProvider: [unclosed"), 0644))
	_, err := Load(path)
	assert.Error(t, err)
}

// Update writes every field so Load reads back exactly what was set.
func TestUpdateRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dozzle.yml")
	provider, enabled, mode, at, which := "simple", true, AutoUpdateWeekly, "23:15", "all"
	started := time.Date(2026, 5, 6, 7, 8, 9, 0, time.UTC)
	want := File{
		AuthProvider:         &provider,
		EnableActions:        &enabled,
		EnableShell:          &enabled,
		AutoUpdate:           &mode,
		AutoUpdateTime:       &at,
		UpdateContainers:     &which,
		RemoteAgents:         []string{"a:7007"},
		PrivateAgents:        []string{"a:7007"},
		SetupWindowStartedAt: &started,
	}
	require.NoError(t, Update(path, func(f *File) { *f = want }))

	got, err := Load(path)
	require.NoError(t, err)
	assert.Equal(t, want, got)

	// A second update only touches what it changes.
	require.NoError(t, Update(path, func(f *File) { f.RemoteAgents = nil }))
	got, err = Load(path)
	require.NoError(t, err)
	assert.Nil(t, got.RemoteAgents)
	assert.Equal(t, want.PrivateAgents, got.PrivateAgents)
	assert.Equal(t, provider, *got.AuthProvider)
}

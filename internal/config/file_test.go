package config

import (
	"os"
	"path/filepath"
	"testing"

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

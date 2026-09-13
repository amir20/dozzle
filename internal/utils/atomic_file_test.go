package utils

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteFileAtomic_CreatesFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yml")

	err := WriteFileAtomic(path, func(w io.Writer) error {
		_, err := w.Write([]byte("hello"))
		return err
	})
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "hello", string(data))

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0644), info.Mode().Perm())
}

func TestWriteFileAtomic_FailedWriteKeepsOriginal(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")
	require.NoError(t, os.WriteFile(path, []byte("original"), 0600))

	err := WriteFileAtomic(path, func(w io.Writer) error {
		w.Write([]byte("partial"))
		return errors.New("encoder blew up")
	})
	assert.Error(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "original", string(data))

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.Len(t, entries, 1, "no temp files left behind")
}

func TestWriteFileAtomic_ReadOnlyDirWritesInPlace(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "notifications.yml")
	require.NoError(t, os.WriteFile(path, []byte("old"), 0644))
	require.NoError(t, os.Chmod(dir, 0555))
	t.Cleanup(func() { os.Chmod(dir, 0755) })

	require.NoError(t, WriteFileAtomic(path, func(w io.Writer) error {
		_, err := w.Write([]byte("new"))
		return err
	}))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "new", string(data))
}

func TestWriteFileAtomic_KeepsExistingMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cloud.yml")
	require.NoError(t, os.WriteFile(path, []byte("old"), 0600))

	require.NoError(t, WriteFileAtomic(path, func(w io.Writer) error {
		_, err := w.Write([]byte("new"))
		return err
	}))

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.Len(t, entries, 1, "no temp files left behind")
}

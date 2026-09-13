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

func writeString(s string) func(io.Writer) error {
	return func(w io.Writer) error {
		_, err := w.Write([]byte(s))
		return err
	}
}

func assertOnlyFiles(t *testing.T, dir string, names ...string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	var got []string
	for _, e := range entries {
		got = append(got, e.Name())
	}
	assert.ElementsMatch(t, names, got, "no temp files left behind")
}

func TestWriteFileAtomic_CreatesFileWithUmask(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")

	require.NoError(t, WriteFileAtomic(path, writeString("hello")))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "hello", string(data))

	// Whatever the umask is, a new file should match what os.Create produces.
	ref, err := os.Create(filepath.Join(dir, "ref"))
	require.NoError(t, err)
	ref.Close()
	refInfo, err := os.Stat(ref.Name())
	require.NoError(t, err)
	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, refInfo.Mode().Perm(), info.Mode().Perm())
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
	assertOnlyFiles(t, dir, "config.yml")
}

func TestWriteFileAtomic_KeepsExistingMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cloud.yml")
	require.NoError(t, os.WriteFile(path, []byte("old"), 0600))

	require.NoError(t, WriteFileAtomic(path, writeString("new")))

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
	assertOnlyFiles(t, dir, "cloud.yml")
}

func TestWriteFileAtomic_WritesThroughSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "shared", "notifications.yml")
	require.NoError(t, os.MkdirAll(filepath.Dir(target), 0755))
	require.NoError(t, os.WriteFile(target, []byte("old"), 0644))
	link := filepath.Join(dir, "notifications.yml")
	require.NoError(t, os.Symlink(target, link))

	require.NoError(t, WriteFileAtomic(link, writeString("new")))

	linkInfo, err := os.Lstat(link)
	require.NoError(t, err)
	assert.Equal(t, os.ModeSymlink, linkInfo.Mode()&os.ModeSymlink, "link is still a symlink")

	data, err := os.ReadFile(target)
	require.NoError(t, err)
	assert.Equal(t, "new", string(data))
	assertOnlyFiles(t, filepath.Dir(target), "notifications.yml")
}

func TestWriteFileAtomic_RenameFailureWritesInPlace(t *testing.T) {
	t.Cleanup(func() { rename = os.Rename })
	rename = func(string, string) error { return errors.New("device or resource busy") }

	dir := t.TempDir()
	path := filepath.Join(dir, "notifications.yml")
	require.NoError(t, os.WriteFile(path, []byte("a much longer old value"), 0644))

	require.NoError(t, WriteFileAtomic(path, writeString("new")))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "new", string(data))
	assertOnlyFiles(t, dir, "notifications.yml")
}

func TestWriteFileAtomic_TempFileFailureWritesInPlace(t *testing.T) {
	t.Cleanup(func() { createTemp = os.CreateTemp })
	createTemp = func(string, string) (*os.File, error) { return nil, os.ErrPermission }

	dir := t.TempDir()
	path := filepath.Join(dir, "notifications.yml")
	require.NoError(t, os.WriteFile(path, []byte("old"), 0644))

	require.NoError(t, WriteFileAtomic(path, writeString("new")))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "new", string(data))
}

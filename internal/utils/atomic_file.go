package utils

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
)

// Swapped in tests to force the fallback paths regardless of who runs them.
var (
	createTemp = os.CreateTemp
	rename     = os.Rename
)

// WriteFileAtomic renders the file into memory first, then writes it to a
// temp file in the same directory and renames it over path. A crash or a
// failing encoder leaves the previous file intact instead of a truncated one.
//
// A symlinked path is resolved first, so the link survives and its target is
// what gets replaced. The existing file's mode is kept.
//
// A file that doesn't exist yet is created directly with O_EXCL: there is no
// previous version to lose, and it picks up the process umask like os.Create.
//
// Rename fails when path is a single-file bind mount (EBUSY), and the temp file
// can't be created when the dir around such a mount is read-only, so in both
// cases it falls back to rewriting path in place. The bytes are already fully
// rendered by then, so the only unsafe window left is the write itself.
func WriteFileAtomic(path string, write func(io.Writer) error) error {
	var buf bytes.Buffer
	if err := write(&buf); err != nil {
		return err
	}
	data := buf.Bytes()

	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		path = resolved
	}

	info, err := os.Stat(path)
	if errors.Is(err, fs.ErrNotExist) {
		err = writeFile(path, data, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0666)
		if !errors.Is(err, fs.ErrExist) {
			return err
		}
		// Lost a race with another writer creating it, so replace it like any existing file.
		info, err = os.Stat(path)
	}
	if err != nil {
		return err
	}
	perm := info.Mode().Perm()

	dir := filepath.Dir(path)
	tmp, err := createTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		// The dir can be read-only while the file itself is a writable bind mount.
		log.Debug().Err(err).Str("path", path).Msg("Could not create temp file, writing in place")
		return writeInPlace(path, data)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath) // no-op once renamed

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpPath, perm); err != nil {
		return err
	}

	if err := rename(tmpPath, path); err != nil {
		log.Debug().Err(err).Str("path", path).Msg("Atomic rename failed, writing in place")
		return writeInPlace(path, data)
	}

	// Best effort: persists the rename itself. Some filesystems refuse to sync a dir.
	if d, err := os.Open(dir); err == nil {
		if err := d.Sync(); err != nil {
			log.Debug().Err(err).Str("dir", dir).Msg("Could not sync directory after rename")
		}
		d.Close()
	}
	return nil
}

func writeInPlace(path string, data []byte) error {
	return writeFile(path, data, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
}

func writeFile(path string, data []byte, flag int, perm os.FileMode) error {
	f, err := os.OpenFile(path, flag, perm)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

package utils

import (
	"bytes"
	"io"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
)

// WriteFileAtomic renders the file into memory first, then writes it to a
// temp file in the same directory and renames it over path. A crash or a
// failing encoder leaves the previous file intact instead of a truncated one.
//
// The existing file's mode is kept; a new file gets 0644.
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

	perm := os.FileMode(0644)
	if info, err := os.Stat(path); err == nil {
		perm = info.Mode().Perm()
	}

	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		// The dir can be read-only while the file itself is a writable bind mount.
		log.Debug().Err(err).Str("path", path).Msg("Could not create temp file, writing in place")
		return writeInPlace(path, data, perm)
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

	if err := os.Rename(tmpPath, path); err != nil {
		log.Debug().Err(err).Str("path", path).Msg("Atomic rename failed, writing in place")
		return writeInPlace(path, data, perm)
	}

	if d, err := os.Open(dir); err == nil {
		d.Sync()
		d.Close()
	}
	return nil
}

func writeInPlace(path string, data []byte, perm os.FileMode) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
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

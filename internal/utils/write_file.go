package utils

import (
	"bytes"
	"io"
	"os"
	"sync"
)

// One lock for every path: saves are rare and small, and a per-path map would
// grow with every distinct profile username.
var writeMu sync.Mutex

// WriteFile renders the file into memory before touching disk, so a failing
// encoder never leaves a half-written file, then writes it in place and fsyncs.
// Saves are serialized so two of them can't interleave.
//
// It writes through os.Create on purpose: a temp file plus rename would also
// survive power loss, but breaks single-file bind mounts, symlinks and umask.
func WriteFile(path string, write func(io.Writer) error) error {
	var buf bytes.Buffer
	if err := write(&buf); err != nil {
		return err
	}

	writeMu.Lock()
	defer writeMu.Unlock()

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if _, err := f.Write(buf.Bytes()); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

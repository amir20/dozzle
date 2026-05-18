//go:build !windows

package container

import "syscall"

// statfs returns (total, free) bytes for the filesystem hosting path.
func statfs(path string) (total uint64, free uint64, err error) {
	var st syscall.Statfs_t
	if err = syscall.Statfs(path, &st); err != nil {
		return 0, 0, err
	}
	bsize := uint64(st.Bsize)
	return st.Blocks * bsize, st.Bavail * bsize, nil
}

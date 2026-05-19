//go:build windows

package container

import "errors"

func statfs(_ string) (total uint64, free uint64, err error) {
	return 0, 0, errors.New("statfs not supported on windows")
}

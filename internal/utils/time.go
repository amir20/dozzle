package utils

import "time"

func Max(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

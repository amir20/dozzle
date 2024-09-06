package utils

import "time"

func Min(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

func Max(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

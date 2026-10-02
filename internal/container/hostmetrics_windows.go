//go:build windows

package container

// ReadHostMetrics is not supported on Windows.
func ReadHostMetrics(_ string) (HostMetrics, bool) {
	return HostMetrics{}, false
}

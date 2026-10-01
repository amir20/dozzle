//go:build windows

package container

// ReadHostMetrics is not supported on Windows.
func ReadHostMetrics() (HostMetrics, bool) {
	return HostMetrics{}, false
}

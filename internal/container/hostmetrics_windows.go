//go:build windows

package container

// ReadHostMetrics is not supported on Windows.
func ReadHostMetrics(_ string, _ bool) (HostMetrics, bool) {
	return HostMetrics{}, false
}

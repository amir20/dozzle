package notification

import "testing"

// A bare "exit code 137" reads as an OOM kill to both people and to Dozzle
// Cloud's triage model, when it actually means SIGKILL — what `docker stop`, a
// reboot, or a Watchtower update cycle produces once a container misses the
// SIGTERM grace period. Naming the signal keeps that misreading from starting
// here, at the only place the exit code is turned into prose.
func TestDescribeExitCode(t *testing.T) {
	tests := []struct {
		name     string
		exitCode string
		want     string
	}{
		{"SIGKILL is named, not left to be read as OOM", "137", "137, SIGKILL"},
		{"graceful stop", "143", "143, SIGTERM"},
		{"interrupt", "130", "130, SIGINT"},
		{"segfault", "139", "139, SIGSEGV"},
		{"clean exit stays bare", "0", "0"},
		{"application failure stays bare", "1", "1"},
		{"docker error stays bare", "125", "125"},
		{"unknown code stays bare", "42", "42"},
		{"empty stays empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := describeExitCode(tt.exitCode); got != tt.want {
				t.Errorf("describeExitCode(%q) = %q, want %q", tt.exitCode, got, tt.want)
			}
		})
	}
}

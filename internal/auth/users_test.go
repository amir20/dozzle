package auth

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Every role bit must be covered by All. Adding a new role without adding it to
// All is what let Cloud fall outside the default in the first place.
func TestAllCoversEveryRole(t *testing.T) {
	for name, role := range map[string]Role{
		"shell":         Shell,
		"actions":       Actions,
		"download":      Download,
		"notifications": Notifications,
		"cloud":         Cloud,
	} {
		require.Truef(t, All.Has(role), "All is missing %s; new roles must be added to All", name)
	}
}

// A user in users.yml with no roles key gets everything.
func TestUserWithoutConfiguredRolesGetsAll(t *testing.T) {
	require.Equal(t, All, ParseRole("all"))
}

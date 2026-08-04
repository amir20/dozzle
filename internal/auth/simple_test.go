package auth

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// The JWT signing key is derived from the users in users.yml so that a token
// stays valid across restarts and is only invalidated when a password or role
// actually changes. Deriving it by ranging over the user map made the digest
// depend on Go's randomized map iteration order, so every restart of a
// multi-user setup rolled the key and logged everyone out.
func TestSimpleAuthSigningKeyIsStableAcrossRestarts(t *testing.T) {
	users := UserDatabase{
		Users: map[string]*User{
			"alice": {Username: "alice", Password: "$2a$11$aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", RolesConfigured: "all"},
			"bob":   {Username: "bob", Password: "$2a$11$bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", RolesConfigured: "shell"},
			"carol": {Username: "carol", Password: "$2a$11$cccccccccccccccccccccccccccccccccccccccccccccccccccccc", RolesConfigured: "actions"},
			"dave":  {Username: "dave", Password: "$2a$11$dddddddddddddddddddddddddddddddddddddddddddddddddddddd", RolesConfigured: "download"},
		},
	}

	_, token, err := NewSimpleAuth(users, 0).tokenAuth.Encode(map[string]any{"username": "alice"})
	require.NoError(t, err)

	// Each iteration stands in for a Dozzle restart against an unchanged users.yml.
	for i := range 50 {
		if _, err := NewSimpleAuth(users, 0).tokenAuth.Decode(token); err != nil {
			t.Fatalf("token issued before restart %d was rejected: %v", i+1, err)
		}
	}
}

// Changing a password or a role must still roll the key so old tokens stop working.
func TestSimpleAuthSigningKeyChangesWhenCredentialsChange(t *testing.T) {
	users := UserDatabase{
		Users: map[string]*User{
			"alice": {Username: "alice", Password: "$2a$11$aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", RolesConfigured: "all"},
			"bob":   {Username: "bob", Password: "$2a$11$bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", RolesConfigured: "shell"},
		},
	}

	_, token, err := NewSimpleAuth(users, 0).tokenAuth.Encode(map[string]any{"username": "alice"})
	require.NoError(t, err)

	users.Users["bob"].RolesConfigured = "shell,actions"
	_, err = NewSimpleAuth(users, 0).tokenAuth.Decode(token)
	require.Error(t, err)
}

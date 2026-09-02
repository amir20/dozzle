package auth

import (
	"net/http"
	"net/http/httptest"
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

// serveWithAuth runs a request carrying token through the simple auth middleware
// and returns the user the handlers would see.
func serveWithAuth(t *testing.T, a *simpleAuthContext, token string) *User {
	t.Helper()

	var user *User
	handler := a.AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user = UserFromContext(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	return user
}

// The roles claim is a bitmask frozen at login. Adding a role to the code (cloud
// was the case that broke) leaves every existing session on the old mask without
// the new bit, and users.yml is unchanged so the signing key does not roll either.
// Roles have to come from the database on each request, not from the token.
func TestSimpleAuthUsesCurrentRolesNotTheTokensRoles(t *testing.T) {
	users := UserDatabase{
		Users: map[string]*User{
			"alice": {Username: "alice", Name: "Alice", Password: "$2a$11$aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", RolesConfigured: "all", Roles: All},
		},
	}

	a := NewSimpleAuth(users, 0)

	// A session minted back when All did not include Cloud.
	stale := All &^ Cloud
	_, token, err := a.tokenAuth.Encode(map[string]any{"username": "alice", "roles": float64(stale)})
	require.NoError(t, err)

	user := serveWithAuth(t, a, token)
	require.NotNil(t, user)
	require.Equal(t, All, user.Roles)
	require.True(t, user.Roles.Has(Cloud), "cloud must come back without forcing a re-login")
}

// A narrowed role in users.yml must apply to live sessions too, not only after
// the user logs out.
func TestSimpleAuthAppliesRevokedRolesToExistingSessions(t *testing.T) {
	users := UserDatabase{
		Users: map[string]*User{
			"alice": {Username: "alice", Password: "$2a$11$aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", RolesConfigured: "all,^shell", Roles: ParseRole("all,^shell")},
		},
	}

	a := NewSimpleAuth(users, 0)
	_, token, err := a.tokenAuth.Encode(map[string]any{"username": "alice", "roles": float64(All)})
	require.NoError(t, err)

	user := serveWithAuth(t, a, token)
	require.NotNil(t, user)
	require.False(t, user.Roles.Has(Shell))
	require.True(t, user.Roles.Has(Actions))
}

// A token for a user who is gone from users.yml resolves to no user, so the
// request is unauthenticated rather than running with the token's claims.
func TestSimpleAuthRejectsTokenForUnknownUser(t *testing.T) {
	users := UserDatabase{
		Users: map[string]*User{
			"alice": {Username: "alice", Password: "$2a$11$aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", RolesConfigured: "all", Roles: All},
		},
	}

	a := NewSimpleAuth(users, 0)
	_, token, err := a.tokenAuth.Encode(map[string]any{"username": "mallory", "roles": float64(All)})
	require.NoError(t, err)

	require.Nil(t, serveWithAuth(t, a, token))
}

// A session minted before roles existed at all carries no roles claim. Resolving
// against users.yml means the session keeps exactly the permissions it is
// configured for instead of silently losing all of them on upgrade.
func TestSimpleAuthResolvesTokenWithNoRolesClaim(t *testing.T) {
	users := UserDatabase{
		Users: map[string]*User{
			"alice": {Username: "alice", Password: "$2a$11$aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", RolesConfigured: "all", Roles: All},
		},
	}

	a := NewSimpleAuth(users, 0)
	_, token, err := a.tokenAuth.Encode(map[string]any{"username": "alice"})
	require.NoError(t, err)

	user := serveWithAuth(t, a, token)
	require.NotNil(t, user)
	require.Equal(t, All, user.Roles)
}

// No token at all stays unauthenticated so the login routes keep working.
func TestSimpleAuthWithoutTokenHasNoUser(t *testing.T) {
	users := UserDatabase{
		Users: map[string]*User{
			"alice": {Username: "alice", Password: "$2a$11$aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", RolesConfigured: "all", Roles: All},
		},
	}

	require.Nil(t, serveWithAuth(t, NewSimpleAuth(users, 0), ""))
}

// The container filter comes from users.yml as well, both parsed into labels for
// downstream filtering and kept as the raw string on the user.
func TestSimpleAuthResolvesFilterFromDatabase(t *testing.T) {
	users := UserDatabase{
		Users: map[string]*User{
			"alice": {Username: "alice", Password: "$2a$11$aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Filter: "name=foo", RolesConfigured: "all", Roles: All},
		},
	}

	a := NewSimpleAuth(users, 0)
	_, token, err := a.tokenAuth.Encode(map[string]any{"username": "alice", "filter": "name=stale"})
	require.NoError(t, err)

	user := serveWithAuth(t, a, token)
	require.NotNil(t, user)
	require.Equal(t, "name=foo", user.Filter)
	require.Equal(t, []string{"foo"}, user.ContainerLabels["name"])
}

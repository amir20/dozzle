package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/jwtauth/v5"
	"github.com/stretchr/testify/require"
)

// userFromToken runs claims through the real Verifier so the test exercises the
// same path a browser's session cookie takes.
func userFromToken(t *testing.T, claims map[string]any) *User {
	t.Helper()

	tokenAuth := jwtauth.New("HS256", []byte("secret"), nil)
	_, tokenString, err := tokenAuth.Encode(claims)
	require.NoError(t, err)

	var user *User
	handler := jwtauth.Verifier(tokenAuth)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user = UserFromContext(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	return user
}

// Tokens minted before roles existed carry no roles claim. Reading that as None
// silently stripped every permission from sessions that were otherwise still
// valid, so upgrading Dozzle hid notifications, cloud and downloads until the
// user happened to log out. Absent means full access, the same as an absent
// `roles` in users.yml and an absent roles header in proxy auth.
func TestUserFromContextMissingRolesClaimGrantsAll(t *testing.T) {
	user := userFromToken(t, map[string]any{
		"username": "alice",
		"email":    "alice@example.com",
		"name":     "Alice",
	})

	require.NotNil(t, user)
	require.Equal(t, All, user.Roles)
	for name, role := range map[string]Role{
		"shell":         Shell,
		"actions":       Actions,
		"download":      Download,
		"notifications": Notifications,
		"cloud":         Cloud,
	} {
		require.Truef(t, user.Roles.Has(role), "expected %s to be granted", name)
	}
}

// An explicit claim still wins, including one that grants nothing.
func TestUserFromContextHonoursExplicitRolesClaim(t *testing.T) {
	user := userFromToken(t, map[string]any{
		"username": "alice",
		"email":    "alice@example.com",
		"name":     "Alice",
		"roles":    float64(Shell | Actions),
	})

	require.NotNil(t, user)
	require.True(t, user.Roles.Has(Shell))
	require.True(t, user.Roles.Has(Actions))
	require.False(t, user.Roles.Has(Cloud))
	require.False(t, user.Roles.Has(Download))
}

func TestUserFromContextExplicitNoneGrantsNothing(t *testing.T) {
	user := userFromToken(t, map[string]any{
		"username": "alice",
		"email":    "alice@example.com",
		"name":     "Alice",
		"roles":    float64(None),
	})

	require.NotNil(t, user)
	require.Equal(t, None, user.Roles)
	require.False(t, user.Roles.Has(Cloud))
}

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

// A user in users.yml with no roles key gets everything, which is the behaviour
// the JWT default above mirrors.
func TestUserWithoutConfiguredRolesGetsAll(t *testing.T) {
	require.Equal(t, All, ParseRole("all"))
}

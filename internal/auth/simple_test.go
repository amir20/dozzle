package auth

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-chi/jwtauth/v5"
	"github.com/stretchr/testify/require"
)

// testSecret stands in for the persisted secret from SessionSecret. It is fixed
// so the "stable across restarts" tests below still compare two contexts built
// from the same inputs.
var testSecret = []byte("test-session-secret")

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

	_, token, err := NewSimpleAuth(users, 0, testSecret).tokenAuth.Encode(map[string]any{"username": "alice"})
	require.NoError(t, err)

	// Each iteration stands in for a Dozzle restart against an unchanged users.yml.
	for i := range 50 {
		if _, err := NewSimpleAuth(users, 0, testSecret).tokenAuth.Decode(token); err != nil {
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

	_, token, err := NewSimpleAuth(users, 0, testSecret).tokenAuth.Encode(map[string]any{"username": "alice"})
	require.NoError(t, err)

	users.Users["bob"].RolesConfigured = "shell,actions"
	_, err = NewSimpleAuth(users, 0, testSecret).tokenAuth.Decode(token)
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

	a := NewSimpleAuth(users, 0, testSecret)

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

	a := NewSimpleAuth(users, 0, testSecret)
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

	a := NewSimpleAuth(users, 0, testSecret)
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

	a := NewSimpleAuth(users, 0, testSecret)
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

	require.Nil(t, serveWithAuth(t, NewSimpleAuth(users, 0, testSecret), ""))
}

// The container filter comes from users.yml as well, parsed into labels when the
// file is read so a token cannot pin a stale one.
func TestSimpleAuthResolvesFilterFromDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "users.yml")
	require.NoError(t, os.WriteFile(path, []byte(`
users:
  alice:
    name: Alice
    password: "$2y$11$pTGu6dnTT7Uh3ob7uC6X7OkAamlhHpJ0/mEbsmiPyO85pumillZme"
    filter: "name=foo"
`), 0o600))

	users, err := ReadUsersFromFile(path)
	require.NoError(t, err)

	a := NewSimpleAuth(users, 0, testSecret)
	_, token, err := a.tokenAuth.Encode(map[string]any{"username": "alice", "filter": "name=stale"})
	require.NoError(t, err)

	user := serveWithAuth(t, a, token)
	require.NotNil(t, user)
	require.Equal(t, "name=foo", user.Filter)
	require.Equal(t, []string{"foo"}, user.ContainerLabels["name"])
	require.Equal(t, All, user.Roles, "no roles key in users.yml means everything")
	require.Empty(t, user.Password, "the resolved user must not carry the password hash")
}

// An unparseable filter fails the read instead of silently locking the user out
// of every container on their next request.
func TestReadUsersRejectsInvalidFilter(t *testing.T) {
	path := filepath.Join(t.TempDir(), "users.yml")
	require.NoError(t, os.WriteFile(path, []byte(`
users:
  alice:
    password: "$2y$11$pTGu6dnTT7Uh3ob7uC6X7OkAamlhHpJ0/mEbsmiPyO85pumillZme"
    filter: "nope"
`), 0o600))

	_, err := ReadUsersFromFile(path)
	require.Error(t, err)
}

// serveWithAuthResponse is serveWithAuth for the tests that care about what the
// middleware wrote back rather than which user it resolved.
func serveWithAuthResponse(t *testing.T, a *simpleAuthContext, token string) *http.Response {
	t.Helper()

	handler := a.AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	return recorder.Result()
}

func aliceDatabase() UserDatabase {
	return UserDatabase{
		Users: map[string]*User{
			"alice": {Username: "alice", Password: "$2a$11$aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", RolesConfigured: "all", Roles: All},
		},
	}
}

// tokenExpiringIn mints a session for alice that runs out at a chosen moment, so
// a test can place it anywhere in its life without waiting.
func tokenExpiringIn(t *testing.T, a *simpleAuthContext, remaining time.Duration) string {
	t.Helper()

	claims := map[string]any{"username": "alice"}
	jwtauth.SetIssuedNow(claims)
	jwtauth.SetExpiry(claims, time.Now().Add(remaining))
	_, token, err := a.tokenAuth.Encode(claims)
	require.NoError(t, err)

	return token
}

// The token minted at login used to be the only one there ever was, so a session
// ended exactly ttl after the password was typed however much it was used. An
// iOS home screen app is the case that made this visible: it is resumed, never
// reloaded, so nothing ever went back through the login page to get a new one.
func TestSimpleAuthSlidesSessionPastHalfLife(t *testing.T) {
	a := NewSimpleAuth(aliceDatabase(), time.Hour, testSecret)

	res := serveWithAuthResponse(t, a, tokenExpiringIn(t, a, 20*time.Minute))

	cookie := sessionCookie(res)
	require.NotNil(t, cookie, "a session past half its life must be renewed")

	token, err := a.tokenAuth.Decode(cookie.Value)
	require.NoError(t, err)
	expiry, ok := token.Expiration()
	require.True(t, ok)
	require.WithinDuration(t, time.Now().Add(time.Hour), expiry, time.Minute)
	require.WithinDuration(t, time.Now().Add(time.Hour), cookie.Expires, time.Minute)
}

// Renewing on every request would put a Set-Cookie on every SSE response and
// every poll, and re-sign a token for nothing.
func TestSimpleAuthLeavesAFreshSessionAlone(t *testing.T) {
	a := NewSimpleAuth(aliceDatabase(), time.Hour, testSecret)

	res := serveWithAuthResponse(t, a, tokenExpiringIn(t, a, 50*time.Minute))

	require.Nil(t, sessionCookie(res))
}

// --auth-ttl=session mints a token with no exp and a cookie the browser expires
// on its own. There is nothing to slide, and writing a cookie with an Expires
// would turn it into a persistent one, which is the opposite of what was asked.
func TestSimpleAuthDoesNotSlideASessionCookie(t *testing.T) {
	a := NewSimpleAuth(aliceDatabase(), 0, testSecret)

	_, token, err := a.tokenAuth.Encode(map[string]any{"username": "alice"})
	require.NoError(t, err)

	require.Nil(t, sessionCookie(serveWithAuthResponse(t, a, token)))
}

// An expired token resolves to no user at all, so there is nothing to renew:
// renewing here would let a session be extended forever from a dead one.
func TestSimpleAuthDoesNotSlideAnExpiredSession(t *testing.T) {
	a := NewSimpleAuth(aliceDatabase(), time.Hour, testSecret)

	res := serveWithAuthResponse(t, a, tokenExpiringIn(t, a, -time.Minute))

	require.Nil(t, sessionCookie(res))
}

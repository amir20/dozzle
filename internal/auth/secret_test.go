package auth

import (
	"crypto/sha256"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-chi/jwtauth/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSessionSecretPersistsAcrossRestarts(t *testing.T) {
	dir := t.TempDir()

	first := SessionSecret(dir)
	require.Len(t, first, sessionSecretSize)

	assert.Equal(t, first, SessionSecret(dir), "a second start should reuse the persisted secret")
}

func TestSessionSecretIsUniquePerInstall(t *testing.T) {
	assert.NotEqual(t, SessionSecret(t.TempDir()), SessionSecret(t.TempDir()))
}

func TestSessionSecretFileIsNotWorldReadable(t *testing.T) {
	dir := t.TempDir()
	SessionSecret(dir)

	info, err := os.Stat(filepath.Join(dir, SessionSecretFile))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

// A truncated file would otherwise be decoded into a near-empty key, which is
// the failure the persisted secret exists to prevent.
func TestSessionSecretRejectsATruncatedFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, SessionSecretFile)
	require.NoError(t, os.WriteFile(path, []byte("dHJ1bmNhdGVk\n"), 0600))

	_, err := readSessionSecret(path)
	assert.Error(t, err)
}

// An unwritable directory is a real deployment. Booting has to keep working, and
// the secret still has to be random rather than fall back to the user digest.
func TestSessionSecretFallsBackToAnEphemeralSecret(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "read-only")
	require.NoError(t, os.Mkdir(dir, 0500))

	secret := SessionSecret(dir)
	require.Len(t, secret, sessionSecretSize)
	assert.NotEqual(t, secret, SessionSecret(dir), "an unpersisted secret is regenerated per start")
}

// The signing key used to be a digest over users.yml alone. Every input to it is
// public once an account is OAuth-only and carries no bcrypt hash, so anyone who
// could guess a user's email and GitHub login could derive the key and mint a
// session for them. See GHSA and internal/auth/secret.go.
func TestSigningKeyIsNotDerivableFromUsersFile(t *testing.T) {
	yml := `
users:
  admin:
    email: admin@example.com
    name: Admin
    github: someadmin
`
	path := filepath.Join(t.TempDir(), "users.yml")
	require.NoError(t, os.WriteFile(path, []byte(yml), 0600))

	db, err := ReadUsersFromFile(path)
	require.NoError(t, err)

	a := NewSimpleAuth(db, 0, SessionSecret(t.TempDir()))

	// Everything an attacker can see: no password, the default role, the email
	// and the GitHub login straight out of users.yml.
	h := sha256.New()
	h.Write([]byte(""))
	h.Write([]byte("all"))
	h.Write([]byte("admin@example.com"))
	h.Write([]byte("someadmin"))

	_, forged, err := jwtauth.New("HS256", h.Sum(nil), nil).Encode(map[string]any{"username": "admin"})
	require.NoError(t, err)

	var resolved *User
	handler := a.AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resolved = UserFromContext(r.Context())
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: "jwt", Value: forged})
	handler.ServeHTTP(httptest.NewRecorder(), req)

	assert.Nil(t, resolved, "a token signed with a key guessed from users.yml must not authenticate")
}

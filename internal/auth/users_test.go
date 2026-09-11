package auth

import (
	"os"
	"path/filepath"
	"testing"
	"time"

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

func writeUsers(t *testing.T, contents string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "users.yml")
	require.NoError(t, os.WriteFile(path, []byte(contents), 0o600))

	return path
}

// A password is optional once an account is linked to an external provider.
func TestUserWithoutPasswordParses(t *testing.T) {
	db, err := ReadUsersFromFile(writeUsers(t, `
users:
  amir:
    email: amir@example.com
    name: Amir
    github: amir20
`))
	require.NoError(t, err)

	user := db.Find("amir")
	require.NotNil(t, user)
	require.Empty(t, user.Password)
	require.Equal(t, "amir20", user.Github)
	require.Equal(t, All, user.Roles, "an entry with no roles key still gets all")
}

// An entry with no way at all to sign in is a typo, not a configuration.
func TestUserWithNoCredentialIsRejected(t *testing.T) {
	_, err := ReadUsersFromFile(writeUsers(t, `
users:
  ghost:
    name: Ghost
`))
	require.ErrorContains(t, err, "ghost")
	require.ErrorContains(t, err, "can never sign in")
}

func TestInvalidPasswordHashIsRejected(t *testing.T) {
	_, err := ReadUsersFromFile(writeUsers(t, `
users:
  amir:
    email: amir@example.com
    password: too-short
`))
	require.ErrorContains(t, err, "invalid password hash")
}

// Two users sharing a linked account would resolve to whichever one the map
// happened to hold, so it has to fail at load rather than authenticate the
// wrong user later.
func TestDuplicateGithubLoginIsRejected(t *testing.T) {
	_, err := ReadUsersFromFile(writeUsers(t, `
users:
  amir:
    github: amir20
  other:
    github: AMIR20
`))
	require.ErrorContains(t, err, "share the github login")
}

func TestDuplicateEmailIsRejected(t *testing.T) {
	_, err := ReadUsersFromFile(writeUsers(t, `
users:
  amir:
    email: amir@example.com
  other:
    email: AMIR@Example.com
`))
	require.ErrorContains(t, err, "share the email")
}

func TestLookupsAreCaseInsensitive(t *testing.T) {
	db, err := ReadUsersFromFile(writeUsers(t, `
users:
  amir:
    email: Amir@Example.com
    github: Amir20
`))
	require.NoError(t, err)

	for _, login := range []string{"Amir20", "amir20", "AMIR20"} {
		user := db.FindByGithub(login)
		require.NotNil(t, user, "github login %q should match", login)
		require.Equal(t, "amir", user.Username)
	}

	require.NotNil(t, db.FindByEmail("amir@example.com"))
	require.Nil(t, db.FindByGithub(""), "an empty login must never match")
	require.Nil(t, db.FindByEmail(""))
	require.Nil(t, db.FindByGithub("nobody"))
}

// readFileIfChanged swaps the maps wholesale, so the indexes have to be rebuilt
// with Users or a lookup would keep resolving against the pre-reload file.
func TestIndexesSurviveReload(t *testing.T) {
	path := writeUsers(t, `
users:
  amir:
    github: amir20
`)
	db, err := ReadUsersFromFile(path)
	require.NoError(t, err)
	require.NotNil(t, db.FindByGithub("amir20"))

	// LastRead is compared against mtime, so the rewrite has to look newer.
	db.LastRead = time.Now().Add(-time.Hour)
	require.NoError(t, os.WriteFile(path, []byte(`
users:
  amir:
    github: renamed
`), 0o600))

	require.Nil(t, db.FindByGithub("amir20"), "the old link must not survive the reload")
	require.NotNil(t, db.FindByGithub("renamed"))
}

// A users.yml that stops parsing fails closed: every lookup resolves to nobody
// until it is fixed, rather than authenticating against a stale copy. Making the
// new password/github validation return an error instead of log.Fatal is what
// keeps a bad edit from taking the whole process down mid-reload.
func TestReloadFailureFailsClosed(t *testing.T) {
	path := writeUsers(t, `
users:
  amir:
    github: amir20
`)
	db, err := ReadUsersFromFile(path)
	require.NoError(t, err)
	require.NotNil(t, db.Find("amir"))

	db.LastRead = time.Now().Add(-time.Hour)
	require.NoError(t, os.WriteFile(path, []byte("users:\n  ghost:\n    name: Ghost\n"), 0o600))

	require.Nil(t, db.Find("amir"))
	require.Nil(t, db.FindByGithub("amir20"))
}

// Changing a linked account has to rotate the signing key, or sessions minted
// under the old link outlive it.
func TestSigningKeyCoversLinkedAccounts(t *testing.T) {
	tokenFor := func(github, email string) string {
		user := &User{Username: "amir", Email: email, Github: github, RolesConfigured: "all"}
		db := UserDatabase{Users: map[string]*User{"amir": user}}
		token, err := NewSimpleAuth(db, 0, testSecret).issueToken(*user)
		require.NoError(t, err)
		return token
	}

	base := tokenFor("amir20", "amir@example.com")
	require.NotEqual(t, base, tokenFor("someone-else", "amir@example.com"), "changing github must rotate sessions")
	require.NotEqual(t, base, tokenFor("amir20", "other@example.com"), "changing email must rotate sessions")
}

// A sha256 hash used to be a supported format, and the length check used to
// accept one long after CompareHashAndPassword stopped comparing it. That
// combination loads clean and then refuses at the first login attempt, which is
// an unauthenticated request. Reject it at load instead.
func TestSha256PasswordHashIsRejected(t *testing.T) {
	_, err := ReadUsersFromFile(writeUsers(t, `
users:
  amir:
    email: amir@example.com
    password: 5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8
`))
	require.ErrorContains(t, err, "amir")
	require.ErrorContains(t, err, "sha256")
	require.ErrorContains(t, err, "dozzle generate", "the error has to say how to fix the file")
	require.ErrorContains(t, err, "GHSA-w7qr-q9fh-fj35")
}

// The comparison runs on an unauthenticated request, so a hash it cannot handle
// has to fail the login rather than take the process down. A users.yml reload
// can also put one in front of this long after startup.
func TestCompareHashAndPasswordRefusesSha256WithoutExiting(t *testing.T) {
	require.False(t, CompareHashAndPassword("5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8", "password"))
	require.False(t, CompareHashAndPassword("too-short", "password"))
	require.True(t, CompareHashAndPassword("$2y$11$pTGu6dnTT7Uh3ob7uC6X7OkAamlhHpJ0/mEbsmiPyO85pumillZme", "password"))
}

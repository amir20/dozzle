package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func envFrom(pairs map[string]string) func(string) string {
	return func(key string) string { return pairs[key] }
}

func secretFile(t *testing.T, contents string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "secret")
	require.NoError(t, os.WriteFile(path, []byte(contents), 0o600))

	return path
}

func TestFileBackedSecretIsRead(t *testing.T) {
	path := secretFile(t, "gh-secret")
	args := Args{}

	require.NoError(t, applyFileBackedSecrets(&args, envFrom(map[string]string{
		"DOZZLE_AUTH_GITHUB_CLIENT_SECRET_FILE": path,
	})))

	require.Equal(t, "gh-secret", args.AuthGithubClientSecret)
}

// `echo secret > file` leaves a trailing newline, and every secret manager does
// something similar. A client secret has no meaningful surrounding whitespace.
func TestFileBackedSecretTrimsWhitespace(t *testing.T) {
	args := Args{}

	require.NoError(t, applyFileBackedSecrets(&args, envFrom(map[string]string{
		"DOZZLE_AUTH_OIDC_CLIENT_SECRET_FILE": secretFile(t, "  oidc-secret\n\n"),
	})))

	require.Equal(t, "oidc-secret", args.AuthOidcClientSecret)
}

func TestFileBackedSecretsAreIndependent(t *testing.T) {
	args := Args{}

	require.NoError(t, applyFileBackedSecrets(&args, envFrom(map[string]string{
		"DOZZLE_AUTH_GITHUB_CLIENT_SECRET_FILE": secretFile(t, "gh"),
		"DOZZLE_AUTH_OIDC_CLIENT_SECRET_FILE":   secretFile(t, "oidc"),
	})))

	require.Equal(t, "gh", args.AuthGithubClientSecret)
	require.Equal(t, "oidc", args.AuthOidcClientSecret)
}

// Silently preferring one would leave an operator staring at a value they
// thought they had overridden.
func TestFileBackedSecretConflictsWithInlineValue(t *testing.T) {
	args := Args{AuthGithubClientSecret: "inline"}

	err := applyFileBackedSecrets(&args, envFrom(map[string]string{
		"DOZZLE_AUTH_GITHUB_CLIENT_SECRET_FILE": secretFile(t, "from-file"),
	}))

	require.ErrorContains(t, err, "both set")
	require.Equal(t, "inline", args.AuthGithubClientSecret, "the inline value must not be clobbered")
}

func TestFileBackedSecretMissingFileIsAnError(t *testing.T) {
	err := applyFileBackedSecrets(&Args{}, envFrom(map[string]string{
		"DOZZLE_AUTH_GITHUB_CLIENT_SECRET_FILE": "/nope/does-not-exist",
	}))

	require.ErrorContains(t, err, "could not read DOZZLE_AUTH_GITHUB_CLIENT_SECRET_FILE")
}

// An empty secret file is a mounting mistake, not a request to disable OAuth.
func TestFileBackedSecretEmptyFileIsAnError(t *testing.T) {
	err := applyFileBackedSecrets(&Args{}, envFrom(map[string]string{
		"DOZZLE_AUTH_OIDC_CLIENT_SECRET_FILE": secretFile(t, "\n  \n"),
	}))

	require.ErrorContains(t, err, "is empty")
}

func TestNoFileEnvLeavesArgsAlone(t *testing.T) {
	args := Args{AuthGithubClientSecret: "inline"}

	require.NoError(t, applyFileBackedSecrets(&args, envFrom(nil)))
	require.Equal(t, "inline", args.AuthGithubClientSecret)
	require.Empty(t, args.AuthOidcClientSecret)
}

// ValidateEnvVars warns on unrecognized DOZZLE_ vars, so every _FILE name it
// should accept has to come from the same list the resolver uses.
func TestFileBackedEnvNamesCoverEverySecret(t *testing.T) {
	names := fileBackedEnvNames()

	require.ElementsMatch(t, []string{
		"DOZZLE_AUTH_GITHUB_CLIENT_SECRET",
		"DOZZLE_AUTH_OIDC_CLIENT_SECRET",
	}, names)

	// Every name must correspond to a real arg field.
	args := Args{}
	secrets := fileBackedSecrets(&args)
	for _, name := range names {
		require.NotNil(t, secrets[name], "%s has no field", name)
	}
}

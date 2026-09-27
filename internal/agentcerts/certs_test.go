package agentcerts

import (
	"crypto/x509"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateMakesAUsablePair(t *testing.T) {
	pair, err := Generate()
	require.NoError(t, err)
	cert, err := pair.TLS()
	require.NoError(t, err)

	leaf, err := x509.ParseCertificate(cert.Certificate[0])
	require.NoError(t, err)
	assert.True(t, leaf.IsCA)
	assert.Contains(t, leaf.ExtKeyUsage, x509.ExtKeyUsageClientAuth)
	assert.Contains(t, leaf.ExtKeyUsage, x509.ExtKeyUsageServerAuth)
}

func TestLoadOrCreateAgentPairIsStable(t *testing.T) {
	dir := t.TempDir()

	_, err := LoadAgentPair(dir)
	assert.ErrorIs(t, err, os.ErrNotExist)

	first, err := LoadOrCreateAgentPair(dir)
	require.NoError(t, err)
	second, err := LoadOrCreateAgentPair(dir)
	require.NoError(t, err)
	assert.Equal(t, first.Cert, second.Cert, "a made pair is never replaced")
	assert.False(t, second.NotAfter.IsZero())

	info, err := os.Stat(filepath.Join(dir, agentKeyFile))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

func TestFromEnv(t *testing.T) {
	pair, err := Generate()
	require.NoError(t, err)
	want, err := pair.TLS()
	require.NoError(t, err)

	lookup := func(values map[string]string) func(string) (string, bool) {
		return func(k string) (string, bool) { v, ok := values[k]; return v, ok }
	}

	_, ok, err := FromEnv(lookup(nil))
	assert.False(t, ok)
	assert.NoError(t, err)

	got, ok, err := FromEnv(lookup(map[string]string{"DOZZLE_CERT_PEM": string(pair.Cert), "DOZZLE_KEY_PEM": string(pair.Key)}))
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, want.Certificate, got.Certificate)

	escaped := func(b []byte) string { return strings.ReplaceAll(string(b), "\n", `\n`) }
	got, _, err = FromEnv(lookup(map[string]string{"DOZZLE_CERT_PEM": escaped(pair.Cert), "DOZZLE_KEY_PEM": escaped(pair.Key)}))
	require.NoError(t, err)
	assert.Equal(t, want.Certificate, got.Certificate)

	_, ok, err = FromEnv(lookup(map[string]string{"DOZZLE_CERT_PEM": string(pair.Cert)}))
	assert.True(t, ok)
	assert.Error(t, err)

}

// A pair with one half missing is broken, not absent: agents may hold it, so it
// must never be quietly replaced by a fresh one.
func TestLoadOrCreateAgentPairRefusesHalfAPair(t *testing.T) {
	for _, missing := range []string{agentCertFile, agentKeyFile} {
		dir := t.TempDir()
		original, err := LoadOrCreateAgentPair(dir)
		require.NoError(t, err)
		require.NoError(t, os.Remove(filepath.Join(dir, missing)))

		_, err = LoadOrCreateAgentPair(dir)
		assert.Error(t, err, missing)

		// The surviving half is untouched.
		for _, name := range []string{agentCertFile, agentKeyFile} {
			if name == missing {
				continue
			}
			got, err := os.ReadFile(filepath.Join(dir, name))
			require.NoError(t, err)
			want := original.Cert
			if name == agentKeyFile {
				want = original.Key
			}
			assert.Equal(t, want, got)
		}
	}
}

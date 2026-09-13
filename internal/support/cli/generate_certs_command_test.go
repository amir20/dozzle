package cli

import (
	"crypto/tls"
	"crypto/x509"
	"embed"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func generatePair(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	cmd := &GenerateCertsCmd{
		CertOut: filepath.Join(dir, "dozzle_cert.pem"),
		KeyOut:  filepath.Join(dir, "dozzle_key.pem"),
	}
	require.NoError(t, cmd.Run(Args{NoAnalytics: true}, embed.FS{}))
	return cmd.CertOut, cmd.KeyOut
}

// The generated pair is presented by both ends and trusted as the CA by both
// ends, which is how internal/agent wires its mTLS. A certificate that is fine
// as a leaf but rejected as a CA would leave the agent refusing every hub.
func TestGenerateCerts_handshakesAsBothSides(t *testing.T) {
	certPath, keyPath := generatePair(t)

	pair, err := tls.LoadX509KeyPair(certPath, keyPath)
	require.NoError(t, err)

	leaf, err := x509.ParseCertificate(pair.Certificate[0])
	require.NoError(t, err)
	pool := x509.NewCertPool()
	pool.AddCert(leaf)

	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	errs := make(chan error, 1)
	go func() {
		server := tls.Server(serverConn, &tls.Config{
			Certificates: []tls.Certificate{pair},
			ClientCAs:    pool,
			ClientAuth:   tls.RequireAndVerifyClientCert,
		})
		errs <- server.Handshake()
	}()

	client := tls.Client(clientConn, &tls.Config{
		Certificates:       []tls.Certificate{pair},
		RootCAs:            pool,
		InsecureSkipVerify: true,
	})
	require.NoError(t, client.Handshake())
	require.NoError(t, <-errs)
}

// Two runs must not produce the same key, otherwise the command would recreate
// the very problem it exists to solve.
func TestGenerateCerts_isUniquePerRun(t *testing.T) {
	firstCert, firstKey := generatePair(t)
	secondCert, secondKey := generatePair(t)

	assert.NotEqual(t, mustRead(t, firstKey), mustRead(t, secondKey))
	assert.NotEqual(t, mustRead(t, firstCert), mustRead(t, secondCert))
}

func TestGenerateCerts_keyIsNotWorldReadable(t *testing.T) {
	_, keyPath := generatePair(t)

	info, err := os.Stat(keyPath)
	require.NoError(t, err)
	assert.Equal(t, fs.FileMode(0600), info.Mode().Perm())
}

func TestGenerateCerts_refusesToOverwriteWithoutForce(t *testing.T) {
	certPath, keyPath := generatePair(t)
	original := mustRead(t, keyPath)

	cmd := &GenerateCertsCmd{CertOut: certPath, KeyOut: keyPath}
	err := cmd.Run(Args{NoAnalytics: true}, embed.FS{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--force")
	assert.Equal(t, original, mustRead(t, keyPath))

	cmd.Force = true
	require.NoError(t, cmd.Run(Args{NoAnalytics: true}, embed.FS{}))
	assert.NotEqual(t, original, mustRead(t, keyPath))
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return data
}

// Package agentcerts makes the certificate pairs hubs and agents authenticate each
// other with. Each end presents the pair and trusts only its certificate, so a
// pair nobody else holds is what turns the agent's mTLS into a credential.
package agentcerts

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Pair is a PEM certificate and its PEM private key.
type Pair struct {
	Cert     []byte
	Key      []byte
	NotAfter time.Time
}

// Generate makes a fresh self-signed pair, valid for five years.
func Generate() (Pair, error) {
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return Pair{}, fmt.Errorf("failed to generate key: %w", err)
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return Pair{}, fmt.Errorf("failed to generate serial number: %w", err)
	}

	now := time.Now()
	template := x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{Organization: []string{"Dozzle"}},
		NotBefore:    now.Add(-time.Hour),
		NotAfter:     now.AddDate(5, 0, 0),
		// Each end presents this certificate and also trusts it as the CA, so it
		// has to be valid as a client, a server and a signer all at once.
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	der, err := x509.CreateCertificate(rand.Reader, &template, &template, public, private)
	if err != nil {
		return Pair{}, fmt.Errorf("failed to create certificate: %w", err)
	}
	pkcs8, err := x509.MarshalPKCS8PrivateKey(private)
	if err != nil {
		return Pair{}, fmt.Errorf("failed to marshal key: %w", err)
	}

	return Pair{
		Cert:     pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
		Key:      pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: pkcs8}),
		NotAfter: template.NotAfter,
	}, nil
}

// TLS parses the pair for use in a tls.Config.
func (p Pair) TLS() (tls.Certificate, error) {
	return tls.X509KeyPair(p.Cert, p.Key)
}

// The hub's private pair for agents added from the UI. It lives beside
// dozzle.yml, apart from --cert/--key, so creating it never changes what the
// hub presents to agents that already exist.
const (
	agentCertFile = "agent_cert.pem"
	agentKeyFile  = "agent_key.pem"
)

var mu sync.Mutex

// LoadAgentPair reads the hub's private agent pair from dir. It returns an
// error wrapping os.ErrNotExist when none has been made yet.
func LoadAgentPair(dir string) (Pair, error) {
	mu.Lock()
	defer mu.Unlock()
	return loadAgentPair(dir)
}

func loadAgentPair(dir string) (Pair, error) {
	cert, err := os.ReadFile(filepath.Join(dir, agentCertFile))
	if err != nil {
		return Pair{}, err
	}
	key, err := os.ReadFile(filepath.Join(dir, agentKeyFile))
	if err != nil {
		return Pair{}, err
	}
	pair := Pair{Cert: cert, Key: key}
	parsed, err := pair.TLS()
	if err != nil {
		return Pair{}, fmt.Errorf("agent pair in %s is unreadable: %w", dir, err)
	}
	if leaf, err := x509.ParseCertificate(parsed.Certificate[0]); err == nil {
		pair.NotAfter = leaf.NotAfter
	}
	return pair, nil
}

// LoadOrCreateAgentPair returns the hub's private agent pair, making it the
// first time it is asked for. It is never replaced once made: every agent
// holding it would stop accepting this hub.
func LoadOrCreateAgentPair(dir string) (Pair, error) {
	mu.Lock()
	defer mu.Unlock()

	pair, err := loadAgentPair(dir)
	if err == nil || !errors.Is(err, os.ErrNotExist) {
		return pair, err
	}
	// Only a dir with neither file gets a new pair. With one half left, agents may
	// still hold the old pair, and replacing it would lock every one of them out.
	for _, name := range []string{agentCertFile, agentKeyFile} {
		if _, statErr := os.Stat(filepath.Join(dir, name)); statErr == nil {
			return Pair{}, fmt.Errorf("%s exists without its other half in %s, restore it or remove both to make a new pair", name, dir)
		}
	}

	pair, err = Generate()
	if err != nil {
		return Pair{}, err
	}
	// Key first, and removed again if the cert cannot follow: a lone half is
	// refused above, so leaving it would fail every later try.
	keyPath := filepath.Join(dir, agentKeyFile)
	if err := writeFile(keyPath, pair.Key, 0600); err != nil {
		return Pair{}, err
	}
	if err := writeFile(filepath.Join(dir, agentCertFile), pair.Cert, 0644); err != nil {
		_ = os.Remove(keyPath)
		return Pair{}, err
	}
	return pair, nil
}

func writeFile(path string, data []byte, perm os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".dozzle-cert-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if err := tmp.Chmod(perm); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// The env vars FromEnv reads. They are not go-arg flags, so the check for
// unknown DOZZLE_* variables has to be told about them.
const (
	CertEnv = "DOZZLE_CERT_PEM"
	KeyEnv  = "DOZZLE_KEY_PEM"
)

// FromEnv parses a pair handed over in DOZZLE_CERT_PEM and DOZZLE_KEY_PEM,
// which is how a snippet copied from the hub gives an agent the pair without
// any files to copy. A literal \n is accepted for shells that cannot pass a
// multi-line value. ok is false when neither is set.
func FromEnv(lookup func(string) (string, bool)) (cert tls.Certificate, ok bool, err error) {
	certPEM, hasCert := lookup(CertEnv)
	keyPEM, hasKey := lookup(KeyEnv)
	if !hasCert && !hasKey {
		return tls.Certificate{}, false, nil
	}
	if !hasCert || !hasKey {
		return tls.Certificate{}, true, errors.New("DOZZLE_CERT_PEM and DOZZLE_KEY_PEM must be set together")
	}
	cert, err = tls.X509KeyPair([]byte(unescape(certPEM)), []byte(unescape(keyPEM)))
	return cert, true, err
}

func unescape(value string) string {
	if strings.Contains(value, "\n") {
		return value
	}
	return strings.ReplaceAll(value, `\n`, "\n")
}

package cli

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"embed"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"time"
)

type GenerateCertsCmd struct {
	CertOut string `arg:"--cert-out" default:"dozzle_cert.pem" help:"path to write the certificate to"`
	KeyOut  string `arg:"--key-out" default:"dozzle_key.pem" help:"path to write the private key to"`
	Force   bool   `arg:"--force" help:"overwrites existing files"`
}

// Run writes a fresh self-signed certificate and key for agent connections.
//
// The certificate Dozzle ships with is embedded in a public image, so every
// install has the same one and it authenticates nobody. This command exists so
// that generating a unique pair, which is the only thing that makes the agent's
// mTLS an actual credential, is one command instead of an openssl recipe.
func (g *GenerateCertsCmd) Run(args Args, embeddedCerts embed.FS) error {
	StartEvent(args, "", nil, "generate-certs")

	if !g.Force {
		for _, path := range []string{g.CertOut, g.KeyOut} {
			if _, err := os.Stat(path); err == nil {
				return fmt.Errorf("%s already exists, use --force to overwrite", path)
			}
		}
	}

	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate key: %w", err)
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return fmt.Errorf("failed to generate serial number: %w", err)
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
		return fmt.Errorf("failed to create certificate: %w", err)
	}

	pkcs8, err := x509.MarshalPKCS8PrivateKey(private)
	if err != nil {
		return fmt.Errorf("failed to marshal key: %w", err)
	}

	if err := writePem(g.KeyOut, "PRIVATE KEY", pkcs8, 0600); err != nil {
		return err
	}
	if err := writePem(g.CertOut, "CERTIFICATE", der, 0644); err != nil {
		return err
	}

	fmt.Printf(`Wrote %s and %s, valid until %s.

Copy both files to the hub and to every agent, then point Dozzle at them:

  dozzle --cert %s --key %s
  dozzle --cert %s --key %s agent

Or with compose, mount them at /dozzle_cert.pem and /dozzle_key.pem, which is
where Dozzle looks by default. Keep %s secret: anyone holding it can connect to
your agents. An agent only accepts hubs presenting the same pair, so replacing
it means restarting the hub and the agents together.
`, g.CertOut, g.KeyOut, template.NotAfter.Format("2006-01-02"), g.CertOut, g.KeyOut, g.CertOut, g.KeyOut, g.KeyOut)

	return nil
}

func writePem(path, blockType string, der []byte, perm os.FileMode) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", path, err)
	}
	if err := pem.Encode(file, &pem.Block{Type: blockType, Bytes: der}); err != nil {
		file.Close()
		return fmt.Errorf("failed to write %s: %w", path, err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}
	return nil
}

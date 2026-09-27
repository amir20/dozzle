package cli

import (
	"embed"
	"fmt"
	"os"

	"github.com/amir20/dozzle/internal/agentcerts"
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

	pair, err := agentcerts.Generate()
	if err != nil {
		return err
	}
	if err := writeFile(g.KeyOut, pair.Key, 0600); err != nil {
		return err
	}
	if err := writeFile(g.CertOut, pair.Cert, 0644); err != nil {
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
`, g.CertOut, g.KeyOut, pair.NotAfter.Format("2006-01-02"), g.CertOut, g.KeyOut, g.CertOut, g.KeyOut, g.KeyOut)

	return nil
}

func writeFile(path string, data []byte, perm os.FileMode) error {
	if err := os.WriteFile(path, data, perm); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}
	return nil
}

package cli

import (
	"crypto/tls"
	"embed"
	"os"

	"github.com/rs/zerolog/log"
)

func ReadCertificates(certs embed.FS) (tls.Certificate, error) {
	// Try multiple certificate paths in order of preference
	certPaths := []struct {
		cert string
		key  string
	}{
		{"dozzle_cert.pem", "dozzle_key.pem"},
		{"/dozzle-cert.pem", "/dozzle-key.pem"},
		{"/certs/dozzle-cert.pem", "/certs/dozzle-key.pem"},
	}

	for _, paths := range certPaths {
		if pair, err := tls.LoadX509KeyPair(paths.cert, paths.key); err == nil {
			log.Info().Str("cert", paths.cert).Str("key", paths.key).Msg("Loaded custom dozzle certificate and key")
			return pair, nil
		} else if !os.IsNotExist(err) {
			log.Fatal().Err(err).Str("cert", paths.cert).Str("key", paths.key).Msg("Failed to load custom dozzle certificate and key. Stopping...")
		}
	}

	cert, err := certs.ReadFile("shared_cert.pem")
	if err != nil {
		return tls.Certificate{}, err
	}

	key, err := certs.ReadFile("shared_key.pem")
	if err != nil {
		return tls.Certificate{}, err
	}

	return tls.X509KeyPair(cert, key)
}

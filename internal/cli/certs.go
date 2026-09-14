package cli

import (
	"crypto/tls"
	"embed"
	"os"

	"github.com/rs/zerolog/log"
)

func ReadCertificates(certs embed.FS, certPath, keyPath string) (tls.Certificate, error) {
	if pair, err := tls.LoadX509KeyPair(certPath, keyPath); err == nil {
		log.Info().Str("cert", certPath).Str("key", keyPath).Msg("Loaded custom dozzle certificate and key")
		return pair, nil
	} else {
		if !os.IsNotExist(err) {
			log.Fatal().Err(err).Str("cert", certPath).Str("key", keyPath).Msg("Failed to load custom dozzle certificate and key. Stopping...")
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

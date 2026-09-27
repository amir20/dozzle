package cli

import (
	"crypto/tls"
	"embed"
	"os"

	"github.com/amir20/dozzle/internal/agentcerts"
	"github.com/rs/zerolog/log"
)

// ReadCertificates picks the pair this process presents to agents or hubs: one
// handed over in DOZZLE_CERT_PEM and DOZZLE_KEY_PEM, then one on disk, then the
// one built into the image.
func ReadCertificates(embedded embed.FS, certPath, keyPath string) (tls.Certificate, error) {
	if pair, ok, err := agentcerts.FromEnv(os.LookupEnv); ok {
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to load certificate from DOZZLE_CERT_PEM and DOZZLE_KEY_PEM. Stopping...")
		}
		log.Info().Msg("Loaded dozzle certificate and key from DOZZLE_CERT_PEM")
		return pair, nil
	}

	if pair, err := tls.LoadX509KeyPair(certPath, keyPath); err == nil {
		log.Info().Str("cert", certPath).Str("key", keyPath).Msg("Loaded custom dozzle certificate and key")
		return pair, nil
	} else {
		if !os.IsNotExist(err) {
			log.Fatal().Err(err).Str("cert", certPath).Str("key", keyPath).Msg("Failed to load custom dozzle certificate and key. Stopping...")
		}
	}

	cert, err := embedded.ReadFile("shared_cert.pem")
	if err != nil {
		return tls.Certificate{}, err
	}

	key, err := embedded.ReadFile("shared_key.pem")
	if err != nil {
		return tls.Certificate{}, err
	}

	return tls.X509KeyPair(cert, key)
}

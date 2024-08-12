package cli

import (
	"crypto/tls"
	"embed"

	log "github.com/sirupsen/logrus"
)

func ReadCertificates(certs embed.FS) (tls.Certificate, error) {
	if pair, err := tls.LoadX509KeyPair("dozzle_cert.pem", "dozzle_key.pem"); err == nil {
		log.Infof("using dozzle certificate and key at ./dozzle_cert.pem and ./dozzle_key.pem")
		return pair, nil
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

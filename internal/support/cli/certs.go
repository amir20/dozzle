package cli

import (
	"crypto/tls"
	"embed"
)

func ReadCertificates(certs embed.FS) (tls.Certificate, error) {
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

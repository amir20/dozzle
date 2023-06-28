package docker

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type Host struct {
	Name       string   `json:"name"`
	Host       string   `json:"host"`
	URL        *url.URL `json:"-"`
	CertPath   string   `json:"-"`
	CACertPath string   `json:"-"`
	KeyPath    string   `json:"-"`
	ValidCerts bool     `json:"-"`
}

func ParseConnection(connection string) (Host, error) {
	parts := strings.Split(connection, "|")
	if len(parts) > 2 {
		return Host{}, fmt.Errorf("invalid connection string: %s", connection)
	}

	remoteUrl, err := url.Parse(parts[0])
	if err != nil {
		return Host{}, err
	}

	name := remoteUrl.Hostname()
	if len(parts) == 2 {
		name = parts[1]
	}

	basePath, err := filepath.Abs("./certs")
	if err != nil {
		log.Fatalf("error converting certs path to absolute: %s", err)
	}

	host := remoteUrl.Hostname()
	if _, err := os.Stat(filepath.Join(basePath, host)); !os.IsNotExist(err) {
		basePath = filepath.Join(basePath, host)
	}

	cacertPath := filepath.Join(basePath, "ca.pem")
	certPath := filepath.Join(basePath, "cert.pem")
	keyPath := filepath.Join(basePath, "key.pem")

	hasCerts := true
	if _, err := os.Stat(cacertPath); os.IsNotExist(err) {
		cacertPath = ""
		hasCerts = false
	}

	return Host{
		Name:       name,
		Host:       host,
		URL:        remoteUrl,
		CertPath:   certPath,
		CACertPath: cacertPath,
		KeyPath:    keyPath,
		ValidCerts: hasCerts,
	}, nil

}

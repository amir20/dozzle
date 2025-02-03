package container

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
)

type Host struct {
	Name          string   `json:"name"`
	ID            string   `json:"id"`
	URL           *url.URL `json:"-"`
	CertPath      string   `json:"-"`
	CACertPath    string   `json:"-"`
	KeyPath       string   `json:"-"`
	ValidCerts    bool     `json:"-"`
	NCPU          int      `json:"nCPU"`
	MemTotal      int64    `json:"memTotal"`
	Endpoint      string   `json:"endpoint"`
	DockerVersion string   `json:"dockerVersion"`
	AgentVersion  string   `json:"agentVersion,omitempty"`
	Type          string   `json:"type"`
	Available     bool     `json:"available"`
	Swarm         bool     `json:"-"`
}

func (h Host) String() string {
	return fmt.Sprintf("ID: %s, Endpoint: %s, nCPU: %d, memTotal: %d", h.ID, h.Endpoint, h.NCPU, h.MemTotal)
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
		return Host{}, err
	}

	host := remoteUrl.Hostname()
	if _, err := os.Stat(filepath.Join(basePath, host)); !os.IsNotExist(err) {
		basePath = filepath.Join(basePath, host)
	} else {
		log.Debug().Msgf("Remote host certificate path does not exist %s, falling back to default: %s", filepath.Join(basePath, host), basePath)
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
		ID:         strings.ReplaceAll(remoteUrl.String(), "/", ""),
		Name:       name,
		URL:        remoteUrl,
		CertPath:   certPath,
		CACertPath: cacertPath,
		KeyPath:    keyPath,
		ValidCerts: hasCerts,
		Endpoint:   remoteUrl.String(),
	}, nil

}

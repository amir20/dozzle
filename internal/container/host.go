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
	MemUsed       uint64   `json:"memUsed,omitempty"`
	Load1         float64  `json:"load1,omitempty"`
	Load5         float64  `json:"load5,omitempty"`
	Load15        float64  `json:"load15,omitempty"`
	Uptime        uint64   `json:"uptime,omitempty"`
	DiskTotal     uint64   `json:"diskTotal,omitempty"`
	DiskFree      uint64   `json:"diskFree,omitempty"`
	NetRxTotal    uint64   `json:"netRxTotal,omitempty"`
	NetTxTotal    uint64   `json:"netTxTotal,omitempty"`
	Endpoint      string   `json:"endpoint"`
	DockerVersion string   `json:"dockerVersion"`
	Runtime       string   `json:"runtime,omitempty"`
	AgentVersion  string   `json:"agentVersion,omitempty"`
	Type          string   `json:"type"`
	Available     bool     `json:"available"`
	Swarm         bool     `json:"-"`
	// SwarmClusterID identifies the swarm this node belongs to, empty outside
	// a swarm. Every node of one swarm reports the same value, which is what
	// lets Dozzle Cloud tell "one swarm, N replicas" apart from one API key
	// reused across unrelated deployments.
	SwarmClusterID string `json:"-"`
	Group          string `json:"group,omitempty"`
	// ReplacesID names an id this host was previously known by, set only when
	// the hub re-keys a client whose agent came back with a different id. It is
	// minted by the hub and never crosses the agent boundary, so the UI can drop
	// the stale entry rather than list one machine twice until the next reload.
	ReplacesID string `json:"replacesId,omitempty"`
	// Removed is set on the one update sent when an agent is removed from the UI,
	// so every open tab drops the host, not only the one that removed it.
	Removed bool `json:"removed,omitempty"`
}

func (h Host) String() string {
	return fmt.Sprintf("ID: %s, Endpoint: %s, nCPU: %d, memTotal: %d", h.ID, h.Endpoint, h.NCPU, h.MemTotal)
}

// HostMetrics carries host-level metrics read from /proc and the root filesystem.
// They are only meaningful for the local host (remote hosts get theirs through the agent).
type HostMetrics struct {
	Load1      float64
	Load5      float64
	Load15     float64
	Uptime     uint64
	MemUsed    uint64
	DiskTotal  uint64
	DiskFree   uint64
	NetRxTotal uint64
	NetTxTotal uint64
}

func (h *Host) ApplyHostMetrics(m HostMetrics) {
	h.Load1, h.Load5, h.Load15 = m.Load1, m.Load5, m.Load15
	h.Uptime = m.Uptime
	h.MemUsed = m.MemUsed
	h.DiskTotal, h.DiskFree = m.DiskTotal, m.DiskFree
	h.NetRxTotal, h.NetTxTotal = m.NetRxTotal, m.NetTxTotal
}

func ParseConnection(connection string) (Host, error) {
	parts := strings.Split(connection, "|")
	if len(parts) > 3 || parts[0] == "" {
		return Host{}, fmt.Errorf("invalid connection string: %s", connection)
	}

	remoteUrl, err := url.Parse(parts[0])
	if err != nil {
		return Host{}, err
	}

	name := remoteUrl.Hostname()
	if len(parts) >= 2 && parts[1] != "" {
		name = parts[1]
	}

	group := ""
	if len(parts) == 3 {
		group = parts[2]
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
		Group:      group,
	}, nil

}

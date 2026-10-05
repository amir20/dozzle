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
	Name             string   `json:"name"`
	ID               string   `json:"id"`
	URL              *url.URL `json:"-"`
	CertPath         string   `json:"-"`
	CACertPath       string   `json:"-"`
	KeyPath          string   `json:"-"`
	ValidCerts       bool     `json:"-"`
	NCPU             int      `json:"nCPU"`
	MemTotal         int64    `json:"memTotal"`
	MetricsAvailable bool     `json:"metricsAvailable,omitempty"`
	Load1            float64  `json:"load1,omitempty"`
	Load5            float64  `json:"load5,omitempty"`
	Load15           float64  `json:"load15,omitempty"`
	Uptime           uint64   `json:"uptime,omitempty"`
	DiskTotal        uint64   `json:"diskTotal,omitempty"`
	DiskFree         uint64   `json:"diskFree,omitempty"`
	Disks            []Disk   `json:"disks,omitempty"`
	// Reclaimable is nil until the size monitor has measured it, and always on
	// engines that cannot (k8s).
	Reclaimable   *Reclaimable `json:"reclaimable,omitempty"`
	Endpoint      string       `json:"endpoint"`
	DockerVersion string       `json:"dockerVersion"`
	Runtime       string       `json:"runtime,omitempty"`
	AgentVersion  string       `json:"agentVersion,omitempty"`
	Type          string       `json:"type"`
	Available     bool         `json:"available"`
	Swarm         bool         `json:"-"`
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

// HostMetrics carries host-level metrics read from /proc and the engine's data
// directory. They are only meaningful for the local host (remote hosts get
// theirs through the agent).
type HostMetrics struct {
	Load1     float64
	Load5     float64
	Load15    float64
	Uptime    uint64
	DiskTotal uint64
	DiskFree  uint64
	// Disks are the extra drives an operator opted into by mounting them under
	// /host/disks/<name>; DiskTotal and DiskFree stay Docker's own disk.
	Disks []Disk
	// Reclaimable comes from the engine rather than the machine, but it travels
	// with the rest so an agent sends it on the same reply.
	Reclaimable *Reclaimable
}

// Reclaimable is what `docker system df` counts as reclaimable: space held by
// things no container is using.
type Reclaimable struct {
	Images         int64 `json:"images"`
	ImagesSize     int64 `json:"imagesSize"`
	Volumes        int64 `json:"volumes"`
	VolumesSize    int64 `json:"volumesSize"`
	Containers     int64 `json:"containers"`
	ContainersSize int64 `json:"containersSize"`
	BuildCacheSize int64 `json:"buildCacheSize"`
}

// Disk is one extra drive's usage, named after its folder under /host/disks.
type Disk struct {
	Name  string `json:"name"`
	Total uint64 `json:"total"`
	Free  uint64 `json:"free"`
}

// ApplyHostMetrics copies host-level metrics onto the host. metricsAvailable is
// false when the host /proc could not be read, which leaves load and uptime at
// zero; disk may still be set from the engine's data directory.
func (h *Host) ApplyHostMetrics(m HostMetrics, metricsAvailable bool) {
	h.MetricsAvailable = metricsAvailable
	h.Load1, h.Load5, h.Load15 = m.Load1, m.Load5, m.Load15
	h.Uptime = m.Uptime
	h.DiskTotal, h.DiskFree = m.DiskTotal, m.DiskFree
	h.Disks = m.Disks
	h.Reclaimable = m.Reclaimable
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

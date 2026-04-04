package cloud

import (
	"strings"
	"time"

	"github.com/amir20/dozzle/internal/container"
	pb "github.com/amir20/dozzle/proto/cloud"
	"github.com/rs/zerolog/log"
)

// buildHostNameMap creates a mapping from host ID to host name.
func buildHostNameMap(hostService ToolHostService) map[string]string {
	hosts := hostService.Hosts()
	m := make(map[string]string, len(hosts))
	for _, h := range hosts {
		m[h.ID] = h.Name
	}
	return m
}

// resolveHostName returns the host name for a given host ID, falling back to the ID itself.
func resolveHostName(hostID string, hostNames map[string]string) string {
	if name, ok := hostNames[hostID]; ok {
		return name
	}
	return hostID
}

func containerToProto(c container.Container, hostNames map[string]string) *pb.ContainerInfo {
	return &pb.ContainerInfo{
		Id:         c.ID,
		Name:       c.Name,
		Image:      c.Image,
		Command:    c.Command,
		Created:    c.Created.UTC().Format(time.RFC3339),
		StartedAt:  c.StartedAt.UTC().Format(time.RFC3339),
		FinishedAt: formatTimeOrEmpty(c.FinishedAt),
		State:      c.State,
		Health:     c.Health,
		Host:       resolveHostName(c.Host, hostNames),
		Group:      c.Group,
	}
}

func formatTimeOrEmpty(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func containsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

func logHostErrors(errs []error) {
	for _, err := range errs {
		if err != nil {
			log.Warn().Err(err).Msg("error listing containers from host")
		}
	}
}

package cloud

import (
	"fmt"
	"strings"
	"time"

	"github.com/amir20/dozzle/internal/container"
	container_support "github.com/amir20/dozzle/internal/support/container"
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
		HostName:   resolveHostName(c.Host, hostNames),
		HostId:     c.Host,
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

// findContainerFlexible finds a container by ID, optionally scoped to a specific host.
// When hostID is empty, it searches across all hosts by listing all containers and matching by ID.
// When hostID is provided, it also tries to resolve host names to host IDs for LLM-friendliness.
func findContainerFlexible(hostID string, containerID string, hostService ToolHostService, labels container.ContainerLabels) (*container_support.ContainerService, error) {
	if containerID == "" {
		return nil, fmt.Errorf("container_id is required")
	}

	// If host is provided, try direct lookup first, then try resolving name to ID
	if hostID != "" {
		cs, err := hostService.FindContainer(hostID, containerID, labels)
		if err == nil {
			return cs, nil
		}

		// Try treating hostID as a host name and resolve to actual ID
		for _, h := range hostService.Hosts() {
			if strings.EqualFold(h.Name, hostID) {
				cs, err := hostService.FindContainer(h.ID, containerID, labels)
				if err == nil {
					return cs, nil
				}
			}
		}
	}

	// Search across all hosts by listing all containers
	containers, errs := hostService.ListAllContainers(labels)
	logHostErrors(errs)

	for _, c := range containers {
		if c.ID == containerID {
			return hostService.FindContainer(c.Host, containerID, labels)
		}
	}

	if hostID != "" {
		return nil, fmt.Errorf("container %s not found on host %s", containerID, hostID)
	}
	return nil, fmt.Errorf("container %s not found on any host", containerID)
}

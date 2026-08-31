package web

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/rs/zerolog/log"
)

// stackLabels mirrors the frontend's namespace derivation so API consumers and
// the UI group containers the same way.
var stackLabels = []string{
	"dev.dozzle.group",
	"coolify.projectName",
	"com.docker.stack.namespace",
	"com.docker.compose.project",
}

func stackName(c container.Container) string {
	for _, label := range stackLabels {
		if v := c.Labels[label]; v != "" {
			return v
		}
	}
	return ""
}

type topologyContainer struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Host      string   `json:"host"`
	State     string   `json:"state"`
	Image     string   `json:"image"`
	Stack     string   `json:"stack,omitempty"`
	Networks  []string `json:"networks,omitempty"`
	DependsOn []string `json:"dependsOn,omitempty"` // container ids resolved from compose depends_on labels
}

type topologyGroup struct {
	Name       string   `json:"name"`
	Containers []string `json:"containers"` // container ids
}

type topologyEdge struct {
	Source string `json:"source"` // container id
	Target string `json:"target"` // container id
}

type topologyResponse struct {
	GeneratedAt time.Time           `json:"generatedAt"`
	Containers  []topologyContainer `json:"containers"`
	Networks    []topologyGroup     `json:"networks"`
	Stacks      []topologyGroup     `json:"stacks"`
	DependsOn   []topologyEdge      `json:"dependsOn"`
}

// topology returns the container/network/stack graph behind the topology and
// dependency map pages as machine-readable JSON for external consumers.
func (h *handler) topology(w http.ResponseWriter, r *http.Request) {
	containers, errs := h.hostService.ListAllContainers(h.config.Labels)
	for _, err := range errs {
		log.Warn().Err(err).Msg("error listing containers for topology")
	}

	// compose depends_on entries name a service; resolve them to container ids
	// scoped by host and project so identical project names on two hosts don't collide
	serviceKey := func(c container.Container, service string) string {
		return c.Host + "\x00" + c.Labels["com.docker.compose.project"] + "\x00" + service
	}
	byService := make(map[string]string, len(containers))
	for _, c := range containers {
		if service := c.Labels["com.docker.compose.service"]; service != "" && c.Labels["com.docker.compose.project"] != "" {
			byService[serviceKey(c, service)] = c.ID
		}
	}

	networks := make(map[string][]string)
	stacks := make(map[string][]string)
	dependsOnEdges := []topologyEdge{}
	response := topologyResponse{
		GeneratedAt: time.Now().UTC(),
		Containers:  make([]topologyContainer, 0, len(containers)),
		DependsOn:   []topologyEdge{},
	}

	for _, c := range containers {
		stack := stackName(c)
		var dependsOn []string
		if c.Labels["com.docker.compose.project"] != "" {
			for entry := range strings.SplitSeq(c.Labels["com.docker.compose.depends_on"], ",") {
				service, _, _ := strings.Cut(strings.TrimSpace(entry), ":")
				if service == "" {
					continue
				}
				if id, ok := byService[serviceKey(c, service)]; ok && id != c.ID {
					dependsOn = append(dependsOn, id)
					dependsOnEdges = append(dependsOnEdges, topologyEdge{Source: c.ID, Target: id})
				}
			}
		}

		response.Containers = append(response.Containers, topologyContainer{
			ID:        c.ID,
			Name:      c.Name,
			Host:      c.Host,
			State:     c.State,
			Image:     c.Image,
			Stack:     stack,
			Networks:  c.Networks,
			DependsOn: dependsOn,
		})

		for _, net := range c.Networks {
			networks[net] = append(networks[net], c.ID)
		}
		if stack != "" {
			stacks[stack] = append(stacks[stack], c.ID)
		}
	}

	sort.Slice(response.Containers, func(i, j int) bool {
		return response.Containers[i].Name < response.Containers[j].Name
	})
	response.Networks = sortedGroups(networks)
	response.Stacks = sortedGroups(stacks)
	sort.Slice(dependsOnEdges, func(i, j int) bool {
		if dependsOnEdges[i].Source != dependsOnEdges[j].Source {
			return dependsOnEdges[i].Source < dependsOnEdges[j].Source
		}
		return dependsOnEdges[i].Target < dependsOnEdges[j].Target
	})
	response.DependsOn = dependsOnEdges

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Error().Err(err).Msg("error encoding topology response")
	}
}

func sortedGroups(groups map[string][]string) []topologyGroup {
	result := make([]topologyGroup, 0, len(groups))
	for name, ids := range groups {
		sort.Strings(ids)
		result = append(result, topologyGroup{Name: name, Containers: ids})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

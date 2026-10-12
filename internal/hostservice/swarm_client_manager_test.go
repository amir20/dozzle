package hostservice

import (
	"errors"
	"testing"

	"github.com/amir20/dozzle/internal/container"
	"github.com/puzpuzpuz/xsync/v4"
	"github.com/stretchr/testify/assert"
)

func swarmManager() *SwarmClientManager {
	return &SwarmClientManager{
		clients: map[string]container.ClientService{
			"node-1": &stubService{host: container.Host{ID: "node-1"}},
			"node-2": &stubService{host: container.Host{ID: "node-2"}},
			"node-3": &stubService{host: container.Host{ID: "node-3"}, err: errors.New("unreachable")},
		},
		subscribers: xsync.NewMap[*hostSubscriber, struct{}](),
		agentManager: managerWith(map[string]container.ClientService{
			"agent": &stubService{host: container.Host{ID: "agent"}},
		}),
	}
}

func TestSwarmClientManager_ListIncludesAgentsAndNodes(t *testing.T) {
	assert.Len(t, swarmManager().List(), 4)
}

// Every node is asked for its host, and one that fails is reported as
// unavailable instead of dropped.
func TestSwarmClientManager_HostsAsksEveryNode(t *testing.T) {
	hosts := swarmManager().Hosts(t.Context())

	assert.ElementsMatch(t, []string{"agent", "node-1", "node-2", "node-3"}, hostIDs(hosts))
	for _, h := range hosts {
		if h.ID == "agent" {
			continue
		}
		assert.Equal(t, "swarm", h.Type)
		assert.Equalf(t, h.ID != "node-3", h.Available, "availability of %s", h.ID)
	}
}

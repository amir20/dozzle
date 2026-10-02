package hostservice

import (
	"context"
	"testing"

	"github.com/amir20/dozzle/internal/container"
	"github.com/puzpuzpuz/xsync/v4"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// A workload is matched on namespace, kind and name together, so a same-named
// Deployment in another namespace never counts as visible.
func TestWorkloadVisible(t *testing.T) {
	containers := []container.Container{{Labels: map[string]string{
		"@k8s.namespace":     "default",
		"@k8s.owner.count":   "2",
		"@k8s.owner.0.kind":  "ReplicaSet",
		"@k8s.owner.0.name":  "api-6f88b977f4",
		"@k8s.owner.1.kind":  "Deployment",
		"@k8s.owner.1.name":  "api",
		"@k8s.workload.kind": "Deployment",
		"@k8s.workload.name": "api",
	}}}

	assert.True(t, workloadVisible(containers, "default", "Deployment", "api"))
	assert.False(t, workloadVisible(containers, "prod", "Deployment", "api"))
	assert.False(t, workloadVisible(containers, "default", "StatefulSet", "api"))
	assert.False(t, workloadVisible(containers, "default", "Deployment", "web"))
	assert.False(t, workloadVisible(nil, "default", "Deployment", "api"))
}

// An operator's custom resource sits above the StatefulSet, so the StatefulSet
// is not the top of the chain but is still what gets rolled out.
func TestWorkloadVisibleBelowCustomResource(t *testing.T) {
	containers := []container.Container{{Labels: map[string]string{
		"@k8s.namespace":     "monitoring",
		"@k8s.owner.count":   "2",
		"@k8s.owner.0.kind":  "StatefulSet",
		"@k8s.owner.0.name":  "prometheus-main",
		"@k8s.owner.1.kind":  "Prometheus",
		"@k8s.owner.1.name":  "main",
		"@k8s.workload.kind": "Prometheus",
		"@k8s.workload.name": "main",
	}}}

	assert.True(t, workloadVisible(containers, "monitoring", "StatefulSet", "prometheus-main"))
}

func TestNodeToHostReadsReadyCondition(t *testing.T) {
	node := &corev1.Node{
		Name: "worker-2",
		Status: corev1.NodeStatus{
			Capacity: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("4"),
				corev1.ResourceMemory: resource.MustParse("8Gi"),
			},
			Conditions: []corev1.NodeCondition{{Type: corev1.NodeReady, Status: corev1.ConditionFalse}},
		},
	}

	host := nodeToHost(node)
	assert.Equal(t, "worker-2", host.ID)
	assert.Equal(t, 4, host.NCPU)
	assert.False(t, host.Available)

	node.Status.Conditions[0].Status = corev1.ConditionTrue
	assert.True(t, nodeToHost(node).Available)
}

func TestSetHostPublishesOnlyChanges(t *testing.T) {
	m := &K8sClusterService{
		hosts:           xsync.NewMap[string, container.Host](),
		hostSubscribers: xsync.NewMap[context.Context, chan<- container.Host](),
	}
	updates := make(chan container.Host, 10)
	m.SubscribeAvailableHosts(t.Context(), updates)

	added := container.Host{ID: "worker-2", Name: "worker-2", Type: "k8s", Available: true}
	m.setHost(added)
	m.setHost(added) // a status heartbeat with nothing new
	gone := added
	gone.Available = false
	m.setHost(gone)

	assert.Equal(t, added, <-updates)
	assert.Equal(t, gone, <-updates)
	assert.Empty(t, updates)
	assert.Equal(t, []container.Host{gone}, m.Hosts())
}

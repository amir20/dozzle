package k8s

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
)

const testSummary = `{"pods":[{
	"podRef":{"name":"web","namespace":"default"},
	"network":{"time":"2026-01-01T00:00:10Z","rxBytes":1000,"txBytes":500},
	"containers":[{"name":"app","rootfs":{"usedBytes":4096}},{"name":"sidecar"}],
	"volume":[{"name":"data","usedBytes":1048576},{"name":"scratch","usedBytes":2048},{"name":"kube-api-access","usedBytes":12}]
}]}`

// kubeletNode stands up a kubelet answering with status and body, and a node that
// points at it.
func kubeletNode(t *testing.T, status int, body string) (*corev1.Node, *http.Client) {
	t.Helper()
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/stats/summary", r.URL.Path)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	host, port, err := net.SplitHostPort(srv.Listener.Addr().String())
	require.NoError(t, err)
	p, err := strconv.Atoi(port)
	require.NoError(t, err)
	node := &corev1.Node{
		Name: "node-1",
		Status: corev1.NodeStatus{
			Addresses:       []corev1.NodeAddress{{Type: corev1.NodeHostName, Address: "node-1"}, {Type: corev1.NodeInternalIP, Address: host}},
			DaemonEndpoints: corev1.NodeDaemonEndpoints{KubeletEndpoint: corev1.DaemonEndpoint{Port: int32(p)}},
		},
	}
	return node, srv.Client()
}

func TestContainerSizesReadsRootfs(t *testing.T) {
	node, kubelet := kubeletNode(t, http.StatusOK, testSummary)
	client := &Client{Clientset: k8sfake.NewSimpleClientset(node), kubelet: kubelet}

	sizes, err := client.ContainerSizes(t.Context())
	require.NoError(t, err)
	assert.Equal(t, map[string]int64{"default:web:app": 4096}, sizes)
}

// A kubelet that RBAC keeps out turns sizes off instead of being retried.
func TestContainerSizesRefusedTurnsSizesOff(t *testing.T) {
	node, kubelet := kubeletNode(t, http.StatusForbidden, "Forbidden")
	client := &Client{Clientset: k8sfake.NewSimpleClientset(node), kubelet: kubelet}

	_, err := client.ContainerSizes(t.Context())
	assert.ErrorIs(t, err, container.ErrLayerSizeUnsupported)
}

// A kubelet whose certificate the cluster CA did not sign is refused the same way,
// so the service account token is never sent to it.
func TestContainerSizesUntrustedKubeletTurnsSizesOff(t *testing.T) {
	node, _ := kubeletNode(t, http.StatusOK, testSummary)
	client := &Client{Clientset: k8sfake.NewSimpleClientset(node), kubelet: &http.Client{}}

	_, err := client.ContainerSizes(t.Context())
	assert.ErrorIs(t, err, container.ErrLayerSizeUnsupported)
}

// A kubeconfig that skips verification never sends its credentials to a kubelet.
func TestInsecureKubeconfigHasNoKubeletClient(t *testing.T) {
	kubelet, err := newKubeletHTTPClient(&rest.Config{Insecure: true})
	require.NoError(t, err)
	assert.Nil(t, kubelet)
}

// A size asked for right after a summary read comes from it, not from another read.
func TestContainerSizeUsesRecentSummary(t *testing.T) {
	node, kubelet := kubeletNode(t, http.StatusOK, testSummary)
	client := &Client{Clientset: k8sfake.NewSimpleClientset(node), kubelet: kubelet}
	_, err := client.ContainerSizes(t.Context())
	require.NoError(t, err)

	// no pod exists, so only the cache can answer
	size, err := client.ContainerSize(t.Context(), "default:web:app")
	require.NoError(t, err)
	assert.Equal(t, int64(4096), size)
}

func TestVolumeUsageCountsClaimsAndEmptyDirs(t *testing.T) {
	pod := corev1.Pod{
		Name: "web", Namespace: "default",
		Spec: corev1.PodSpec{
			Volumes: []corev1.Volume{
				{Name: "data", PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "web-data"}},
				{Name: "scratch", EmptyDir: &corev1.EmptyDirVolumeSource{}},
				{Name: "kube-api-access", Projected: &corev1.ProjectedVolumeSource{}},
			},
			Containers: []corev1.Container{
				{Name: "app", VolumeMounts: []corev1.VolumeMount{
					{Name: "data", MountPath: "/data"},
					{Name: "scratch", MountPath: "/tmp"},
					{Name: "kube-api-access", MountPath: "/var/run/secrets"},
				}},
				{Name: "sidecar", VolumeMounts: []corev1.VolumeMount{{Name: "data", MountPath: "/mnt"}}},
			},
		},
	}
	node, kubelet := kubeletNode(t, http.StatusOK, testSummary)
	client := &Client{Clientset: k8sfake.NewSimpleClientset(node), kubelet: kubelet}
	stats, _, err := client.kubeletStats(t.Context())
	require.NoError(t, err)

	volumes := volumeUsage([]corev1.Pod{pod}, stats)

	assert.Equal(t, map[string][]container.VolumeUsage{
		"default:web:app": {
			{Name: "web-data", Destination: "/data", Size: 1 << 20, Links: 2},
			{Name: "scratch", Destination: "/tmp", Size: 2048, Links: 1},
		},
		"default:web:sidecar": {
			{Name: "web-data", Destination: "/mnt", Size: 1 << 20, Links: 2},
		},
	}, volumes)
}

func TestPodNetworkTotalsWalkBetweenReadings(t *testing.T) {
	start := time.Unix(1000, 0)
	first := netSample{rx: 10_000, tx: 5_000, kubelet: start, at: start}
	p := podNetwork{base: first, prev: first, last: first}

	rx, tx := p.totals(start.Add(3 * time.Second))
	assert.Zero(t, rx, "the first reading is the baseline, not traffic")
	assert.Zero(t, tx)

	p = p.observe(netSample{rx: 20_000, tx: 6_000, kubelet: start.Add(10 * time.Second), at: start.Add(10 * time.Second)})
	rx, _ = p.totals(start.Add(10 * time.Second))
	assert.Equal(t, uint64(0), rx, "a new reading starts from the previous one")
	rx, tx = p.totals(start.Add(15 * time.Second))
	assert.Equal(t, uint64(5_000), rx, "halfway through the interval")
	assert.Equal(t, uint64(500), tx)
	rx, _ = p.totals(start.Add(time.Minute))
	assert.Equal(t, uint64(10_000), rx, "holds at the last reading")

	// the kubelet has not refreshed yet, so the same reading is not a new one
	p = p.observe(netSample{rx: 20_000, tx: 6_000, kubelet: start.Add(10 * time.Second), at: start.Add(20 * time.Second)})
	rx, _ = p.totals(start.Add(15 * time.Second))
	assert.Equal(t, uint64(5_000), rx)

	// counters that go backwards started over
	p = p.observe(netSample{rx: 100, tx: 100, kubelet: start.Add(30 * time.Second), at: start.Add(30 * time.Second)})
	rx, tx = p.totals(start.Add(31 * time.Second))
	assert.Zero(t, rx)
	assert.Zero(t, tx)
}

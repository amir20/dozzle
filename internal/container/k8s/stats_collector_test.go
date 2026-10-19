package k8s

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/puzpuzpuz/xsync/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	metricsclient "k8s.io/metrics/pkg/client/clientset/versioned"
)

// newTestCollector builds a collector with no namespaces, so each tick polls
// nothing and the metrics client is never touched.
func newTestCollector() *StatsCollector {
	return &StatsCollector{
		client:         &Client{Clientset: k8sfake.NewSimpleClientset()},
		subscribers:    xsync.NewMap[context.Context, chan<- container.ContainerStat](),
		metricsFailing: xsync.NewMap[string, bool](),
	}
}

// running reports whether a collector is live, read under the same lock Start
// and forceStop use.
func running(sc *StatsCollector) bool {
	return sc.lifecycle.Running()
}

func TestStopBeforeStartDoesNotLeakCollector(t *testing.T) {
	collector := newTestCollector()
	collector.Stop()

	done := make(chan bool)
	go func() { done <- collector.Start(t.Context()) }()

	select {
	case started := <-done:
		assert.False(t, started, "start after its own stop should not run a collector")
	case <-time.After(2 * time.Second):
		t.Fatal("Start blocked, so it started a collector nobody holds")
	}
	assert.False(t, running(collector), "no collector should be running")
	assert.Equal(t, 0, collector.lifecycle.Holders())
}

func TestStopTimerEndsStartedCollector(t *testing.T) {
	old := timeToStop
	timeToStop = 10 * time.Millisecond
	t.Cleanup(func() { timeToStop = old })

	collector := newTestCollector()
	done := make(chan bool)
	go func() { done <- collector.Start(t.Context()) }()

	assert.Eventually(t, func() bool { return running(collector) }, 2*time.Second, time.Millisecond)
	collector.Stop()

	select {
	case stopped := <-done:
		assert.True(t, stopped, "the stop timer should end the collector")
	case <-time.After(2 * time.Second):
		t.Fatal("stop timer never ended the collector")
	}
	assert.False(t, running(collector))
}

// Each tick polls every namespace, and one namespace the metrics API refuses
// does not stop the others from reporting.
func TestStatsCollectorPollsEveryNamespace(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// /apis/metrics.k8s.io/v1beta1/namespaces/{ns}/pods
		parts := strings.Split(r.URL.Path, "/")
		ns := parts[len(parts)-2]
		w.Header().Set("Content-Type", "application/json")
		if ns == "denied" {
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprint(w, `{"kind":"Status","apiVersion":"v1","status":"Failure","reason":"Forbidden","code":403}`)
			return
		}
		fmt.Fprintf(w, `{"kind":"PodMetricsList","apiVersion":"metrics.k8s.io/v1beta1","items":[
			{"metadata":{"name":"pod-%[1]s","namespace":"%[1]s"},"timestamp":"2026-01-01T00:00:00Z","window":"1s",
			 "containers":[{"name":"app","usage":{"cpu":"500m","memory":"1Mi"}}]}]}`, ns)
	}))
	t.Cleanup(srv.Close)

	metrics, err := metricsclient.NewForConfig(&rest.Config{Host: srv.URL})
	require.NoError(t, err)
	collector := newTestCollector()
	collector.client.Clientset = k8sfake.NewSimpleClientset(
		&corev1.Pod{
			Name: "pod-a", Namespace: "a",
			Spec: corev1.PodSpec{Containers: []corev1.Container{{
				Name:      "app",
				Resources: corev1.ResourceRequirements{Limits: corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("4Mi")}},
			}}},
		},
	)
	collector.client.namespace = []string{"a", "denied", "b", "c"}
	collector.metrics = metrics

	ctx, cancel := context.WithCancel(t.Context())
	stats := make(chan container.ContainerStat, 100)
	collector.Subscribe(ctx, stats)
	done := make(chan struct{})
	go func() {
		collector.Start(ctx)
		close(done)
	}()

	want := map[string]bool{"a:pod-a:app": true, "b:pod-b:app": true, "c:pod-c:app": true}
	seen := map[string]bool{}
	timeout := time.After(5 * time.Second)
	for len(seen) < len(want) {
		select {
		case stat := <-stats:
			assert.Truef(t, want[stat.ID], "unexpected stat for %s", stat.ID)
			assert.InDelta(t, 50.0, stat.CPUPercent, 0.001)
			if stat.ID == "a:pod-a:app" {
				assert.InDelta(t, 25.0, stat.MemoryPercent, 0.001, "1Mi used of a 4Mi limit")
			}
			seen[stat.ID] = true
		case <-timeout:
			t.Fatalf("only saw stats for %v", seen)
		}
	}
	cancel()
	<-done

	_, failing := collector.metricsFailing.Load("denied")
	assert.True(t, failing, "the refused namespace is remembered as failing")
}

// A container's own limit wins; without one it is measured against its node, the
// way Docker reports a container with no limit against the host.
func TestMemoryLimitsFallBackToNodeCapacity(t *testing.T) {
	limited := corev1.Container{
		Name:      "limited",
		Resources: corev1.ResourceRequirements{Limits: corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("256Mi")}},
	}
	pods := []corev1.Pod{{
		Name: "web", Namespace: "default",
		Spec: corev1.PodSpec{
			NodeName:       "node-1",
			InitContainers: []corev1.Container{{Name: "migrate"}},
			Containers:     []corev1.Container{limited, {Name: "sidecar"}},
		},
	}, {
		Name: "pending", Namespace: "default",
		Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "app"}}},
	}}

	limits := memoryLimits(pods, map[string]float64{"node-1": 8 << 30})

	assert.Equal(t, map[string]float64{
		"default:web:limited": 256 << 20,
		"default:web:sidecar": 8 << 30,
		"default:web:migrate": 8 << 30,
		"default:pending:app": 0, // not scheduled yet, so no node to measure against
	}, limits)
}

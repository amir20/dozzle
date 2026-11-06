package k8s

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
)

func TestContainerStatusToState(t *testing.T) {
	podStarted := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	runStarted := metav1.NewTime(podStarted.Add(time.Minute))
	runFinished := metav1.NewTime(podStarted.Add(2 * time.Minute))

	tests := []struct {
		name   string
		status corev1.ContainerStatus
		want   string
	}{
		{"running", corev1.ContainerStatus{State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{StartedAt: runStarted}}}, "running"},
		{"terminated", corev1.ContainerStatus{State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{StartedAt: runStarted, FinishedAt: runFinished}}}, "exited"},
		{"crash loop", corev1.ContainerStatus{
			State:                corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff"}},
			LastTerminationState: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{StartedAt: runStarted, FinishedAt: runFinished}},
		}, "restarting"},
		{"creating", corev1.ContainerStatus{State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "ContainerCreating"}}}, "created"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state, _, _ := containerStatusToState(tt.status, podStarted)
			assert.Equal(t, tt.want, state)
		})
	}
}

// The pod phase stays Running while one of its containers crash-loops.
func TestPodToContainersUsesEachContainersOwnState(t *testing.T) {
	client := newTestK8sClient(t)
	pod := &corev1.Pod{
		Namespace: "default", Name: "web",
		Spec: corev1.PodSpec{
			InitContainers: []corev1.Container{{Name: "migrate"}},
			Containers:     []corev1.Container{{Name: "app"}, {Name: "sidecar"}},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			InitContainerStatuses: []corev1.ContainerStatus{
				{Name: "migrate", State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{}}},
			},
			ContainerStatuses: []corev1.ContainerStatus{
				{Name: "app", State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}},
				{
					Name:                 "sidecar",
					State:                corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff"}},
					LastTerminationState: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{}},
				},
			},
		},
	}

	containers := client.podToContainers(t.Context(), pod)
	require.Len(t, containers, 3)

	assert.Equal(t, "default:web:migrate", containers[0].ID)
	assert.Equal(t, "exited", containers[0].State)
	assert.Equal(t, "true", containers[0].Labels["@k8s.init"])

	assert.Equal(t, "running", containers[1].State)
	assert.Empty(t, containers[1].Labels["@k8s.init"])
	assert.Equal(t, "restarting", containers[2].State)
}

func TestPreviousRunIsJoinedOnALineBreak(t *testing.T) {
	for _, previous := range []string{"2026-01-01T00:00:00Z boot\n", "2026-01-01T00:00:00Z boot"} {
		joined, err := io.ReadAll(io.MultiReader(&newlineTerminated{r: iotest.OneByteReader(strings.NewReader(previous))}, strings.NewReader("2026-01-01T00:00:05Z again\n")))
		require.NoError(t, err)
		assert.Equal(t, "2026-01-01T00:00:00Z boot\n2026-01-01T00:00:05Z again\n", string(joined))
	}

	assert.True(t, startsWithTimestamp([]byte("2026-09-15T14:09:05.417027173Z boot")))
	assert.False(t, startsWithTimestamp([]byte("unable to retrieve container logs for containerd://c286")))
}

func TestPodLabelSelectorKeepsOnlyPodLabels(t *testing.T) {
	selector := podLabelSelector(container.ContainerLabels{
		"dozzle":    {"true"},
		"app":       {"web"},
		"namespace": {"prod"},
	})
	assert.Equal(t, "app=web,dozzle=true", selector)
}

func TestContainerActionsRestartDeletesControlledPod(t *testing.T) {
	isController := true
	controlled := &corev1.Pod{
		Namespace: "default", Name: "api-abc",
		OwnerReferences: []metav1.OwnerReference{{APIVersion: "apps/v1", Kind: "ReplicaSet", Name: "api", Controller: &isController}}}
	bare := &corev1.Pod{Namespace: "default", Name: "debug"}

	client := newTestK8sClient(t)
	client.Clientset = k8sfake.NewSimpleClientset(controlled, bare)

	require.NoError(t, client.ContainerActions(t.Context(), container.Restart, "default:api-abc:api"))
	_, err := client.Clientset.CoreV1().Pods("default").Get(t.Context(), "api-abc", metav1.GetOptions{})
	assert.True(t, apierrors.IsNotFound(err), "restart should delete the pod so its controller recreates it")

	err = client.ContainerActions(t.Context(), container.Restart, "default:debug:debug")
	assert.ErrorIs(t, err, errors.ErrUnsupported)
	_, err = client.Clientset.CoreV1().Pods("default").Get(t.Context(), "debug", metav1.GetOptions{})
	assert.NoError(t, err, "a pod with no controller must not be deleted")

	assert.ErrorIs(t, client.ContainerActions(t.Context(), container.Stop, "default:debug:debug"), errors.ErrUnsupported)
	assert.ErrorIs(t, client.ContainerStats(t.Context(), "default:debug:debug", nil), errors.ErrUnsupported)
}

func TestContainerEventsReportsInitialPodsAsUpdatesAndNewOnesAsCreates(t *testing.T) {
	existing := &corev1.Pod{
		Namespace: "default", Name: "existing",
		Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "app"}}},
	}
	client := newTestK8sClient(t)
	client.Clientset = k8sfake.NewSimpleClientset(existing)
	client.namespace = []string{"default"}

	events := make(chan container.ContainerEvent, 10)
	done := make(chan error)
	go func() { done <- client.ContainerEvents(t.Context(), events) }()

	next := func() container.ContainerEvent {
		t.Helper()
		select {
		case event := <-events:
			return event
		case <-time.After(5 * time.Second):
			t.Fatal("timed out waiting for a container event")
			return container.ContainerEvent{}
		}
	}

	// The first list goes out as updates, so the store can pick up a pod created
	// between its own list and the informer's.
	initial := next()
	assert.Equal(t, "update", initial.Name)
	assert.Equal(t, "default:existing:app", initial.ActorID)

	pods := client.Clientset.CoreV1().Pods("default")
	_, err := pods.Create(t.Context(), &corev1.Pod{
		Namespace: "default", Name: "job-1",
		Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "run"}}},
	}, metav1.CreateOptions{})
	require.NoError(t, err)

	created := next()
	assert.Equal(t, "create", created.Name)
	assert.Equal(t, "default:job-1:run", created.ActorID)

	_, name, _ := parsePodContainerID(created.ActorID)
	require.NoError(t, pods.Delete(t.Context(), name, metav1.DeleteOptions{}))
	for {
		if event := next(); event.ActorID == created.ActorID && event.Name == "destroy" {
			break
		}
	}
}

func TestContainerRunFactsReadsTheLastRun(t *testing.T) {
	crashLoop := corev1.ContainerStatus{
		RestartCount:         31,
		State:                corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff"}},
		LastTerminationState: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{Reason: "OOMKilled", ExitCode: 137}},
	}
	f := containerRunFacts(crashLoop)
	if f.restarts != 31 || !f.oomKilled || f.exitCode != 137 {
		t.Fatalf("crash loop facts = %+v", f)
	}
	done := corev1.ContainerStatus{State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{Reason: "Error", ExitCode: 1}}}
	if f := containerRunFacts(done); f.oomKilled || f.exitCode != 1 {
		t.Fatalf("terminated facts = %+v", f)
	}
}

type closeCounter struct {
	io.Reader
	closes *int
}

func (c closeCounter) Close() error {
	*c.closes++
	return nil
}

func TestLogsUntil(t *testing.T) {
	end := time.Date(2026, 11, 6, 14, 0, 50, 0, time.UTC)
	lines := "2026-11-06T14:00:49.5Z before\n" +
		"2026-11-06T14:00:50Z at the end\n" +
		"not a timestamp stays\n" +
		"2026-11-06T14:00:50.000000001Z past\n" +
		"2026-11-06T14:00:40Z never reached\n"
	want := "2026-11-06T14:00:49.5Z before\n2026-11-06T14:00:50Z at the end\nnot a timestamp stays\n"

	readers := map[string]func(io.Reader) io.Reader{
		"whole":    func(r io.Reader) io.Reader { return r },
		"one byte": iotest.OneByteReader,
		"half":     iotest.HalfReader,
	}
	for name, wrap := range readers {
		t.Run(name, func(t *testing.T) {
			got, err := io.ReadAll(iotest.OneByteReader(newLogsUntil(io.NopCloser(wrap(strings.NewReader(lines))), end)))
			require.NoError(t, err)
			assert.Equal(t, want, string(got))
		})
	}

	t.Run("a numeric offset with a full fraction is cut too", func(t *testing.T) {
		offset := "2026-11-06T09:00:49.123456789-05:00 in\n2026-11-06T09:00:51.123456789-05:00 past\n"
		got, err := io.ReadAll(newLogsUntil(io.NopCloser(strings.NewReader(offset)), end))
		require.NoError(t, err)
		assert.Equal(t, "2026-11-06T09:00:49.123456789-05:00 in\n", string(got))
	})

	t.Run("the cut closes the stream", func(t *testing.T) {
		closes := 0
		stream := closeCounter{Reader: strings.NewReader(lines), closes: &closes}
		_, err := io.ReadAll(newLogsUntil(stream, end))
		require.NoError(t, err)
		assert.Equal(t, 1, closes)
	})

	t.Run("a zero end is no end", func(t *testing.T) {
		got, err := io.ReadAll(newLogsUntil(io.NopCloser(strings.NewReader(lines)), time.Time{}))
		require.NoError(t, err)
		assert.Equal(t, lines, string(got))
	})

	t.Run("a last line with no newline is kept", func(t *testing.T) {
		got, err := io.ReadAll(newLogsUntil(io.NopCloser(strings.NewReader("2026-11-06T14:00:49Z partial")), end))
		require.NoError(t, err)
		assert.Equal(t, "2026-11-06T14:00:49Z partial", string(got))
	})

	t.Run("a line longer than the buffer passes through whole", func(t *testing.T) {
		long := "2026-11-06T14:00:49Z " + strings.Repeat("x", 100_000) + "\n"
		got, err := io.ReadAll(newLogsUntil(io.NopCloser(strings.NewReader(long+"2026-11-06T14:00:51Z past\n")), end))
		require.NoError(t, err)
		assert.Equal(t, long, string(got))
	})
}

// A window that ends before now returns that window, not the newest lines. Real HTTP,
// since the fake clientset answers every log request with one fixed untimestamped body.
func TestContainerLogsBetweenDatesStopsAtEnd(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("previous") == "true" {
			http.Error(w, "no previous run", http.StatusBadRequest)
			return
		}
		io.WriteString(w, "2026-11-06T14:00:49Z in window\n2026-11-06T14:00:51Z past the end\n")
	}))
	defer server.Close()
	clientset, err := kubernetes.NewForConfig(&rest.Config{Host: server.URL})
	require.NoError(t, err)

	client := &Client{Clientset: clientset}
	end := time.Date(2026, 11, 6, 14, 0, 50, 0, time.UTC)
	stream, err := client.ContainerLogsBetweenDates(t.Context(), "ns:pod:c", end.Add(-time.Minute), end, container.STDALL)
	require.NoError(t, err)
	defer stream.Close()

	got, err := io.ReadAll(stream)
	require.NoError(t, err)
	assert.Equal(t, "2026-11-06T14:00:49Z in window\n", string(got))
}

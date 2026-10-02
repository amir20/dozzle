package k8s

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestPodToContainersAddsOwnerChainLabels(t *testing.T) {
	client := newTestK8sClient(t,
		&appsv1.ReplicaSet{
			APIVersion: "apps/v1", Kind: "ReplicaSet",
			Namespace: "default",
			Name:      "api-6f88b977f4",
			UID:       types.UID("rs-uid"),
			OwnerReferences: []metav1.OwnerReference{
				{APIVersion: "apps/v1", Kind: "Deployment", Name: "api", UID: types.UID("deploy-uid")},
			},
		},
		&appsv1.Deployment{
			APIVersion: "apps/v1", Kind: "Deployment",
			Namespace: "default", Name: "api", UID: types.UID("deploy-uid"),
		},
	)

	containers := client.podToContainers(t.Context(), podWithOwner())
	require.Len(t, containers, 1)

	labels := containers[0].Labels
	assert.Equal(t, "default", labels["namespace"])
	assert.Equal(t, "default", labels["@k8s.namespace"])
	assert.Equal(t, "ReplicaSet", labels["owner.kind"])
	assert.Equal(t, "api-6f88b977f4", labels["owner.name"])
	assert.Equal(t, "ReplicaSet~default~api-6f88b977f4", labels["owner.key"])
	assert.Equal(t, "2", labels["@k8s.owner.count"])

	assert.Equal(t, "ReplicaSet", labels["@k8s.owner.0.kind"])
	assert.Equal(t, "api-6f88b977f4", labels["@k8s.owner.0.name"])
	assert.Equal(t, "ReplicaSet~default~api-6f88b977f4", labels["@k8s.owner.0.key"])
	assert.Equal(t, "Deployment", labels["@k8s.owner.1.kind"])
	assert.Equal(t, "api", labels["@k8s.owner.1.name"])
	assert.Equal(t, "Deployment~default~api", labels["@k8s.owner.1.key"])
	assert.Equal(t, "Deployment", labels["@k8s.workload.kind"])
	assert.Equal(t, "api", labels["@k8s.workload.name"])
	// Legacy duplicated k8s.owner.* labels are no longer emitted.
	assert.Empty(t, labels["k8s.owner.count"])
	assert.Empty(t, labels["k8s.owner.0.kind"])
	assert.Equal(t, "true", labels[ownerMembershipLabel("ReplicaSet~default~api-6f88b977f4")])
	assert.Equal(t, "true", labels[ownerMembershipLabel("Deployment~default~api")])
}

func TestPodToContainersStopsOwnerChainWhenOwnerCannotBeFetched(t *testing.T) {
	client := newTestK8sClient(t)

	containers := client.podToContainers(t.Context(), podWithOwner())
	require.Len(t, containers, 1)

	labels := containers[0].Labels
	assert.Equal(t, "1", labels["@k8s.owner.count"])
	assert.Equal(t, "ReplicaSet", labels["@k8s.owner.0.kind"])
	assert.Equal(t, "api-6f88b977f4", labels["@k8s.owner.0.name"])
	assert.Empty(t, labels["@k8s.owner.1.kind"])
}

func TestPodToContainersDoesNotAddNodeOwner(t *testing.T) {
	client := newTestK8sClient(t)
	pod := podWithOwner()
	pod.OwnerReferences = []metav1.OwnerReference{
		{APIVersion: "v1", Kind: "Node", Name: "node-1", UID: types.UID("node-uid")},
	}

	containers := client.podToContainers(t.Context(), pod)
	require.Len(t, containers, 1)

	labels := containers[0].Labels
	assert.Empty(t, labels["owner.kind"])
	assert.Empty(t, labels["@k8s.owner.count"])
}

func TestPodToContainersStopsBeforeNodeOwnerInChain(t *testing.T) {
	client := newTestK8sClient(t,
		&appsv1.ReplicaSet{
			APIVersion: "apps/v1", Kind: "ReplicaSet",
			Namespace: "default",
			Name:      "api-6f88b977f4",
			UID:       types.UID("rs-uid"),
			OwnerReferences: []metav1.OwnerReference{
				{APIVersion: "v1", Kind: "Node", Name: "node-1", UID: types.UID("node-uid")},
			},
		},
	)

	containers := client.podToContainers(t.Context(), podWithOwner())
	require.Len(t, containers, 1)

	labels := containers[0].Labels
	assert.Equal(t, "1", labels["@k8s.owner.count"])
	assert.Equal(t, "ReplicaSet", labels["@k8s.owner.0.kind"])
	assert.Empty(t, labels["@k8s.owner.1.kind"])
}

func TestListContainersAppliesSyntheticOwnerFiltersAfterPodList(t *testing.T) {
	client := newTestK8sClient(t,
		&appsv1.ReplicaSet{
			APIVersion: "apps/v1", Kind: "ReplicaSet",
			Namespace: "default",
			Name:      "api-6f88b977f4",
			UID:       types.UID("rs-uid"),
			OwnerReferences: []metav1.OwnerReference{
				{APIVersion: "apps/v1", Kind: "Deployment", Name: "api", UID: types.UID("deploy-uid")},
			},
		},
		&appsv1.Deployment{
			APIVersion: "apps/v1", Kind: "Deployment",
			Namespace: "default", Name: "api", UID: types.UID("deploy-uid"),
		},
	)
	client.Clientset = k8sfake.NewSimpleClientset(podWithOwner())

	containers, err := client.ListContainers(t.Context(), container.ContainerLabels{
		"app": {"api"},
		ownerMembershipLabel("Deployment~default~api"): {"true"},
	})

	require.NoError(t, err)
	require.Len(t, containers, 1)
	assert.Equal(t, "default:api-6f88b977f4-pod:api", containers[0].ID)
}

func TestSplitK8sFiltersKeepsInvalidSyntheticKeysOutOfPodSelector(t *testing.T) {
	podLabels, metadataLabels := splitK8sFilters(container.ContainerLabels{
		"app":                        {"api"},
		"team.example.com/component": {"backend"},
		"@k8s.namespace":             {"default"},
		"@k8s.owner.key.abc123":      {"true"},
		"not:a:kubernetes:label:key": {"value"},
	})

	assert.Equal(t, container.ContainerLabels{
		"app":                        {"api"},
		"team.example.com/component": {"backend"},
	}, podLabels)
	assert.Equal(t, container.ContainerLabels{
		"@k8s.namespace":             {"default"},
		"@k8s.owner.key.abc123":      {"true"},
		"not:a:kubernetes:label:key": {"value"},
	}, metadataLabels)
}

func TestOwnerTypeKeyUsesFullAPINameForUnknownTypes(t *testing.T) {
	assert.Equal(t, "Deployment", ownerTypeKey("apps/v1", "Deployment"))
	assert.Equal(t, "argoproj.io~v1alpha1~Rollout", ownerTypeKey("argoproj.io/v1alpha1", "Rollout"))

	ref := metav1.OwnerReference{APIVersion: "argoproj.io/v1alpha1", Kind: "Rollout", Name: "api"}
	assert.Equal(t, "argoproj.io~v1alpha1~Rollout~default~api", newK8sOwner("default", ref).Key)
}

func TestOwnerReferenceToFollowPrefersController(t *testing.T) {
	controller := true

	ref := ownerReferenceToFollow([]metav1.OwnerReference{
		{Kind: "ConfigMap", Name: "sidecar-config"},
		{Kind: "ReplicaSet", Name: "api-6f88b977f4", Controller: &controller},
	})

	assert.Equal(t, "ReplicaSet", ref.Kind)
	assert.Equal(t, "api-6f88b977f4", ref.Name)
}

func TestLookupOwnerReferencesDoesNotCacheTransientFailures(t *testing.T) {
	client := newTestK8sClient(t)
	dynamicClient := client.DynamicClient.(*dynamicfake.FakeDynamicClient)
	calls := 0
	dynamicClient.PrependReactor("get", "replicasets", func(action k8stesting.Action) (bool, runtime.Object, error) {
		calls++
		return true, nil, context.Canceled
	})

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, ok := client.lookupOwnerReferences(ctx, replicaSetOwner())
	assert.False(t, ok)

	_, ok = client.lookupOwnerReferences(ctx, replicaSetOwner())
	assert.False(t, ok)
	assert.Equal(t, 2, calls)
}

func TestLookupOwnerReferencesCachesRealNegatives(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{
			name: "forbidden",
			err:  apierrors.NewForbidden(schema.GroupResource{Group: "apps", Resource: "replicasets"}, "api-6f88b977f4", errors.New("denied")),
		},
		{
			name: "not found",
			err:  apierrors.NewNotFound(schema.GroupResource{Group: "apps", Resource: "replicasets"}, "api-6f88b977f4"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestK8sClient(t)
			dynamicClient := client.DynamicClient.(*dynamicfake.FakeDynamicClient)
			calls := 0
			dynamicClient.PrependReactor("get", "replicasets", func(action k8stesting.Action) (bool, runtime.Object, error) {
				calls++
				return true, nil, tt.err
			})

			_, ok := client.lookupOwnerReferences(t.Context(), replicaSetOwner())
			assert.False(t, ok)

			_, ok = client.lookupOwnerReferences(t.Context(), replicaSetOwner())
			assert.False(t, ok)
			assert.Equal(t, 1, calls)
		})
	}
}

func TestLookupOwnerReferencesRefetchesAfterTTL(t *testing.T) {
	client := newTestK8sClient(t,
		&appsv1.ReplicaSet{
			APIVersion: "apps/v1", Kind: "ReplicaSet",
			Namespace: "default", Name: "api-6f88b977f4", UID: types.UID("rs-uid"),
		},
	)
	dynamicClient := client.DynamicClient.(*dynamicfake.FakeDynamicClient)
	calls := 0
	dynamicClient.PrependReactor("get", "replicasets", func(action k8stesting.Action) (bool, runtime.Object, error) {
		calls++
		return false, nil, nil
	})
	owner := replicaSetOwner()

	_, ok := client.lookupOwnerReferences(t.Context(), owner)
	assert.True(t, ok)
	client.lookupOwnerReferences(t.Context(), owner)
	assert.Equal(t, 1, calls, "second lookup should be served from cache")

	// Expire the cached entry and confirm the next lookup refetches.
	client.ownerCacheMu.Lock()
	entry := client.ownerCache[owner.cacheKey()]
	entry.expiresAt = time.Now().Add(-time.Minute)
	client.ownerCache[owner.cacheKey()] = entry
	client.ownerCacheMu.Unlock()

	client.lookupOwnerReferences(t.Context(), owner)
	assert.Equal(t, 2, calls, "expired entry should trigger a refetch")
}

func TestPruneOwnerCacheEvictsGracefully(t *testing.T) {
	client := newTestK8sClient(t)
	now := time.Now()

	client.ownerCacheMu.Lock()
	for i := range 10 {
		client.ownerCache[fmt.Sprintf("expired-%d", i)] = ownerLookupResult{expiresAt: now.Add(-time.Minute)}
	}
	for i := range ownerCacheMaxSize + 100 {
		client.ownerCache[fmt.Sprintf("fresh-%d", i)] = ownerLookupResult{expiresAt: now.Add(time.Minute)}
	}

	client.pruneOwnerCache(now)

	assert.LessOrEqual(t, len(client.ownerCache), ownerCacheEvictTo)
	assert.NotEmpty(t, client.ownerCache, "should evict down, not clear wholesale")
	for i := range 10 {
		_, ok := client.ownerCache[fmt.Sprintf("expired-%d", i)]
		assert.False(t, ok, "expired entries should be dropped first")
	}
	client.ownerCacheMu.Unlock()
}

type resettableMapper struct {
	meta.RESTMapper
	resets int
}

func (m *resettableMapper) Reset() { m.resets++ }

func TestResetRESTMapperThrottles(t *testing.T) {
	mapper := &resettableMapper{}
	client := &Client{restMapper: mapper}

	assert.True(t, client.resetRESTMapper(), "first reset should proceed")
	assert.False(t, client.resetRESTMapper(), "immediate second reset should be throttled")
	assert.Equal(t, 1, mapper.resets)

	// Simulate the throttle interval elapsing.
	client.mapperMu.Lock()
	client.lastMapperReset = time.Now().Add(-2 * mapperResetInterval)
	client.mapperMu.Unlock()

	assert.True(t, client.resetRESTMapper(), "reset should proceed after the interval")
	assert.Equal(t, 2, mapper.resets)
}

func TestPodToContainersDefaultNameAndNoGroup(t *testing.T) {
	client := newTestK8sClient(t)

	containers := client.podToContainers(t.Context(), podWithOwner())
	require.Len(t, containers, 1)
	assert.Equal(t, "api-6f88b977f4-pod/api", containers[0].Name)
	assert.Empty(t, containers[0].Group)
}

func TestPodToContainersHonorsDozzleLabels(t *testing.T) {
	client := newTestK8sClient(t)
	pod := podWithOwner()
	pod.Labels["dev.dozzle.name"] = "api"
	pod.Labels["dev.dozzle.group"] = "backend"

	containers := client.podToContainers(t.Context(), pod)
	require.Len(t, containers, 1)
	assert.Equal(t, "api", containers[0].Name)
	assert.Equal(t, "backend", containers[0].Group)
}

func TestPodToContainersPrefersDozzleAnnotations(t *testing.T) {
	client := newTestK8sClient(t)
	pod := podWithOwner()
	pod.Labels["dev.dozzle.name"] = "api"
	pod.Labels["dev.dozzle.group"] = "backend"
	pod.Annotations = map[string]string{
		"dev.dozzle.name":  "Public API",
		"dev.dozzle.group": "Payments team",
	}

	containers := client.podToContainers(t.Context(), pod)
	require.Len(t, containers, 1)
	assert.Equal(t, "Public API", containers[0].Name)
	assert.Equal(t, "Payments team", containers[0].Group)
}

func TestPodToContainersReadsDozzleAnnotationsWithoutLabels(t *testing.T) {
	client := newTestK8sClient(t)
	pod := podWithOwner()
	pod.Annotations = map[string]string{
		"dev.dozzle.name":  "Public API",
		"dev.dozzle.group": "Payments team",
		"dev.dozzle.url":   "https://api.example.com",
		"dev.dozzle.icon":  "nginx",
		"other.io/ignored": "x",
	}

	containers := client.podToContainers(t.Context(), pod)
	require.Len(t, containers, 1)
	c := containers[0]
	assert.Equal(t, "Public API", c.Name)
	assert.Equal(t, "Payments team", c.Group)
	// The UI reads url, icon and group off labels, so annotations have to land there.
	assert.Equal(t, "https://api.example.com", c.Labels["dev.dozzle.url"])
	assert.Equal(t, "nginx", c.Labels["dev.dozzle.icon"])
	assert.Equal(t, "Payments team", c.Labels["dev.dozzle.group"])
	assert.NotContains(t, c.Labels, "other.io/ignored")
}

func TestPodToContainersKeepsBareNameWithInitContainer(t *testing.T) {
	client := newTestK8sClient(t)
	pod := podWithOwner()
	pod.Labels["dev.dozzle.name"] = "api"
	pod.Spec.InitContainers = []corev1.Container{{Name: "migrate", Image: "example/migrate:latest"}}

	containers := client.podToContainers(t.Context(), pod)
	require.Len(t, containers, 2)
	assert.Equal(t, "api/migrate", containers[0].Name)
	assert.Equal(t, "api", containers[1].Name)
}

func TestPodToContainersSuffixesCustomNameInMultiContainerPod(t *testing.T) {
	client := newTestK8sClient(t)
	pod := podWithOwner()
	pod.Labels["dev.dozzle.name"] = "api"
	pod.Labels["dev.dozzle.group"] = "backend"
	pod.Spec.InitContainers = []corev1.Container{{Name: "migrate", Image: "example/migrate:latest"}}
	pod.Spec.Containers = append(pod.Spec.Containers, corev1.Container{Name: "proxy", Image: "envoyproxy/envoy"})

	containers := client.podToContainers(t.Context(), pod)
	require.Len(t, containers, 3)
	assert.Equal(t, "api/migrate", containers[0].Name)
	assert.Equal(t, "api/api", containers[1].Name)
	assert.Equal(t, "api/proxy", containers[2].Name)
	for _, c := range containers {
		assert.Equal(t, "backend", c.Group)
	}
}

func podWithOwner() *corev1.Pod {
	return &corev1.Pod{
		APIVersion: "v1", Kind: "Pod",
		Namespace: "default",
		Name:      "api-6f88b977f4-pod",
		Labels:    map[string]string{"app": "api"},
		OwnerReferences: []metav1.OwnerReference{
			{APIVersion: "apps/v1", Kind: "ReplicaSet", Name: "api-6f88b977f4", UID: types.UID("rs-uid")},
		},
		Spec: corev1.PodSpec{
			NodeName: "node-1",
			Containers: []corev1.Container{
				{Name: "api", Image: "example/api:latest"},
			},
		},
		Status: corev1.PodStatus{Phase: corev1.PodRunning},
	}
}

func replicaSetOwner() k8sOwner {
	return newK8sOwner("default", metav1.OwnerReference{
		APIVersion: "apps/v1",
		Kind:       "ReplicaSet",
		Name:       "api-6f88b977f4",
		UID:        types.UID("rs-uid"),
	})
}

func TestRolloutRestartStampsPodTemplate(t *testing.T) {
	deployment := &appsv1.Deployment{Namespace: "default", Name: "api"}
	statefulSet := &appsv1.StatefulSet{Namespace: "default", Name: "db"}
	daemonSet := &appsv1.DaemonSet{Namespace: "default", Name: "agent"}
	client := &Client{Clientset: k8sfake.NewSimpleClientset(deployment, statefulSet, daemonSet)}
	apps := client.Clientset.AppsV1()

	require.NoError(t, client.RolloutRestart(t.Context(), "default", "Deployment", "api"))
	require.NoError(t, client.RolloutRestart(t.Context(), "default", "StatefulSet", "db"))
	require.NoError(t, client.RolloutRestart(t.Context(), "default", "DaemonSet", "agent"))

	d, err := apps.Deployments("default").Get(t.Context(), "api", metav1.GetOptions{})
	require.NoError(t, err)
	assert.NotEmpty(t, d.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"])
	s, err := apps.StatefulSets("default").Get(t.Context(), "db", metav1.GetOptions{})
	require.NoError(t, err)
	assert.NotEmpty(t, s.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"])
	ds, err := apps.DaemonSets("default").Get(t.Context(), "agent", metav1.GetOptions{})
	require.NoError(t, err)
	assert.NotEmpty(t, ds.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"])
}

func TestRolloutRestartRefusesOtherKinds(t *testing.T) {
	client := &Client{Clientset: k8sfake.NewSimpleClientset()}

	err := client.RolloutRestart(t.Context(), "default", "CronJob", "nightly")
	assert.ErrorIs(t, err, errors.ErrUnsupported)
}

func TestRolloutRestartReportsMissingWorkload(t *testing.T) {
	client := &Client{Clientset: k8sfake.NewSimpleClientset()}

	err := client.RolloutRestart(t.Context(), "default", "Deployment", "gone")
	assert.True(t, apierrors.IsNotFound(err))
}

func newTestK8sClient(t *testing.T, objects ...runtime.Object) *Client {
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, appsv1.AddToScheme(scheme))

	mapper := meta.NewDefaultRESTMapper([]schema.GroupVersion{appsv1.SchemeGroupVersion, corev1.SchemeGroupVersion})
	mapper.Add(appsv1.SchemeGroupVersion.WithKind("ReplicaSet"), meta.RESTScopeNamespace)
	mapper.Add(appsv1.SchemeGroupVersion.WithKind("Deployment"), meta.RESTScopeNamespace)

	return &Client{
		Clientset:     k8sfake.NewSimpleClientset(),
		DynamicClient: dynamicfake.NewSimpleDynamicClient(scheme, objects...),
		restMapper:    mapper,
		namespace:     []string{"default"},
		ownerCache:    make(map[string]ownerLookupResult),
	}
}

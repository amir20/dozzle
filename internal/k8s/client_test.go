package k8s

import (
	"context"
	"errors"
	"testing"

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
			TypeMeta: metav1.TypeMeta{APIVersion: "apps/v1", Kind: "ReplicaSet"},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "api-6f88b977f4",
				UID:       types.UID("rs-uid"),
				OwnerReferences: []metav1.OwnerReference{
					{APIVersion: "apps/v1", Kind: "Deployment", Name: "api", UID: types.UID("deploy-uid")},
				},
			},
		},
		&appsv1.Deployment{
			TypeMeta:   metav1.TypeMeta{APIVersion: "apps/v1", Kind: "Deployment"},
			ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "api", UID: types.UID("deploy-uid")},
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
	assert.Equal(t, "2", labels["k8s.owner.count"])
	assert.Equal(t, "2", labels["@k8s.owner.count"])

	assert.Equal(t, "ReplicaSet", labels["k8s.owner.0.kind"])
	assert.Equal(t, "ReplicaSet", labels["@k8s.owner.0.kind"])
	assert.Equal(t, "api-6f88b977f4", labels["k8s.owner.0.name"])
	assert.Equal(t, "ReplicaSet~default~api-6f88b977f4", labels["k8s.owner.0.key"])
	assert.Equal(t, "Deployment", labels["k8s.owner.1.kind"])
	assert.Equal(t, "Deployment", labels["@k8s.owner.1.kind"])
	assert.Equal(t, "api", labels["k8s.owner.1.name"])
	assert.Equal(t, "Deployment~default~api", labels["k8s.owner.1.key"])
	assert.Equal(t, "true", labels[ownerMembershipLabel("ReplicaSet~default~api-6f88b977f4")])
	assert.Equal(t, "true", labels[ownerMembershipLabel("Deployment~default~api")])
}

func TestPodToContainersStopsOwnerChainWhenOwnerCannotBeFetched(t *testing.T) {
	client := newTestK8sClient(t)

	containers := client.podToContainers(t.Context(), podWithOwner())
	require.Len(t, containers, 1)

	labels := containers[0].Labels
	assert.Equal(t, "1", labels["k8s.owner.count"])
	assert.Equal(t, "ReplicaSet", labels["k8s.owner.0.kind"])
	assert.Equal(t, "api-6f88b977f4", labels["k8s.owner.0.name"])
	assert.Empty(t, labels["k8s.owner.1.kind"])
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
	assert.Empty(t, labels["k8s.owner.count"])
	assert.Empty(t, labels["@k8s.owner.count"])
}

func TestPodToContainersStopsBeforeNodeOwnerInChain(t *testing.T) {
	client := newTestK8sClient(t,
		&appsv1.ReplicaSet{
			TypeMeta: metav1.TypeMeta{APIVersion: "apps/v1", Kind: "ReplicaSet"},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "api-6f88b977f4",
				UID:       types.UID("rs-uid"),
				OwnerReferences: []metav1.OwnerReference{
					{APIVersion: "v1", Kind: "Node", Name: "node-1", UID: types.UID("node-uid")},
				},
			},
		},
	)

	containers := client.podToContainers(t.Context(), podWithOwner())
	require.Len(t, containers, 1)

	labels := containers[0].Labels
	assert.Equal(t, "1", labels["k8s.owner.count"])
	assert.Equal(t, "ReplicaSet", labels["k8s.owner.0.kind"])
	assert.Empty(t, labels["k8s.owner.1.kind"])
}

func TestListContainersAppliesSyntheticOwnerFiltersAfterPodList(t *testing.T) {
	client := newTestK8sClient(t,
		&appsv1.ReplicaSet{
			TypeMeta: metav1.TypeMeta{APIVersion: "apps/v1", Kind: "ReplicaSet"},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "api-6f88b977f4",
				UID:       types.UID("rs-uid"),
				OwnerReferences: []metav1.OwnerReference{
					{APIVersion: "apps/v1", Kind: "Deployment", Name: "api", UID: types.UID("deploy-uid")},
				},
			},
		},
		&appsv1.Deployment{
			TypeMeta:   metav1.TypeMeta{APIVersion: "apps/v1", Kind: "Deployment"},
			ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "api", UID: types.UID("deploy-uid")},
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
		"k8s.owner.key.legacyabc123": {"true"},
	})

	assert.Equal(t, container.ContainerLabels{
		"app":                        {"api"},
		"team.example.com/component": {"backend"},
	}, podLabels)
	assert.Equal(t, container.ContainerLabels{
		"@k8s.namespace":             {"default"},
		"@k8s.owner.key.abc123":      {"true"},
		"not:a:kubernetes:label:key": {"value"},
		"k8s.owner.key.legacyabc123": {"true"},
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

func podWithOwner() *corev1.Pod {
	return &corev1.Pod{
		TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Pod"},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "api-6f88b977f4-pod",
			Labels:    map[string]string{"app": "api"},
			OwnerReferences: []metav1.OwnerReference{
				{APIVersion: "apps/v1", Kind: "ReplicaSet", Name: "api-6f88b977f4", UID: types.UID("rs-uid")},
			},
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

func newTestK8sClient(t *testing.T, objects ...runtime.Object) *K8sClient {
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, appsv1.AddToScheme(scheme))

	mapper := meta.NewDefaultRESTMapper([]schema.GroupVersion{appsv1.SchemeGroupVersion, corev1.SchemeGroupVersion})
	mapper.Add(appsv1.SchemeGroupVersion.WithKind("ReplicaSet"), meta.RESTScopeNamespace)
	mapper.Add(appsv1.SchemeGroupVersion.WithKind("Deployment"), meta.RESTScopeNamespace)

	return &K8sClient{
		Clientset:     k8sfake.NewSimpleClientset(),
		DynamicClient: dynamicfake.NewSimpleDynamicClient(scheme, objects...),
		restMapper:    mapper,
		namespace:     []string{"default"},
		ownerCache:    make(map[string]ownerLookupResult),
	}
}

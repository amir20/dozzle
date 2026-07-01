package k8s

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"sync"
	"time"

	"os"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/utils"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"

	"github.com/rs/zerolog/log"

	"github.com/samber/lo"
	lop "github.com/samber/lo/parallel"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
)

type K8sClient struct {
	Clientset     kubernetes.Interface
	DynamicClient dynamic.Interface
	restMapper    meta.RESTMapper
	namespace     []string
	config        *rest.Config
	host          container.Host
	ownerCacheMu  sync.Mutex
	ownerCache    map[string]ownerLookupResult
}

func NewK8sClient(namespace []string) (*K8sClient, error) {
	var config *rest.Config
	var err error

	if len(namespace) == 0 {
		namespace = []string{metav1.NamespaceAll}
	}

	// Check if we're running in cluster
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
		log.Info().Msg("Running in-cluster mode")
	} else {
		kubeconfig := os.Getenv("KUBECONFIG")
		if kubeconfig == "" {
			kubeconfig = os.Getenv("HOME") + "/.kube/config"
		}
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, err
		}
		log.Info().Msgf("Running in local mode with kubeconfig: %s", kubeconfig)

	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, err
	}

	nodes, err := clientset.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	if len(nodes.Items) == 0 {
		return nil, fmt.Errorf("nodes not found")
	}
	node := nodes.Items[0]

	return &K8sClient{
		Clientset:     clientset,
		DynamicClient: dynamicClient,
		restMapper:    restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(discoveryClient)),
		namespace:     namespace,
		config:        config,
		host: container.Host{
			ID:   node.Status.NodeInfo.MachineID,
			Name: node.Name,
		},
		ownerCache: make(map[string]ownerLookupResult),
	}, nil
}

type k8sOwner struct {
	APIVersion string
	Kind       string
	Namespace  string
	Name       string
	UID        string
	TypeKey    string
	Key        string
}

type ownerLookupResult struct {
	ownerReferences []metav1.OwnerReference
	found           bool
}

func (k *K8sClient) podToContainers(ctx context.Context, pod *corev1.Pod) []container.Container {
	started := time.Time{}
	if pod.Status.StartTime != nil {
		started = pod.Status.StartTime.Time
	}

	// Build labels map with pod labels, namespace, and owner reference
	labels := make(map[string]string)
	for k, v := range pod.Labels {
		labels[k] = v
	}
	labels["namespace"] = pod.Namespace
	labels["@k8s.namespace"] = pod.Namespace

	owners := k.resolveOwnerChain(ctx, pod.Namespace, pod.OwnerReferences)
	if len(owners) > 0 {
		labels["owner.kind"] = owners[0].Kind
		labels["owner.name"] = owners[0].Name
		labels["owner.key"] = owners[0].Key
		labels["k8s.owner.count"] = fmt.Sprintf("%d", len(owners))
		labels["@k8s.owner.count"] = fmt.Sprintf("%d", len(owners))
	}
	for i, owner := range owners {
		prefix := fmt.Sprintf("k8s.owner.%d.", i)
		syntheticPrefix := fmt.Sprintf("@k8s.owner.%d.", i)
		labels[prefix+"apiVersion"] = owner.APIVersion
		labels[prefix+"kind"] = owner.Kind
		labels[prefix+"namespace"] = owner.Namespace
		labels[prefix+"name"] = owner.Name
		labels[prefix+"uid"] = owner.UID
		labels[prefix+"key"] = owner.Key
		labels[syntheticPrefix+"apiVersion"] = owner.APIVersion
		labels[syntheticPrefix+"kind"] = owner.Kind
		labels[syntheticPrefix+"namespace"] = owner.Namespace
		labels[syntheticPrefix+"name"] = owner.Name
		labels[syntheticPrefix+"uid"] = owner.UID
		labels[syntheticPrefix+"key"] = owner.Key
		labels[ownerMembershipLabel(owner.Key)] = "true"
	}

	var containers []container.Container
	for _, c := range pod.Spec.Containers {
		containers = append(containers, container.Container{
			ID:          pod.Namespace + ":" + pod.Name + ":" + c.Name,
			Name:        pod.Name + "/" + c.Name,
			Image:       c.Image,
			Created:     pod.CreationTimestamp.Time,
			State:       phaseToState(pod.Status.Phase),
			StartedAt:   started,
			Command:     strings.Join(c.Command, " "),
			Host:        pod.Spec.NodeName,
			Tty:         c.TTY,
			Labels:      labels,
			Stats:       utils.NewRingBuffer[container.ContainerStat](300),
			FullyLoaded: true,
		})
	}
	return containers
}

func (k *K8sClient) resolveOwnerChain(ctx context.Context, namespace string, refs []metav1.OwnerReference) []k8sOwner {
	owners := make([]k8sOwner, 0)
	seen := make(map[string]struct{})

	for len(refs) > 0 {
		ref := ownerReferenceToFollow(refs)
		if isNodeOwnerReference(ref) {
			break
		}
		owner := newK8sOwner(namespace, ref)
		if _, ok := seen[owner.cacheKey()]; ok {
			break
		}
		seen[owner.cacheKey()] = struct{}{}
		owners = append(owners, owner)

		next, ok := k.lookupOwnerReferences(ctx, owner)
		if !ok {
			break
		}
		refs = next
	}

	return owners
}

func ownerReferenceToFollow(refs []metav1.OwnerReference) metav1.OwnerReference {
	for _, ref := range refs {
		if ref.Controller != nil && *ref.Controller {
			return ref
		}
	}
	return refs[0]
}

func isNodeOwnerReference(ref metav1.OwnerReference) bool {
	return ref.APIVersion == "v1" && ref.Kind == "Node"
}

func newK8sOwner(namespace string, ref metav1.OwnerReference) k8sOwner {
	typeKey := ownerTypeKey(ref.APIVersion, ref.Kind)
	// "~" is URL-safe and not allowed in Kubernetes resource names/namespaces,
	// so owner route keys stay readable without colliding with real names.
	key := fmt.Sprintf("%s~%s~%s", typeKey, namespace, ref.Name)
	return k8sOwner{
		APIVersion: ref.APIVersion,
		Kind:       ref.Kind,
		Namespace:  namespace,
		Name:       ref.Name,
		UID:        string(ref.UID),
		TypeKey:    typeKey,
		Key:        key,
	}
}

func (o k8sOwner) cacheKey() string {
	return fmt.Sprintf("%s/%s/%s/%s/%s", o.APIVersion, o.Kind, o.Namespace, o.Name, o.UID)
}

func ownerMembershipLabel(key string) string {
	return "@k8s.owner.key." + base64.RawURLEncoding.EncodeToString([]byte(key))
}

func ownerTypeKey(apiVersion, kind string) string {
	if isKnownK8sOwnerType(apiVersion, kind) {
		return kind
	}
	return strings.ReplaceAll(apiVersion, "/", "~") + "~" + kind
}

func isKnownK8sOwnerType(apiVersion, kind string) bool {
	switch apiVersion + "/" + kind {
	case "apps/v1/Deployment",
		"apps/v1/ReplicaSet",
		"apps/v1/DaemonSet",
		"apps/v1/StatefulSet",
		"batch/v1/Job",
		"batch/v1/CronJob",
		"v1/Pod",
		"v1/Service",
		"v1/ConfigMap",
		"v1/Secret":
		return true
	default:
		return false
	}
}

func (k *K8sClient) lookupOwnerReferences(ctx context.Context, owner k8sOwner) ([]metav1.OwnerReference, bool) {
	cacheKey := owner.cacheKey()
	k.ownerCacheMu.Lock()
	if k.ownerCache == nil {
		k.ownerCache = make(map[string]ownerLookupResult)
	}
	if result, ok := k.ownerCache[cacheKey]; ok {
		k.ownerCacheMu.Unlock()
		return result.ownerReferences, result.found
	}
	k.ownerCacheMu.Unlock()

	refs, ok, cacheable := k.fetchOwnerReferences(ctx, owner)

	if cacheable {
		k.ownerCacheMu.Lock()
		k.ownerCache[cacheKey] = ownerLookupResult{ownerReferences: refs, found: ok}
		k.ownerCacheMu.Unlock()
	}

	return refs, ok
}

func (k *K8sClient) fetchOwnerReferences(ctx context.Context, owner k8sOwner) ([]metav1.OwnerReference, bool, bool) {
	if k.DynamicClient == nil || k.restMapper == nil {
		return nil, false, false
	}

	groupVersion, err := schema.ParseGroupVersion(owner.APIVersion)
	if err != nil {
		log.Debug().Err(err).Str("owner", owner.Key).Msg("failed to parse owner apiVersion")
		return nil, false, false
	}

	mapping, err := k.restMapper.RESTMapping(groupVersion.WithKind(owner.Kind).GroupKind(), groupVersion.Version)
	if err != nil {
		log.Debug().Err(err).Str("owner", owner.Key).Msg("failed to map owner resource")
		return nil, false, false
	}

	var resource dynamic.ResourceInterface
	if mapping.Scope.Name() != meta.RESTScopeNameRoot {
		resource = k.DynamicClient.Resource(mapping.Resource).Namespace(owner.Namespace)
	} else {
		resource = k.DynamicClient.Resource(mapping.Resource)
	}

	obj, err := resource.Get(ctx, owner.Name, metav1.GetOptions{})
	if err != nil {
		log.Debug().Err(err).Str("owner", owner.Key).Msg("failed to fetch owner resource")
		if ctx.Err() != nil {
			return nil, false, false
		}
		return nil, false, apierrors.IsNotFound(err) || apierrors.IsForbidden(err)
	}

	return obj.GetOwnerReferences(), true, true
}

func splitK8sFilters(labels container.ContainerLabels) (container.ContainerLabels, container.ContainerLabels) {
	podLabels := make(container.ContainerLabels)
	metadataLabels := make(container.ContainerLabels)
	for key, values := range labels {
		if isK8sMetadataLabel(key) || !isValidK8sLabelKey(key) {
			metadataLabels[key] = values
		} else {
			podLabels[key] = values
		}
	}
	return podLabels, metadataLabels
}

func isK8sMetadataLabel(key string) bool {
	if strings.HasPrefix(key, "@k8s.") {
		return true
	}
	return key == "namespace" ||
		key == "owner.kind" ||
		key == "owner.name" ||
		key == "owner.key" ||
		strings.HasPrefix(key, "k8s.owner.")
}

var (
	k8sLabelNamePattern   = regexp.MustCompile(`^[A-Za-z0-9]([A-Za-z0-9_.-]{0,61}[A-Za-z0-9])?$`)
	k8sLabelPrefixPattern = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`)
)

func isValidK8sLabelKey(key string) bool {
	prefix, name, hasPrefix := strings.Cut(key, "/")
	if !hasPrefix {
		name = prefix
	} else if len(prefix) == 0 || len(prefix) > 253 || !k8sLabelPrefixPattern.MatchString(prefix) {
		return false
	}
	return len(name) <= 63 && k8sLabelNamePattern.MatchString(name)
}

func matchesContainerLabels(labels map[string]string, filters container.ContainerLabels) bool {
	for key, values := range filters {
		value, ok := labels[key]
		if !ok {
			return false
		}
		matched := false
		for _, expected := range values {
			if value == expected {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	return true
}

func (k *K8sClient) ListContainers(ctx context.Context, labels container.ContainerLabels) ([]container.Container, error) {
	podLabels, metadataLabels := splitK8sFilters(labels)
	selector := ""
	if podLabels.Exists() {
		for key, values := range podLabels {
			for _, value := range values {
				if selector != "" {
					selector += ","
				}
				selector += fmt.Sprintf("%s=%s", key, value)
			}
		}
		log.Debug().Str("selector", selector).Msg("Listing containers with labels")
	}
	containerList := lop.Map(k.namespace, func(namespace string, index int) lo.Tuple2[[]container.Container, error] {
		pods, err := k.Clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
		if err != nil {
			return lo.T2[[]container.Container, error](nil, fmt.Errorf("failed to list pods in namespace %s: %w", namespace, err))
		}
		var containers []container.Container
		for _, pod := range pods.Items {
			for _, c := range k.podToContainers(ctx, &pod) {
				if metadataLabels.Exists() && !matchesContainerLabels(c.Labels, metadataLabels) {
					continue
				}
				containers = append(containers, c)
			}
		}
		return lo.T2[[]container.Container, error](containers, nil)
	})

	var containers []container.Container
	var lastError error
	success := false
	for _, t2 := range containerList {
		items, err := t2.Unpack()
		if err != nil {
			log.Error().Err(err).Msg("failed to fetch containers")
			lastError = err
			continue
		}
		success = true
		containers = append(containers, items...)
	}

	if !success {
		return nil, lastError
	}

	return containers, nil
}

func phaseToState(phase corev1.PodPhase) string {
	switch phase {
	case corev1.PodPending:
		return "created"
	case corev1.PodRunning:
		return "running"
	case corev1.PodSucceeded:
		return "exited"
	case corev1.PodFailed:
		return "exited"
	case corev1.PodUnknown:
		return "unknown"
	default:
		return "unknown"
	}
}

func (k *K8sClient) FindContainer(ctx context.Context, id string) (container.Container, error) {
	log.Debug().Str("id", id).Msg("Finding container")
	namespace, podName, containerName := parsePodContainerID(id)

	pod, err := k.Clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return container.Container{}, err
	}

	for _, c := range k.podToContainers(ctx, pod) {
		if c.ID == id {
			return c, nil
		}
	}

	return container.Container{}, fmt.Errorf("container %s not found in pod %s", containerName, podName)
}

func (k *K8sClient) ContainerLogs(ctx context.Context, id string, since time.Time, stdType container.StdType) (io.ReadCloser, error) {
	namespace, podName, containerName := parsePodContainerID(id)

	var lines int64 = 500
	opts := &corev1.PodLogOptions{
		Container:  containerName,
		Follow:     true,
		Previous:   false,
		Timestamps: true,
		SinceTime:  &metav1.Time{Time: since},
		TailLines:  &lines,
	}

	return k.Clientset.CoreV1().Pods(namespace).GetLogs(podName, opts).Stream(ctx)
}

func (k *K8sClient) ContainerLogsBetweenDates(ctx context.Context, id string, start time.Time, end time.Time, stdType container.StdType) (io.ReadCloser, error) {
	namespace, podName, containerName := parsePodContainerID(id)

	opts := &corev1.PodLogOptions{
		Container:  containerName,
		Follow:     false,
		Timestamps: true,
		SinceTime:  &metav1.Time{Time: start},
	}

	return k.Clientset.CoreV1().Pods(namespace).GetLogs(podName, opts).Stream(ctx)
}

func (k *K8sClient) ContainerEvents(ctx context.Context, ch chan<- container.ContainerEvent) error {
	watchers := lo.Map(k.namespace, func(namespace string, index int) watch.Interface {
		watcher, err := k.Clientset.CoreV1().Pods(namespace).Watch(ctx, metav1.ListOptions{})
		if err != nil {
			log.Error().Err(err).Msg("Failed to watch pods")
			return nil
		}
		return watcher
	})

	if len(watchers) == 0 {
		return errors.New("no namespaces to watch")
	}

	wg := sync.WaitGroup{}

	for _, watcher := range watchers {
		wg.Go(func() {
			for event := range watcher.ResultChan() {
				log.Debug().Interface("event.type", event.Type).Msg("Received kubernetes event")
				pod, ok := event.Object.(*corev1.Pod)
				if !ok {
					continue
				}

				name := ""
				switch event.Type {
				case "ADDED":
					name = "create"
				case "DELETED":
					name = "destroy"
				case "MODIFIED":
					name = "update"
				}

				for _, c := range k.podToContainers(ctx, pod) {
					ch <- container.ContainerEvent{
						Name:      name,
						ActorID:   c.ID,
						Host:      pod.Spec.NodeName,
						Time:      time.Now(),
						Container: &c,
					}
				}
			}
		})
	}

	wg.Wait()

	return nil
}

func (k *K8sClient) ContainerStats(ctx context.Context, id string, stats chan<- container.ContainerStat) error {
	// Stats collection is implemented in stats_collector.go using K8s metrics API
	panic("not implemented - use K8sStatsCollector instead")
}

func (k *K8sClient) Ping(ctx context.Context) error {
	_, err := k.Clientset.CoreV1().Pods("default").List(ctx, metav1.ListOptions{Limit: 1})
	return err
}

func (k *K8sClient) Host() container.Host {
	return k.host
}

func (k *K8sClient) ContainerActions(ctx context.Context, action container.ContainerAction, containerID string) error {
	panic("not implemented")
}

func (k *K8sClient) ContainerAttach(ctx context.Context, id string) (*container.ExecSession, error) {
	namespace, podName, containerName := parsePodContainerID(id)
	log.Debug().Str("container", containerName).Str("pod", podName).Msg("Attaching to pod")
	req := k.Clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("attach")

	option := &corev1.PodAttachOptions{
		Container: containerName,
		Stdin:     true,
		Stdout:    true,
		Stderr:    true,
		TTY:       true,
	}

	req.VersionedParams(
		option,
		scheme.ParameterCodec,
	)

	exec, err := remotecommand.NewSPDYExecutor(k.config, "POST", req.URL())
	if err != nil {
		return nil, err
	}

	stdinReader, stdinWriter := io.Pipe()
	stdoutReader, stdoutWriter := io.Pipe()

	// Create TerminalSizeQueue for dynamic resizing
	sizeQueue := &terminalSizeQueue{
		resizeChan: make(chan remotecommand.TerminalSize, 1),
	}

	go func() {
		err := exec.StreamWithContext(ctx, remotecommand.StreamOptions{
			Stdin:             stdinReader,
			Stdout:            stdoutWriter,
			Tty:               true,
			TerminalSizeQueue: sizeQueue,
		})
		if err != nil {
			log.Error().Err(err).Msg("Error streaming command")
		}
	}()

	// Create resize closure that sends to the queue
	resizeFn := func(width uint, height uint) error {
		select {
		case sizeQueue.resizeChan <- remotecommand.TerminalSize{Width: uint16(width), Height: uint16(height)}:
			return nil
		default:
			return fmt.Errorf("resize queue full")
		}
	}

	return &container.ExecSession{
		Writer: stdinWriter,
		Reader: stdoutReader,
		Resize: resizeFn,
	}, nil
}

// terminalSizeQueue implements remotecommand.TerminalSizeQueue
type terminalSizeQueue struct {
	resizeChan chan remotecommand.TerminalSize
}

func (t *terminalSizeQueue) Next() *remotecommand.TerminalSize {
	size, ok := <-t.resizeChan
	if !ok {
		return nil
	}
	return &size
}

func (k *K8sClient) ContainerExec(ctx context.Context, id string, cmd []string) (*container.ExecSession, error) {
	namespace, podName, containerName := parsePodContainerID(id)
	log.Debug().Str("container", containerName).Str("pod", podName).Msg("Executing command in pod")
	req := k.Clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec")

	option := &corev1.PodExecOptions{
		Command:   cmd,
		Container: containerName,
		Stdin:     true,
		Stdout:    true,
		Stderr:    true,
		TTY:       true,
	}

	req.VersionedParams(
		option,
		scheme.ParameterCodec,
	)

	exec, err := remotecommand.NewSPDYExecutor(k.config, "POST", req.URL())
	if err != nil {
		return nil, err
	}

	stdinReader, stdinWriter := io.Pipe()
	stdoutReader, stdoutWriter := io.Pipe()

	// Create TerminalSizeQueue for dynamic resizing
	sizeQueue := &terminalSizeQueue{
		resizeChan: make(chan remotecommand.TerminalSize, 1),
	}

	go func() {
		err := exec.StreamWithContext(ctx, remotecommand.StreamOptions{
			Stdin:             stdinReader,
			Stdout:            stdoutWriter,
			Tty:               true,
			TerminalSizeQueue: sizeQueue,
		})
		if err != nil {
			log.Error().Err(err).Msg("Error streaming command")
		}
	}()

	// Create resize closure that sends to the queue
	resizeFn := func(width uint, height uint) error {
		select {
		case sizeQueue.resizeChan <- remotecommand.TerminalSize{Width: uint16(width), Height: uint16(height)}:
			return nil
		default:
			return fmt.Errorf("resize queue full")
		}
	}

	return &container.ExecSession{
		Writer: stdinWriter,
		Reader: stdoutReader,
		Resize: resizeFn,
	}, nil
}

// Helper function to parse pod and container names from container ID
func parsePodContainerID(id string) (string, string, string) {
	parts := strings.Split(id, ":")
	return parts[0], parts[1], parts[2]
}

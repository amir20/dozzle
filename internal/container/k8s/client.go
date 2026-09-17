package k8s

import (
	"bufio"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"maps"
	"regexp"
	"slices"
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
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	coreinformers "k8s.io/client-go/informers/core/v1"

	"github.com/rs/zerolog/log"

	"github.com/samber/lo"
	lop "github.com/samber/lo/parallel"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
)

type Client struct {
	Clientset     kubernetes.Interface
	DynamicClient dynamic.Interface
	restMapper    meta.RESTMapper
	namespace     []string
	labelSelector string // pod-label half of --filter, pushed down to the API server
	config        *rest.Config
	host          container.Host
	ownerCacheMu  sync.Mutex
	ownerCache    map[string]ownerLookupResult

	mapperMu        sync.Mutex
	lastMapperReset time.Time
}

// hostIDs decides what this node is called; see container.HostIDResolver. filter is
// --filter: its pod-label entries narrow what is listed and watched at the API server.
func NewClient(namespace []string, filter container.ContainerLabels, hostIDs container.HostIDResolver) (*Client, error) {
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

	return &Client{
		Clientset:     clientset,
		DynamicClient: dynamicClient,
		restMapper:    restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(discoveryClient)),
		namespace:     namespace,
		labelSelector: podLabelSelector(filter),
		config:        config,
		host: container.Host{
			ID:   hostIDs.Resolve(container.EngineIdentity{Runtime: "k8s", EngineID: node.Status.NodeInfo.MachineID}),
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
	expiresAt       time.Time
}

const (
	// TTL bounds owner-cache staleness so entries recover after RBAC is granted
	// and let expired entries be swept, keeping the cache from growing forever.
	ownerCacheTTL = 5 * time.Minute
	// Hard cap so a cluster with heavy ReplicaSet churn can't grow the cache unbounded.
	ownerCacheMaxSize = 4096
	// Evict down to this size once the cap is hit, so eviction has hysteresis and
	// doesn't re-run on every subsequent insert.
	ownerCacheEvictTo = ownerCacheMaxSize * 3 / 4
	// Discovery resets are throttled to this interval so an unmappable owner kind
	// (typo, uninstalled CRD) can't hammer the API server's discovery endpoint.
	mapperResetInterval = time.Minute
)

func (k *Client) podToContainers(ctx context.Context, pod *corev1.Pod) []container.Container {
	started := time.Time{}
	if pod.Status.StartTime != nil {
		started = pod.Status.StartTime.Time
	}

	// Build labels map with pod labels, namespace, and owner reference
	labels := make(map[string]string)
	maps.Copy(labels, pod.Labels)
	labels["namespace"] = pod.Namespace
	labels["@k8s.namespace"] = pod.Namespace

	owners := k.resolveOwnerChain(ctx, pod.Namespace, pod.OwnerReferences)
	if len(owners) > 0 {
		// Immediate owner kept as top-level labels for backward compatibility.
		labels["owner.kind"] = owners[0].Kind
		labels["owner.name"] = owners[0].Name
		labels["owner.key"] = owners[0].Key
		labels["@k8s.owner.count"] = fmt.Sprintf("%d", len(owners))
	}
	for i, owner := range owners {
		prefix := fmt.Sprintf("@k8s.owner.%d.", i)
		labels[prefix+"apiVersion"] = owner.APIVersion
		labels[prefix+"kind"] = owner.Kind
		labels[prefix+"namespace"] = owner.Namespace
		labels[prefix+"name"] = owner.Name
		labels[prefix+"uid"] = owner.UID
		labels[prefix+"key"] = owner.Key
		labels[ownerMembershipLabel(owner.Key)] = "true"
	}

	statuses := make(map[string]corev1.ContainerStatus, len(pod.Status.InitContainerStatuses)+len(pod.Status.ContainerStatuses))
	for _, status := range pod.Status.InitContainerStatuses {
		statuses[status.Name] = status
	}
	for _, status := range pod.Status.ContainerStatuses {
		statuses[status.Name] = status
	}

	// Init containers first, in the order they run. A failing one is usually why a pod
	// is stuck, and a native sidecar (an init container with restartPolicy Always)
	// runs for the pod's whole life, so both need to be viewable.
	var initLabels map[string]string
	if len(pod.Spec.InitContainers) > 0 {
		initLabels = maps.Clone(labels)
		initLabels["@k8s.init"] = "true"
	}

	containers := make([]container.Container, 0, len(pod.Spec.InitContainers)+len(pod.Spec.Containers))
	add := func(c corev1.Container, labels map[string]string) {
		state, containerStarted, finished := phaseToState(pod.Status.Phase), started, time.Time{}
		if status, ok := statuses[c.Name]; ok {
			state, containerStarted, finished = containerStatusToState(status, started)
		}
		containers = append(containers, container.Container{
			ID:          pod.Namespace + ":" + pod.Name + ":" + c.Name,
			Name:        pod.Name + "/" + c.Name,
			Image:       c.Image,
			Created:     pod.CreationTimestamp.Time,
			State:       state,
			StartedAt:   containerStarted,
			FinishedAt:  finished,
			Command:     strings.Join(c.Command, " "),
			Host:        pod.Spec.NodeName,
			Tty:         c.TTY,
			Labels:      labels,
			Stats:       utils.NewRingBuffer[container.ContainerStat](300),
			FullyLoaded: true,
		})
	}
	for _, c := range pod.Spec.InitContainers {
		add(c, initLabels)
	}
	for _, c := range pod.Spec.Containers {
		add(c, labels)
	}
	return containers
}

// containerStatusToState reads one container's own status. The pod phase stays
// Running while a container crash-loops or a sidecar has died, so it cannot say
// whether this particular container is healthy.
// Like Docker, StartedAt is when the latest run started, not the pod.
func containerStatusToState(status corev1.ContainerStatus, podStarted time.Time) (state string, started time.Time, finished time.Time) {
	switch {
	case status.State.Running != nil:
		return "running", status.State.Running.StartedAt.Time, time.Time{}
	case status.State.Terminated != nil:
		t := status.State.Terminated
		return "exited", t.StartedAt.Time, t.FinishedAt.Time
	case status.State.Waiting != nil && status.LastTerminationState.Terminated != nil:
		// Waiting after having run before is a crash loop (CrashLoopBackOff), which
		// Docker calls restarting.
		t := status.LastTerminationState.Terminated
		return "restarting", t.StartedAt.Time, t.FinishedAt.Time
	default:
		return "created", podStarted, time.Time{}
	}
}

func (k *Client) resolveOwnerChain(ctx context.Context, namespace string, refs []metav1.OwnerReference) []k8sOwner {
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

func (k *Client) lookupOwnerReferences(ctx context.Context, owner k8sOwner) ([]metav1.OwnerReference, bool) {
	cacheKey := owner.cacheKey()
	now := time.Now()

	k.ownerCacheMu.Lock()
	if k.ownerCache == nil {
		k.ownerCache = make(map[string]ownerLookupResult)
	}
	if result, ok := k.ownerCache[cacheKey]; ok && now.Before(result.expiresAt) {
		k.ownerCacheMu.Unlock()
		return result.ownerReferences, result.found
	}
	k.ownerCacheMu.Unlock()

	refs, ok, cacheable := k.fetchOwnerReferences(ctx, owner)

	if cacheable {
		k.ownerCacheMu.Lock()
		k.pruneOwnerCache(now)
		k.ownerCache[cacheKey] = ownerLookupResult{ownerReferences: refs, found: ok, expiresAt: now.Add(ownerCacheTTL)}
		k.ownerCacheMu.Unlock()
	}

	return refs, ok
}

// pruneOwnerCache drops expired entries and, only once the cache grows past
// ownerCacheMaxSize, evicts arbitrary entries down to ownerCacheEvictTo. This
// hysteresis (grow to max, drop to evictTo) bounds growth without evicting on
// every insert, and avoids clearing the map wholesale so a cluster with more
// than ownerCacheMaxSize live owners doesn't thrash. Callers must hold ownerCacheMu.
func (k *Client) pruneOwnerCache(now time.Time) {
	for key, result := range k.ownerCache {
		if !now.Before(result.expiresAt) {
			delete(k.ownerCache, key)
		}
	}
	if len(k.ownerCache) <= ownerCacheMaxSize {
		return
	}
	// Map iteration order is randomized, so this evicts an arbitrary subset.
	for key := range k.ownerCache {
		if len(k.ownerCache) <= ownerCacheEvictTo {
			break
		}
		delete(k.ownerCache, key)
	}
}

// resetRESTMapper resets the cached discovery mapper so newly-registered CRDs
// can be mapped, at most once per mapperResetInterval. Returns true if it reset.
func (k *Client) resetRESTMapper() bool {
	resetter, ok := k.restMapper.(interface{ Reset() })
	if !ok {
		return false
	}

	k.mapperMu.Lock()
	defer k.mapperMu.Unlock()
	now := time.Now()
	if !k.lastMapperReset.IsZero() && now.Sub(k.lastMapperReset) < mapperResetInterval {
		return false
	}
	k.lastMapperReset = now
	resetter.Reset()
	return true
}

func (k *Client) fetchOwnerReferences(ctx context.Context, owner k8sOwner) ([]metav1.OwnerReference, bool, bool) {
	if k.DynamicClient == nil || k.restMapper == nil {
		return nil, false, false
	}

	groupVersion, err := schema.ParseGroupVersion(owner.APIVersion)
	if err != nil {
		log.Debug().Err(err).Str("owner", owner.Key).Msg("failed to parse owner apiVersion")
		return nil, false, false
	}

	gk := groupVersion.WithKind(owner.Kind).GroupKind()
	mapping, err := k.restMapper.RESTMapping(gk, groupVersion.Version)
	if err != nil {
		// Discovery is cached, so a CRD registered after startup won't map until the
		// cache is reset. Reset and retry, but throttle resets so a permanently
		// unmappable kind can't hammer the discovery endpoint on every pod event.
		if k.resetRESTMapper() {
			mapping, err = k.restMapper.RESTMapping(gk, groupVersion.Version)
		}
		if err != nil {
			log.Debug().Err(err).Str("owner", owner.Key).Msg("failed to map owner resource")
			return nil, false, false
		}
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
		key == "owner.key"
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
		matched := slices.Contains(values, value)
		if !matched {
			return false
		}
	}
	return true
}

// podLabelSelector turns the pod-label entries of a filter into a label selector.
// Metadata entries (namespace, owner) are not pod labels and are matched after listing.
func podLabelSelector(labels container.ContainerLabels) string {
	podLabels, _ := splitK8sFilters(labels)
	var parts []string
	for key, values := range podLabels {
		for _, value := range values {
			parts = append(parts, fmt.Sprintf("%s=%s", key, value))
		}
	}
	slices.Sort(parts)
	return strings.Join(parts, ",")
}

func (k *Client) ListContainers(ctx context.Context, labels container.ContainerLabels) ([]container.Container, error) {
	_, metadataLabels := splitK8sFilters(labels)
	selector := podLabelSelector(labels)
	if selector != "" {
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

func (k *Client) FindContainer(ctx context.Context, id string) (container.Container, error) {
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

func (k *Client) ContainerLogs(ctx context.Context, id string, since time.Time, stdType container.StdType) (io.ReadCloser, error) {
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

func (k *Client) ContainerLogsBetweenDates(ctx context.Context, id string, start time.Time, end time.Time, stdType container.StdType) (io.ReadCloser, error) {
	namespace, podName, containerName := parsePodContainerID(id)

	opts := &corev1.PodLogOptions{
		Container:  containerName,
		Follow:     false,
		Timestamps: true,
		SinceTime:  &metav1.Time{Time: start},
	}

	pods := k.Clientset.CoreV1().Pods(namespace)
	current, err := pods.GetLogs(podName, opts).Stream(ctx)
	if err != nil {
		return nil, err
	}

	// A restarted container keeps the same ID, but the logs API only serves the current
	// run unless asked for the previous one. That run is what explains a crash loop, so
	// put it in front. With no previous run the API returns an error, and when the
	// runtime has already pruned it the kubelet answers 200 with "unable to retrieve
	// container logs" as the body. Every real line starts with a timestamp, so anything
	// else is dropped.
	previousOpts := *opts
	previousOpts.Previous = true
	previous, err := pods.GetLogs(podName, &previousOpts).Stream(ctx)
	if err != nil {
		return current, nil
	}
	buffered := bufio.NewReader(previous)
	if head, _ := buffered.Peek(len(time.RFC3339)); !startsWithTimestamp(head) {
		previous.Close()
		return current, nil
	}

	return &multiReadCloser{
		Reader:  io.MultiReader(&newlineTerminated{r: buffered}, current),
		closers: []io.Closer{previous, current},
	}, nil
}

func startsWithTimestamp(b []byte) bool {
	_, err := time.Parse("2006-01-02T15:04:05", string(b[:min(len(b), len("2006-01-02T15:04:05"))]))
	return err == nil
}

// newlineTerminated makes sure the previous run ends on a line break, so its last
// line is not glued to the first line of the current run.
type newlineTerminated struct {
	r    io.Reader
	last byte
	eof  bool
}

func (n *newlineTerminated) Read(p []byte) (int, error) {
	if n.eof {
		if n.last != 0 && n.last != '\n' && len(p) > 0 {
			p[0], n.last = '\n', '\n'
			return 1, io.EOF
		}
		return 0, io.EOF
	}
	read, err := n.r.Read(p)
	if read > 0 {
		n.last = p[read-1]
	}
	if err == io.EOF {
		n.eof = true
		if read == 0 {
			return n.Read(p)
		}
		return read, nil
	}
	return read, err
}

type multiReadCloser struct {
	io.Reader
	closers []io.Closer
}

func (m *multiReadCloser) Close() error {
	var errs []error
	for _, c := range m.closers {
		errs = append(errs, c.Close())
	}
	return errors.Join(errs...)
}

// ContainerEvents streams pod changes until ctx is done. It uses an informer rather
// than a bare watch: the API server closes every watch after 30 to 60 minutes, and a
// bare watch that ends leaves Dozzle deaf to new pods (and their alerts) until
// something happens to list containers again. The informer re-establishes the watch
// and re-lists after a gap, so a missed delete still arrives.
func (k *Client) ContainerEvents(ctx context.Context, ch chan<- container.ContainerEvent) error {
	if len(k.namespace) == 0 {
		return errors.New("no namespaces to watch")
	}

	wg := sync.WaitGroup{}
	for _, namespace := range k.namespace {
		informer := k.newPodInformer(namespace)
		if _, err := informer.AddEventHandlerWithOptions(k.podEventHandler(ctx, ch), cache.HandlerOptions{}); err != nil {
			return fmt.Errorf("failed to watch pods in namespace %s: %w", namespace, err)
		}
		wg.Go(func() { informer.RunWithContext(ctx) })
	}
	wg.Wait()

	return ctx.Err()
}

func (k *Client) newPodInformer(namespace string) cache.SharedIndexInformer {
	return coreinformers.NewFilteredPodInformer(k.Clientset, namespace, 0, cache.Indexers{}, func(options *metav1.ListOptions) {
		options.LabelSelector = k.labelSelector
	})
}

func (k *Client) podEventHandler(ctx context.Context, ch chan<- container.ContainerEvent) cache.ResourceEventHandler {
	send := func(name string, pod *corev1.Pod) {
		log.Debug().Str("event", name).Str("pod", pod.Name).Msg("Received kubernetes event")
		for _, c := range k.podToContainers(ctx, pod) {
			select {
			case ch <- container.ContainerEvent{
				Name:      name,
				ActorID:   c.ID,
				Host:      pod.Spec.NodeName,
				Time:      time.Now(),
				Container: &c,
			}:
			case <-ctx.Done():
				return
			}
		}
	}

	return cache.ResourceEventHandlerDetailedFuncs{
		AddFunc: func(obj any, isInInitialList bool) {
			pod, ok := obj.(*corev1.Pod)
			if !ok {
				return
			}
			// The store lists pods itself when it connects, so the informer's own first
			// list is sent as updates: cheap for pods the store already has, and it
			// catches one created between the two lists.
			if isInInitialList {
				send("update", pod)
			} else {
				send("create", pod)
			}
		},
		UpdateFunc: func(_, obj any) {
			if pod, ok := obj.(*corev1.Pod); ok {
				send("update", pod)
			}
		},
		DeleteFunc: func(obj any) {
			if tombstone, ok := obj.(cache.DeletedFinalStateUnknown); ok {
				obj = tombstone.Obj
			}
			if pod, ok := obj.(*corev1.Pod); ok {
				send("destroy", pod)
			}
		},
	}
}

// ContainerStats is not used in k8s mode: StatsCollector polls the metrics API for
// every pod at once instead of streaming per container.
func (k *Client) ContainerStats(ctx context.Context, id string, stats chan<- container.ContainerStat) error {
	return fmt.Errorf("per-container stats are not supported in Kubernetes mode: %w", errors.ErrUnsupported)
}

func (k *Client) Ping(ctx context.Context) error {
	_, err := k.Clientset.CoreV1().Pods("default").List(ctx, metav1.ListOptions{Limit: 1})
	return err
}

func (k *Client) Host() container.Host {
	return k.host
}

// ContainerActions supports restart only. Kubernetes has no stop or start for a
// single container; what it does have is deleting a pod so its controller schedules a
// fresh one, which is what `kubectl delete pod` is used for day to day. A pod nothing
// controls would be gone for good, so that is refused rather than done.
func (k *Client) ContainerActions(ctx context.Context, action container.ContainerAction, containerID string) error {
	if action != container.Restart {
		return fmt.Errorf("%s is not supported in Kubernetes mode, only restart: %w", action, errors.ErrUnsupported)
	}

	namespace, podName, _ := parsePodContainerID(containerID)
	pods := k.Clientset.CoreV1().Pods(namespace)
	pod, err := pods.Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	// A static pod's mirror is "controlled" by its Node, but deleting the mirror only
	// recreates the API object; the kubelet never restarts the containers.
	if controller := metav1.GetControllerOf(pod); controller == nil || controller.Kind == "Node" {
		return fmt.Errorf("pod %s has no controller to recreate it, so it cannot be restarted: %w", podName, errors.ErrUnsupported)
	}

	log.Info().Str("pod", podName).Str("namespace", namespace).Msg("restarting pod by deleting it")
	return pods.Delete(ctx, podName, metav1.DeleteOptions{
		Preconditions: &metav1.Preconditions{UID: &pod.UID},
	})
}

func (k *Client) ContainerAttach(ctx context.Context, id string) (*container.ExecSession, error) {
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

func (k *Client) ContainerExec(ctx context.Context, id string, cmd []string) (*container.ExecSession, error) {
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

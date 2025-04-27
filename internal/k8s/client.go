package k8s

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"os"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/rs/zerolog/log"

	"github.com/samber/lo"
	lop "github.com/samber/lo/parallel"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
)

type K8sClient struct {
	Clientset *kubernetes.Clientset
	namespace []string
	config    *rest.Config
	host      container.Host
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

	nodes, err := clientset.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	if len(nodes.Items) == 0 {
		return nil, fmt.Errorf("nodes not found")
	}
	node := nodes.Items[0]

	return &K8sClient{
		Clientset: clientset,
		namespace: namespace,
		config:    config,
		host: container.Host{
			ID:   node.Status.NodeInfo.MachineID,
			Name: node.Name,
		},
	}, nil
}

func podToContainers(pod *corev1.Pod) []container.Container {
	started := time.Time{}
	if pod.Status.StartTime != nil {
		started = pod.Status.StartTime.Time
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
			Stats:       utils.NewRingBuffer[container.ContainerStat](300),
			FullyLoaded: true,
		})
	}
	return containers
}

func (k *K8sClient) ListContainers(ctx context.Context, labels container.ContainerLabels) ([]container.Container, error) {
	selector := ""
	if labels.Exists() {
		for key, values := range labels {
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
			containers = append(containers, podToContainers(&pod)...)
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

	for _, c := range podToContainers(pod) {
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
	wg.Add(len(watchers))

	for _, watcher := range watchers {
		go func(w watch.Interface) {
			defer wg.Done()
			for event := range w.ResultChan() {
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

				for _, c := range podToContainers(pod) {
					ch <- container.ContainerEvent{
						Name:      name,
						ActorID:   c.ID,
						Host:      pod.Spec.NodeName,
						Time:      time.Now(),
						Container: &c,
					}
				}
			}
		}(watcher)
	}

	wg.Wait()

	return nil
}

func (k *K8sClient) ContainerStats(ctx context.Context, id string, stats chan<- container.ContainerStat) error {
	panic("not implemented")
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

func (k *K8sClient) ContainerAttach(ctx context.Context, id string) (io.WriteCloser, io.Reader, error) {
	namespace, podName, containerName := parsePodContainerID(id)
	log.Debug().Str("container", containerName).Str("pod", podName).Msg("Executing command in pod")
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
		return nil, nil, err
	}

	stdinReader, stdinWriter := io.Pipe()
	stdoutReader, stdoutWriter := io.Pipe()

	go func() {
		err := exec.StreamWithContext(ctx, remotecommand.StreamOptions{
			Stdin:  stdinReader,
			Stdout: stdoutWriter,
			Tty:    true,
		})
		if err != nil {
			log.Error().Err(err).Msg("Error streaming command")
		}
	}()

	return stdinWriter, stdoutReader, nil
}

func (k *K8sClient) ContainerExec(ctx context.Context, id string, cmd []string) (io.WriteCloser, io.Reader, error) {
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
		return nil, nil, err
	}

	stdinReader, stdinWriter := io.Pipe()
	stdoutReader, stdoutWriter := io.Pipe()

	go func() {
		err := exec.StreamWithContext(ctx, remotecommand.StreamOptions{
			Stdin:  stdinReader,
			Stdout: stdoutWriter,
			Tty:    true,
		})
		if err != nil {
			log.Error().Err(err).Msg("Error streaming command")
		}
	}()

	return stdinWriter, stdoutReader, nil
}

// Helper function to parse pod and container names from container ID
func parsePodContainerID(id string) (string, string, string) {
	parts := strings.Split(id, ":")
	return parts[0], parts[1], parts[2]
}
